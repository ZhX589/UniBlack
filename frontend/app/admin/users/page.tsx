'use client'

import { useCallback, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/app/providers'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import type { AdminUser } from '@/lib/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { ErrorState } from '@/components/ui/error-state'
import { Input } from '@/components/ui/input'
import { LoadingState } from '@/components/ui/loading-state'
import { Panel } from '@/components/ui/panel'
import { Table, Th, Td } from '@/components/ui/table'

const AVAILABLE_ROLES = ['admin', 'moderator', 'user']

export default function AdminUsersPage() {
  const { status } = useAuth()
  const router = useRouter()
  const [users, setUsers] = useState<AdminUser[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [editingUser, setEditingUser] = useState<string | null>(null)
  const [resetResult, setResetResult] = useState<Record<string, string>>({})

  const fetchUsers = useCallback(async () => {
    if (status !== 'authenticated') return
    setLoading(true)
    setError('')
    try {
      const data = await apiRequest<{ users?: AdminUser[]; total?: number }>(
        `/api/admin/users?page=${page}&page_size=20&search=${encodeURIComponent(search)}`,
        { auth: true },
      )
      setUsers(data.users || [])
      setTotal(data.total || 0)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '加载用户失败')
    } finally {
      setLoading(false)
    }
  }, [page, search, status])

  useEffect(() => {
    if (status === 'anonymous') router.replace('/login?next=/admin/users')
  }, [status, router])

  useEffect(() => {
    void fetchUsers()
  }, [fetchUsers])

  const handleToggleActive = async (userId: string, active: boolean) => {
    try {
      await apiRequest(`/api/admin/users/${userId}/active`, {
        method: 'PUT',
        auth: true,
        json: { active },
      })
      await fetchUsers()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '更新用户失败')
    }
  }

  const handleAssignRole = async (userId: string, roleName: string) => {
    try {
      await apiRequest(`/api/admin/users/${userId}/roles`, {
        method: 'POST',
        auth: true,
        json: { role_name: roleName },
      })
      await fetchUsers()
    } catch {
      setError('角色分配失败')
    }
  }

  const handleRemoveRole = async (userId: string, roleName: string) => {
    try {
      await apiRequest(`/api/admin/users/${userId}/roles/${roleName}`, {
        method: 'DELETE',
        auth: true,
      })
      await fetchUsers()
    } catch {
      setError('角色移除失败')
    }
  }

  const handleResetPassword = async (userId: string) => {
    if (!window.confirm('确定重置该用户密码？')) return
    try {
      const data = await apiRequest<{ new_password: string }>(
        `/api/admin/users/${userId}/reset-password`,
        { method: 'POST', auth: true },
      )
      const pw = data.new_password
      setResetResult((prev) => ({ ...prev, [userId]: pw }))
    } catch {
      setError('密码重置失败')
    }
  }

  if (status === 'loading') return <LoadingState />
  if (status === 'anonymous') return null

  return (
    <div className="py-4">
      <h1 className="mb-6 text-3xl font-bold">用户管理</h1>
      <Panel className="p-0">
        <div className="border-b border-border p-4">
          <label htmlFor="user-search" className="sr-only">
            搜索用户
          </label>
          <Input
            id="user-search"
            value={search}
            onChange={(e) => {
              setPage(1)
              setSearch(e.target.value)
            }}
            placeholder="搜索用户名或邮箱..."
          />
        </div>
        {error && (
          <div className="p-4">
            <ErrorState message={error} />
          </div>
        )}
        {loading ? (
          <LoadingState />
        ) : users.length === 0 ? (
          <div className="p-6">
            <EmptyState message="暂无用户" />
          </div>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <thead className="text-left">
                <tr>
                  <Th>用户名</Th>
                  <Th>邮箱</Th>
                  <Th>角色</Th>
                  <Th>状态</Th>
                  <Th>操作</Th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => (
                  <tr key={user.id} className="border-t border-border">
                    <Td>{user.username}</Td>
                    <Td>{user.email}</Td>
                    <Td>
                      <div className="flex flex-wrap items-center gap-1">
                        {(user.roles || []).map((role) => (
                          <span key={role} className="inline-flex items-center gap-1">
                            <Badge>{role}</Badge>
                            {editingUser === user.id && (
                              <button
                                type="button"
                                onClick={() => handleRemoveRole(user.id, role)}
                                className="ml-0.5 text-xs text-danger hover:underline"
                                aria-label={`移除角色 ${role}`}
                              >
                                ×
                              </button>
                            )}
                          </span>
                        ))}
                        {(user.roles || []).length === 0 && <span className="text-muted">无角色</span>}
                      </div>
                      {editingUser === user.id && (
                        <div className="mt-1 flex flex-wrap gap-1">
                          {AVAILABLE_ROLES.filter(
                            (r) => !(user.roles || []).includes(r),
                          ).map((r) => (
                            <button
                              key={r}
                              type="button"
                              onClick={() => handleAssignRole(user.id, r)}
                              className="rounded border border-border px-1.5 py-0.5 text-xs hover:bg-surface"
                            >
                              +{r}
                            </button>
                          ))}
                        </div>
                      )}
                    </Td>
                    <Td>
                      <Badge tone={user.is_active ? 'success' : 'danger'}>
                        {user.is_active ? '活跃' : '禁用'}
                      </Badge>
                    </Td>
                    <Td>
                      <div className="flex flex-wrap gap-1">
                        <Button variant="ghost" onClick={() => handleToggleActive(user.id, !user.is_active)}>
                          {user.is_active ? '禁用' : '启用'}
                        </Button>
                        <Button
                          variant="ghost"
                          onClick={() => setEditingUser(editingUser === user.id ? null : user.id)}
                        >
                          {editingUser === user.id ? '完成' : '角色'}
                        </Button>
                        <Button variant="ghost" onClick={() => handleResetPassword(user.id)}>
                          重置密码
                        </Button>
                      </div>
                      {resetResult[user.id] && (
                        <div className="mt-1 rounded bg-success/10 p-1.5 text-xs text-success">
                          新密码: {resetResult[user.id]}
                          <button
                            type="button"
                            onClick={() => {
                              const next = { ...resetResult }
                              delete next[user.id]
                              setResetResult(next)
                            }}
                            className="ml-2 text-muted hover:text-foreground"
                          >
                            清除
                          </button>
                        </div>
                      )}
                    </Td>
                  </tr>
                ))}
              </tbody>
            </Table>
          </div>
        )}
        <div className="flex items-center justify-between border-t border-border p-4 text-sm">
          <div className="text-muted">共 {total} 个用户</div>
          <div className="flex gap-2">
            <Button variant="secondary" onClick={() => setPage(Math.max(1, page - 1))} disabled={page === 1}>
              上一页
            </Button>
            <span className="px-3 py-2">{page}</span>
            <Button variant="secondary" onClick={() => setPage(page + 1)} disabled={users.length < 20}>
              下一页
            </Button>
          </div>
        </div>
      </Panel>
    </div>
  )
}

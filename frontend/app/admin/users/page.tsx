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
import { LoadingState } from '@/components/ui/loading-state'
import { Panel } from '@/components/ui/panel'

export default function AdminUsersPage() {
  const { status } = useAuth()
  const router = useRouter()
  const [users, setUsers] = useState<AdminUser[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

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
          <input
            id="user-search"
            value={search}
            onChange={(e) => {
              setPage(1)
              setSearch(e.target.value)
            }}
            placeholder="搜索用户名或邮箱..."
            className="min-h-touch w-full rounded border border-border px-4 py-2"
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
            <table className="w-full min-w-[48rem] text-sm">
              <thead className="bg-background text-left text-muted">
                <tr>
                  <th className="px-4 py-3">用户名</th>
                  <th className="px-4 py-3">邮箱</th>
                  <th className="px-4 py-3">角色</th>
                  <th className="px-4 py-3">状态</th>
                  <th className="px-4 py-3">操作</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => (
                  <tr key={user.id} className="border-t border-border">
                    <td className="px-4 py-3">{user.username}</td>
                    <td className="px-4 py-3">{user.email}</td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {(user.roles || []).map((role) => (
                          <Badge key={role}>{role}</Badge>
                        ))}
                        {(user.roles || []).length === 0 && <span className="text-muted">无角色</span>}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <Badge tone={user.is_active ? 'success' : 'danger'}>{user.is_active ? '活跃' : '禁用'}</Badge>
                    </td>
                    <td className="px-4 py-3">
                      <Button variant="ghost" onClick={() => handleToggleActive(user.id, !user.is_active)}>
                        {user.is_active ? '禁用' : '启用'}
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
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

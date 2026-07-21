'use client'

import { useCallback, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/app/providers'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import type { AccessListEntry } from '@/lib/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { ErrorState } from '@/components/ui/error-state'
import { LoadingState } from '@/components/ui/loading-state'
import { Panel } from '@/components/ui/panel'

type AccessListRow = AccessListEntry & { created_at?: string }

export default function AccessListsPage() {
  const { status } = useAuth()
  const router = useRouter()
  const [entries, setEntries] = useState<AccessListRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [type, setType] = useState('')
  const [loading, setLoading] = useState(true)
  const [showAdd, setShowAdd] = useState(false)
  const [error, setError] = useState('')
  const [newEntry, setNewEntry] = useState({ type: 'blacklist', target: 'ip', value: '', reason: '' })

  const fetchEntries = useCallback(async () => {
    if (status !== 'authenticated') return
    setLoading(true)
    setError('')
    try {
      const data = await apiRequest<{ entries?: AccessListRow[]; total?: number }>(
        `/api/admin/access-lists?page=${page}&page_size=20&type=${encodeURIComponent(type)}`,
        { auth: true },
      )
      setEntries(data.entries || [])
      setTotal(data.total || 0)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '加载名单失败')
    } finally {
      setLoading(false)
    }
  }, [page, status, type])

  useEffect(() => {
    if (status === 'anonymous') router.replace('/login?next=/admin/access-lists')
  }, [status, router])

  useEffect(() => {
    void fetchEntries()
  }, [fetchEntries])

  async function handleAdd() {
    try {
      await apiRequest('/api/admin/access-lists', { auth: true, json: newEntry })
      setShowAdd(false)
      setNewEntry({ type: 'blacklist', target: 'ip', value: '', reason: '' })
      await fetchEntries()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '添加失败')
    }
  }

  async function handleDelete(id: string) {
    if (!window.confirm('确定删除此条目？')) return
    try {
      await apiRequest(`/api/admin/access-lists/${id}`, { method: 'DELETE', auth: true })
      await fetchEntries()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '删除失败')
    }
  }

  if (status === 'loading') return <LoadingState />
  if (status === 'anonymous') return null

  return (
    <div className="py-4">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-3xl font-bold">名单管理</h1>
        <Button onClick={() => setShowAdd(true)}>添加条目</Button>
      </div>

      <div className="mb-6 flex flex-wrap gap-2">
        {[
          { value: '', label: '全部' },
          { value: 'whitelist', label: '白名单' },
          { value: 'blacklist', label: '黑名单' },
        ].map((item) => (
          <Button
            key={item.value || 'all'}
            variant={type === item.value ? 'primary' : 'secondary'}
            onClick={() => {
              setPage(1)
              setType(item.value)
            }}
          >
            {item.label}
          </Button>
        ))}
      </div>

      {error && (
        <div className="mb-4">
          <ErrorState message={error} />
        </div>
      )}

      {showAdd && (
        <Panel className="mb-6">
          <h2 className="mb-4 text-xl font-semibold">添加条目</h2>
          <div className="grid gap-4 md:grid-cols-2">
            <label className="block text-sm">
              类型
              <select
                value={newEntry.type}
                onChange={(e) => setNewEntry({ ...newEntry, type: e.target.value })}
                className="mt-1 min-h-touch w-full rounded border border-border px-4 py-2"
              >
                <option value="blacklist">黑名单</option>
                <option value="whitelist">白名单</option>
              </select>
            </label>
            <label className="block text-sm">
              目标
              <select
                value={newEntry.target}
                onChange={(e) => setNewEntry({ ...newEntry, target: e.target.value })}
                className="mt-1 min-h-touch w-full rounded border border-border px-4 py-2"
              >
                <option value="ip">IP</option>
                <option value="email">邮箱</option>
                <option value="username">用户名</option>
              </select>
            </label>
            <label className="block text-sm">
              值
              <input
                value={newEntry.value}
                onChange={(e) => setNewEntry({ ...newEntry, value: e.target.value })}
                className="mt-1 min-h-touch w-full rounded border border-border px-4 py-2"
                placeholder="IP/邮箱/用户名"
              />
            </label>
            <label className="block text-sm">
              原因
              <input
                value={newEntry.reason}
                onChange={(e) => setNewEntry({ ...newEntry, reason: e.target.value })}
                className="mt-1 min-h-touch w-full rounded border border-border px-4 py-2"
                placeholder="可选"
              />
            </label>
          </div>
          <div className="mt-4 flex gap-2">
            <Button onClick={handleAdd}>添加</Button>
            <Button variant="secondary" onClick={() => setShowAdd(false)}>
              取消
            </Button>
          </div>
        </Panel>
      )}

      <Panel className="p-0">
        {loading ? (
          <LoadingState />
        ) : entries.length === 0 ? (
          <div className="p-6">
            <EmptyState message="暂无名单记录" />
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[48rem] text-sm">
              <thead className="bg-background text-left text-muted">
                <tr>
                  <th className="px-4 py-3">类型</th>
                  <th className="px-4 py-3">目标</th>
                  <th className="px-4 py-3">值</th>
                  <th className="px-4 py-3">原因</th>
                  <th className="px-4 py-3">创建时间</th>
                  <th className="px-4 py-3">操作</th>
                </tr>
              </thead>
              <tbody>
                {entries.map((entry) => (
                  <tr key={entry.id} className="border-t border-border">
                    <td className="px-4 py-3">
                      <Badge tone={entry.type === 'whitelist' ? 'success' : 'danger'}>
                        {entry.type === 'whitelist' ? '白名单' : '黑名单'}
                      </Badge>
                    </td>
                    <td className="px-4 py-3">{entry.target}</td>
                    <td className="px-4 py-3 font-mono">{entry.value}</td>
                    <td className="px-4 py-3">{entry.reason || '-'}</td>
                    <td className="px-4 py-3 text-muted">
                      {entry.created_at ? new Date(entry.created_at).toLocaleString() : '-'}
                    </td>
                    <td className="px-4 py-3">
                      <Button variant="ghost" onClick={() => handleDelete(entry.id)}>
                        删除
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex items-center justify-between border-t border-border p-4 text-sm">
          <div className="text-muted">共 {total} 条记录</div>
          <div className="flex gap-2">
            <Button variant="secondary" onClick={() => setPage(Math.max(1, page - 1))} disabled={page === 1}>
              上一页
            </Button>
            <span className="px-3 py-2">{page}</span>
            <Button variant="secondary" onClick={() => setPage(page + 1)} disabled={entries.length < 20}>
              下一页
            </Button>
          </div>
        </div>
      </Panel>
    </div>
  )
}

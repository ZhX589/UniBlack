'use client'

import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/app/providers'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import type { Sanction } from '@/lib/types'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { ErrorState } from '@/components/ui/error-state'
import { LoadingState } from '@/components/ui/loading-state'
import { Panel } from '@/components/ui/panel'

export default function SanctionsPage() {
  const { status } = useAuth()
  const router = useRouter()
  const [form, setForm] = useState({ user_id: '', type: 'warning', reason: '', ends_at: '' })
  const [items, setItems] = useState<Sanction[]>([])
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    if (status !== 'authenticated') return
    setLoading(true)
    setError('')
    try {
      const data = await apiRequest<{ items?: Sanction[] }>('/api/admin/sanctions?page=1&page_size=50', {
        auth: true,
      })
      setItems(data.items || [])
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }, [status])

  useEffect(() => {
    if (status === 'anonymous') router.replace('/login?next=/admin/sanctions')
  }, [status, router])

  useEffect(() => {
    void load()
  }, [load])

  async function submit(e: FormEvent) {
    e.preventDefault()
    const body: Record<string, unknown> = { ...form }
    if (form.type !== 'submission_suspension') delete body.ends_at
    else if (form.ends_at) body.ends_at = new Date(form.ends_at).toISOString()
    try {
      await apiRequest('/api/admin/sanctions', { auth: true, json: body })
      setMessage('处罚已记录')
      setForm({ user_id: '', type: 'warning', reason: '', ends_at: '' })
      await load()
    } catch (err) {
      setMessage(err instanceof ApiError ? err.message : '保存失败')
    }
  }

  async function revoke(id: string) {
    const reason = window.prompt('撤销原因')
    if (!reason) return
    try {
      await apiRequest(`/api/admin/sanctions/${id}/revoke`, {
        auth: true,
        json: { reason },
      })
      setMessage('处罚已撤销')
      await load()
    } catch (err) {
      setMessage(err instanceof ApiError ? err.message : '撤销失败')
    }
  }

  if (status === 'loading') return <LoadingState />
  if (status === 'anonymous') return null

  return (
    <main className="space-y-8 py-4">
      <div>
        <h1 className="text-3xl font-bold">提交处罚</h1>
        <p className="mt-2 text-muted">仅用于已核查的恶意、伪造或重复滥用提交。所有操作写入审计记录。</p>
      </div>
      {message && <Alert tone="success">{message}</Alert>}
      {error && <ErrorState message={error} />}
      <Panel className="max-w-xl space-y-3">
        <form onSubmit={submit} className="space-y-3">
          <label className="block text-sm">
            用户 UUID
            <input
              required
              className="mt-1 min-h-touch w-full rounded border border-border p-2"
              value={form.user_id}
              onChange={(e) => setForm({ ...form, user_id: e.target.value })}
            />
          </label>
          <label className="block text-sm">
            类型
            <select
              className="mt-1 min-h-touch w-full rounded border border-border p-2"
              value={form.type}
              onChange={(e) => setForm({ ...form, type: e.target.value })}
            >
              <option value="warning">警告</option>
              <option value="submission_suspension">限期禁止提交</option>
              <option value="submission_ban">永久禁止提交</option>
            </select>
          </label>
          {form.type === 'submission_suspension' && (
            <label className="block text-sm">
              截止时间
              <input
                required
                type="datetime-local"
                className="mt-1 min-h-touch w-full rounded border border-border p-2"
                value={form.ends_at}
                onChange={(e) => setForm({ ...form, ends_at: e.target.value })}
              />
            </label>
          )}
          <label className="block text-sm">
            处罚原因
            <textarea
              required
              className="mt-1 w-full rounded border border-border p-2"
              value={form.reason}
              onChange={(e) => setForm({ ...form, reason: e.target.value })}
            />
          </label>
          <Button type="submit" variant="danger">
            记录处罚
          </Button>
        </form>
      </Panel>
      <Panel>
        <h2 className="font-semibold">最近处罚</h2>
        {loading ? (
          <LoadingState />
        ) : (
          <div className="mt-3 overflow-x-auto">
            <table className="min-w-full text-left text-sm">
              <thead>
                <tr className="border-b border-border">
                  <th className="py-2 pr-4">用户</th>
                  <th className="py-2 pr-4">类型</th>
                  <th className="py-2 pr-4">原因</th>
                  <th className="py-2 pr-4">状态</th>
                  <th className="py-2">操作</th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.id} className="border-b border-border align-top">
                    <td className="py-2 pr-4 font-mono text-xs">{item.user_id}</td>
                    <td className="py-2 pr-4">{item.type}</td>
                    <td className="py-2 pr-4">{item.reason}</td>
                    <td className="py-2 pr-4">{item.revoked_at ? '已撤销' : '生效中'}</td>
                    <td className="py-2">
                      {!item.revoked_at && (
                        <Button type="button" variant="ghost" onClick={() => revoke(item.id)}>
                          撤销
                        </Button>
                      )}
                    </td>
                  </tr>
                ))}
                {items.length === 0 && (
                  <tr>
                    <td colSpan={5} className="py-4">
                      <EmptyState message="暂无处罚记录" />
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </Panel>
    </main>
  )
}

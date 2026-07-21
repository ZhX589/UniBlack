'use client'

import { useCallback, useEffect, useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/app/providers'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import type { Appeal } from '@/lib/types'
import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { ErrorState } from '@/components/ui/error-state'
import { LoadingState } from '@/components/ui/loading-state'
import { Panel } from '@/components/ui/panel'

const OUTCOMES = [
  { value: 'upheld', label: '维持 (upheld)' },
  { value: 'corrected', label: '更正 (corrected)' },
  { value: 'withdrawn', label: '撤回 (withdrawn)' },
  { value: 'malicious_submission', label: '恶意提交 (malicious)' },
] as const

export default function AdminAppealsPage() {
  const { status, hasRole } = useAuth()
  const router = useRouter()
  const [items, setItems] = useState<Appeal[]>([])
  const [filter, setFilter] = useState('pending')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [busyID, setBusyID] = useState('')

  const load = useCallback(async () => {
    if (status !== 'authenticated') return
    setLoading(true)
    setError('')
    try {
      const qs = new URLSearchParams({ page: '1', page_size: '50' })
      if (filter) qs.set('status', filter)
      const data = await apiRequest<{ appeals?: Appeal[] }>(`/api/appeals?${qs}`, { auth: true })
      setItems(data.appeals || [])
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }, [status, filter])

  useEffect(() => {
    if (status === 'anonymous') router.replace('/login?next=/admin/appeals')
  }, [status, router])

  useEffect(() => {
    void load()
  }, [load])

  async function resolve(id: string, outcome: string) {
    const reason = window.prompt(`裁决说明（outcome=${outcome}）`)
    if (reason === null) return
    setBusyID(id)
    setMessage('')
    try {
      await apiRequest(`/api/appeals/${id}/review/resolve`, {
        auth: true,
        json: { outcome, reason: reason.trim() },
      })
      setMessage('已裁决')
      await load()
    } catch (err) {
      setMessage(err instanceof ApiError ? err.message : '裁决失败')
    } finally {
      setBusyID('')
    }
  }

  if (status === 'loading') return <LoadingState />
  if (status === 'anonymous') return null
  if (!hasRole('admin', 'moderator')) {
    return <ErrorState message="403：没有审核权限" />
  }

  return (
    <main className="space-y-6 py-4">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold">事件申诉队列</h1>
          <p className="mt-2 text-muted">审核 Event 申诉并写入 outcome。处罚申诉请走「处罚」页。</p>
        </div>
        <label className="text-sm">
          状态
          <select
            className="ml-2 min-h-touch rounded border border-border bg-surface px-2"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          >
            <option value="pending">pending</option>
            <option value="approved">approved</option>
            <option value="rejected">rejected</option>
            <option value="">全部</option>
          </select>
        </label>
      </div>

      {message && <Alert tone={message.includes('失败') ? 'danger' : 'success'}>{message}</Alert>}
      {error && <ErrorState message={error} />}

      <Panel>
        {loading ? (
          <LoadingState />
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full text-left text-sm">
              <thead>
                <tr className="border-b border-border">
                  <th className="py-2 pr-3">事件</th>
                  <th className="py-2 pr-3">状态</th>
                  <th className="py-2 pr-3">理由</th>
                  <th className="py-2 pr-3">outcome</th>
                  <th className="py-2">操作</th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.id} className="border-b border-border align-top">
                    <td className="py-3 pr-3 font-mono text-xs">
                      {item.event_id ? (
                        <Link href={`/events/${item.event_id}`} className="text-primary hover:underline">
                          {item.event_id}
                        </Link>
                      ) : item.case_id ? (
                        <span title="legacy case">{item.case_id} (case)</span>
                      ) : (
                        '—'
                      )}
                    </td>
                    <td className="py-3 pr-3">
                      <Badge tone={item.status === 'pending' ? 'warning' : item.status === 'approved' ? 'success' : 'neutral'}>
                        {item.status}
                      </Badge>
                    </td>
                    <td className="max-w-xs py-3 pr-3 whitespace-pre-wrap">{item.reason}</td>
                    <td className="py-3 pr-3 text-muted">{item.outcome || '—'}</td>
                    <td className="py-3">
                      {item.status === 'pending' && (
                        <div className="flex flex-col gap-2">
                          {OUTCOMES.map((o) => (
                            <Button
                              key={o.value}
                              type="button"
                              variant="secondary"
                              className="justify-start text-xs"
                              disabled={busyID === item.id}
                              onClick={() => resolve(item.id, o.value)}
                            >
                              {o.label}
                            </Button>
                          ))}
                        </div>
                      )}
                    </td>
                  </tr>
                ))}
                {items.length === 0 && (
                  <tr>
                    <td colSpan={5} className="py-6">
                      <EmptyState message="当前筛选下无申诉" />
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

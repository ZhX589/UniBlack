'use client'

import Link from 'next/link'
import { FormEvent, useCallback, useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { useAuth } from '@/app/providers'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import type { Appeal, EventItem } from '@/lib/types'
import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { ErrorState } from '@/components/ui/error-state'
import { LoadingState } from '@/components/ui/loading-state'
import { Panel } from '@/components/ui/panel'

export default function EventDetailPage() {
  const params = useParams()
  const id = params?.id as string
  const { status, user } = useAuth()
  const [event, setEvent] = useState<EventItem | null>(null)
  const [appeals, setAppeals] = useState<Appeal[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [reason, setReason] = useState('')
  const [message, setMessage] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const loadAppeals = useCallback(async (eventId: string, signal?: AbortSignal) => {
    if (status !== 'authenticated') {
      setAppeals([])
      return
    }
    try {
      const data = await apiRequest<Appeal[]>(`/api/events/${eventId}/appeals`, {
        auth: true,
        signal,
      })
      setAppeals(Array.isArray(data) ? data : [])
    } catch {
      // Appeal history is authenticated-only; ignore if unavailable.
      setAppeals([])
    }
  }, [status])

  useEffect(() => {
    if (!id) return
    const controller = new AbortController()
    setLoading(true)
    apiRequest<EventItem>(`/api/v1/events/${id}`, { signal: controller.signal })
      .then(async (data) => {
        setEvent(data)
        await loadAppeals(id, controller.signal)
      })
      .catch((err) => {
        if (controller.signal.aborted) return
        setError(err instanceof ApiError ? err.message : '加载失败')
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false)
      })
    return () => controller.abort()
  }, [id, loadAppeals])

  async function submitAppeal(e: FormEvent) {
    e.preventDefault()
    if (!id || !reason.trim()) return
    setSubmitting(true)
    setMessage('')
    try {
      await apiRequest(`/api/events/${id}/appeals`, {
        auth: true,
        json: { reason: reason.trim() },
      })
      setReason('')
      setMessage('申诉已提交，等待审核')
      await loadAppeals(id)
    } catch (err) {
      setMessage(err instanceof ApiError ? err.message : '申诉失败')
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) return <LoadingState />
  if (error || !event) return <ErrorState message={error || '事件不存在或未公开'} />

  const canAppeal = event.status === 'published' || event.status === 'corrected'

  return (
    <div className="mx-auto max-w-3xl space-y-6 py-8">
      <Panel>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <h1 className="text-2xl font-bold">{event.title}</h1>
          <Badge tone={event.status === 'corrected' ? 'warning' : 'neutral'}>{event.status}</Badge>
        </div>
        <p className="mt-2 font-mono text-sm text-muted">{event.id}</p>
        {event.details && <p className="mt-4 whitespace-pre-wrap text-foreground">{event.details}</p>}
        <p className="mt-4 text-sm text-muted">严重程度 {event.severity ?? 1}/5</p>
        {event.evidence && event.evidence.length > 0 && (
          <div className="mt-6">
            <h2 className="mb-2 font-semibold">证据</h2>
            <ul className="space-y-2 text-sm">
              {event.evidence.map((item, index) => (
                <li key={item.id || index} className="rounded border border-border p-3">
                  <p className="font-medium">
                    {item.type}
                    {item.title ? ` · ${item.title}` : ''}
                  </p>
                  {item.description && <p className="mt-1 text-muted">{item.description}</p>}
                  {item.url && (
                    <a href={item.url} className="mt-1 inline-block text-primary hover:underline" target="_blank" rel="noreferrer">
                      打开链接
                    </a>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )}
        <div className="mt-6 flex flex-wrap gap-4">
          <Link href="/search" className="text-primary hover:underline">
            返回查询
          </Link>
          {event.id && (
            <Link href={`/subjects`} className="text-muted hover:underline">
              浏览名单
            </Link>
          )}
        </div>
      </Panel>

      <Panel>
        <h2 className="text-xl font-semibold">事件申诉</h2>
        <p className="mt-2 text-sm text-muted">
          对已公开事件可提出更正或撤回申诉。需登录；管理方将记录 outcome（upheld / corrected / withdrawn / malicious_submission）。
        </p>
        {message && (
          <div className="mt-4">
            <Alert tone={message.includes('失败') ? 'danger' : 'success'}>{message}</Alert>
          </div>
        )}
        {status === 'anonymous' && (
          <p className="mt-4 text-sm">
            <Link href={`/login?next=/events/${id}`} className="text-primary hover:underline">
              登录后提交申诉
            </Link>
          </p>
        )}
        {status === 'authenticated' && canAppeal && (
          <form onSubmit={submitAppeal} className="mt-4 space-y-3">
            <label className="block text-sm">
              申诉理由
              <textarea
                required
                className="mt-1 min-h-[6rem] w-full rounded border border-border p-2"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="说明事实错误、证据问题或请求撤回的原因"
              />
            </label>
            <p className="text-xs text-muted">当前用户：{user?.username || user?.email}</p>
            <Button type="submit" disabled={submitting}>
              {submitting ? '提交中…' : '提交事件申诉'}
            </Button>
          </form>
        )}
        {status === 'authenticated' && !canAppeal && (
          <p className="mt-4 text-sm text-muted">仅 published / corrected 状态的事件可申诉。</p>
        )}

        <div className="mt-6">
          <h3 className="font-semibold">申诉记录</h3>
          {status !== 'authenticated' ? (
            <p className="mt-2 text-sm text-muted">登录后可查看该事件的申诉历史。</p>
          ) : appeals.length === 0 ? (
            <EmptyState message="暂无申诉" />
          ) : (
            <ul className="mt-3 space-y-2">
              {appeals.map((item) => (
                <li key={item.id} className="rounded border border-border p-3 text-sm">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <Badge tone={item.status === 'pending' ? 'warning' : item.status === 'approved' ? 'success' : 'neutral'}>
                      {item.status}
                    </Badge>
                    {item.outcome && <span className="text-muted">outcome: {item.outcome}</span>}
                  </div>
                  <p className="mt-2 whitespace-pre-wrap">{item.reason}</p>
                  {item.resolution_reason && <p className="mt-1 text-muted">裁决：{item.resolution_reason}</p>}
                </li>
              ))}
            </ul>
          )}
        </div>
      </Panel>
    </div>
  )
}

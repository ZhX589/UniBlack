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
import { Textarea } from '@/components/ui/input'
import { EmptyState } from '@/components/ui/empty-state'
import { ErrorState } from '@/components/ui/error-state'
import { LoadingState } from '@/components/ui/loading-state'
import {
  appealOutcomeLabel,
  appealStatusPresentation,
  evidenceAction,
  evidenceTypeLabel,
  eventStatusPresentation,
  severityPresentation,
} from '@/lib/trustworthy-profile'

type Feedback = { tone: 'success' | 'danger'; message: string } | null

export default function EventDetailPage() {
  const params = useParams()
  const id = params?.id as string
  const { status, user } = useAuth()
  const [event, setEvent] = useState<EventItem | null>(null)
  const [appeals, setAppeals] = useState<Appeal[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [reason, setReason] = useState('')
  // Feedback is the single source of truth for tone — never inferred from
  // message wording. Message text alone is too brittle to decide severity.
  const [feedback, setFeedback] = useState<Feedback>(null)
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
    setFeedback(null)
    try {
      await apiRequest(`/api/events/${id}/appeals`, {
        auth: true,
        json: { reason: reason.trim() },
      })
      // Success: clear user input and refresh appeal history.
      setReason('')
      setFeedback({ tone: 'success', message: '申诉已提交，等待审核' })
      await loadAppeals(id)
    } catch (err) {
      // Failure: preserve `reason` so the user does not lose their drafting.
      setFeedback({
        tone: 'danger',
        message: err instanceof ApiError ? err.message : '申诉提交失败，请稍后重试',
      })
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) return <LoadingState />
  if (error || !event) return <ErrorState message={error || '事件不存在或未公开'} />

  const eventStatus = eventStatusPresentation(event.status)
  const severity = severityPresentation(event.severity)
  const evidence = event.evidence || []
  const canAppeal = event.status === 'published' || event.status === 'corrected'

  return (
    <main className="mx-auto max-w-4xl space-y-8 py-8">
      {/* Layer 1: event header */}
      <header className="border-b border-border pb-6">
        <h1 className="text-3xl font-bold">{event.title}</h1>
        <div className="mt-3 flex flex-wrap items-center gap-3">
          <Badge tone={eventStatus.tone}>{eventStatus.label}</Badge>
          <span className="text-sm text-muted">
            严重程度 {severity.label} {severity.value}/5
          </span>
        </div>
        <p className="mt-2 font-mono text-sm text-muted">事件 ID：{event.id}</p>
      </header>

      {/* Layer 2: facts */}
      <section aria-labelledby="event-facts-heading">
        <h2 id="event-facts-heading" className="text-xl font-semibold">
          事实描述
        </h2>
        <p className="mt-2 text-xs text-muted">
          以下内容为社区公开记录，请结合证据审阅。
        </p>
        {event.details && event.details.trim() ? (
          <p className="mt-3 whitespace-pre-wrap text-foreground">{event.details}</p>
        ) : (
          <EmptyState message="当前公开记录中没有补充事实描述。" />
        )}
      </section>

      {/* Layer 3: evidence */}
      <section aria-labelledby="event-evidence-heading">
        <h2 id="event-evidence-heading" className="text-xl font-semibold">
          证据材料
        </h2>
        {evidence.length === 0 ? (
          <EmptyState message="当前公开响应中没有可展示的证据材料。" />
        ) : (
          <ul className="mt-4 space-y-3">
            {evidence.map((item, index) => {
              const typeLabel = evidenceTypeLabel(item.type)
              const title = item.title?.trim() || item.original_filename?.trim() || typeLabel
              const action = evidenceAction(item)
              const sizeLabel = item.file_size != null ? formatFileSize(item.file_size) : null
              return (
                <li
                  key={item.id || index}
                  className="rounded border border-border bg-white/40 px-4 py-4 glass"
                >
                  <p className="font-mono text-xs text-muted">{typeLabel}</p>
                  <p className="mt-1 font-medium text-foreground">{title}</p>
                  {item.description && (
                    <p className="mt-1 text-sm text-muted">{item.description}</p>
                  )}
                  {(item.original_filename || item.mime_type || sizeLabel) && (
                    <p className="mt-1 text-xs text-muted">
                      {[
                        item.original_filename?.trim(),
                        item.mime_type,
                        sizeLabel,
                      ]
                        .filter(Boolean)
                        .join(' · ')}
                    </p>
                  )}
                  {action && (
                    <a
                      href={action.href}
                      target="_blank"
                      rel="noreferrer"
                      className="mt-2 inline-block text-sm text-primary hover:underline"
                    >
                      {action.label}
                    </a>
                  )}
                  {/* File evidence without a real URL renders no action link. */}
                </li>
              )
            })}
          </ul>
        )}
      </section>

      {/* Layer 4: appeals and correction */}
      <section aria-labelledby="event-appeal-heading">
        <h2 id="event-appeal-heading" className="text-xl font-semibold">
          申诉与纠错
        </h2>
        <p className="mt-2 text-sm text-muted">
          对已公开或已更正事件可提出更正或撤回申诉。需登录提交；管理方将记录裁决结果。
        </p>

        {feedback && (
          <div className="mt-3">
            <Alert tone={feedback.tone}>{feedback.message}</Alert>
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
              <Textarea
                required
                className="min-h-[6rem]"
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
          <p className="mt-4 text-sm text-muted">仅已公开 / 已更正状态的事件可申诉。</p>
        )}

        <div className="mt-6">
          <h3 className="font-semibold">申诉记录</h3>
          {status !== 'authenticated' ? (
            <p className="mt-2 text-sm text-muted">登录后可查看该事件的申诉历史。</p>
          ) : appeals.length === 0 ? (
            <EmptyState message="当前事件暂无申诉记录。" />
          ) : (
            <ul className="mt-3 space-y-2">
              {appeals.map((item) => {
                const appealStatus = appealStatusPresentation(item.status)
                const outcomeLabel = appealOutcomeLabel(item.outcome, true)
                return (
                  <li
                    key={item.id}
                    className="rounded border border-border bg-white/40 p-3 text-sm glass"
                  >
                    <div className="flex flex-wrap items-center gap-3">
                      <Badge tone={appealStatus.tone}>{appealStatus.label}</Badge>
                      {outcomeLabel && (
                        <span className="text-muted">裁决：{outcomeLabel}</span>
                      )}
                    </div>
                    <p className="mt-2 whitespace-pre-wrap">{item.reason}</p>
                    {item.resolution_reason && (
                      <p className="mt-1 text-muted">说明：{item.resolution_reason}</p>
                    )}
                  </li>
                )
              })}
            </ul>
          )}
        </div>
      </section>
    </main>
  )
}

function formatFileSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return ''
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

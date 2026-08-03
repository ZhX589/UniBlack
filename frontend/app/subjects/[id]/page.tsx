'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import type { Account, EventItem, Subject } from '@/lib/types'
import { Badge } from '@/components/ui/badge'
import { LoadingState } from '@/components/ui/loading-state'
import { ErrorState } from '@/components/ui/error-state'
import { EmptyState } from '@/components/ui/empty-state'
import {
  accountDisplayValue,
  eventStatusPresentation,
  riskPresentation,
  severityPresentation,
  subjectStatusPresentation,
} from '@/lib/trustworthy-profile'

export default function SubjectDetailPage() {
  const params = useParams()
  const id = params?.id as string
  const [subject, setSubject] = useState<Subject | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!id) return
    const controller = new AbortController()
    setLoading(true)
    setError('')
    apiRequest<Subject>(`/api/v1/subjects/${id}`, { signal: controller.signal })
      .then((data) => setSubject(data))
      .catch((err) => {
        if (controller.signal.aborted) return
        setError(err instanceof ApiError ? err.message : '加载失败')
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false)
      })
    return () => controller.abort()
  }, [id])

  if (loading) return <LoadingState />
  if (error || !subject) {
    return (
      <div className="space-y-4 py-8">
        <ErrorState message={error || '未找到该对象'} />
        <Link href="/subjects" className="text-primary hover:underline">
          ← 返回列表
        </Link>
      </div>
    )
  }

  const events: EventItem[] = subject.events || []
  const statusPresentation = subjectStatusPresentation(subject.status)
  const risk = riskPresentation(subject.risk_level)

  // Accounts whose display value resolves to null are filtered out — the
  // backend may return empty rows; do not present blank identity slots.
  const visibleAccounts: Account[] = (subject.accounts || []).filter((account) =>
    Boolean(accountDisplayValue(account)),
  )
  const visibleIdentifiers = (subject.identifiers || []).filter(
    (identifier) => identifier.value && identifier.value.trim(),
  )

  return (
    <main className="mx-auto max-w-4xl py-8">
      <Link href="/subjects" className="text-sm text-primary hover:underline">
        ← 返回列表
      </Link>

      <header className="mt-4 border-b border-border pb-6">
        <p className="text-sm font-mono text-muted">公开对象档案</p>
        <h1 className="mt-2 text-3xl font-bold">对象档案</h1>
        <div className="mt-3 flex flex-wrap items-center gap-3">
          <span className="text-xl font-semibold text-foreground">{subject.display_name}</span>
          <Badge tone={statusPresentation.tone}>{statusPresentation.label}</Badge>
        </div>
        <p className="mt-2 font-mono text-sm text-muted">{subject.public_id || subject.id}</p>
        <p className="mt-2 text-sm text-muted">
          风险等级：{risk.label} {risk.value}/5
        </p>
        <p className="mt-1 text-sm text-muted">
          这是一份社区维护的黑名单记录。记录状态、风险等级与事件轨迹以下方公开信息为准。
        </p>
      </header>

      <section aria-labelledby="subject-accounts-heading" className="mt-8">
        <h2 id="subject-accounts-heading" className="text-xl font-semibold">
          关联账号
        </h2>
        {visibleAccounts.length === 0 && visibleIdentifiers.length === 0 ? (
          <EmptyState message="当前没有可公开展示的关联账号。" />
        ) : (
          <ul className="mt-4 space-y-2">
            {visibleAccounts.map((account, index) => {
              const value = accountDisplayValue(account)
              if (!value) return null
              const isPrimary = index === 0
              return (
                <li
                  key={`a-${account.id || index}`}
                  className="flex flex-wrap items-center gap-3 rounded border border-border bg-white/40 px-4 py-3 glass"
                >
                  <span className="font-mono text-sm text-muted">{account.platform}</span>
                  <span className="text-sm text-foreground">{value}</span>
                  {isPrimary && <Badge tone="neutral">主标识</Badge>}
                  {account.account_type && (
                    <span className="text-xs text-muted">（{account.account_type}）</span>
                  )}
                </li>
              )
            })}
            {visibleIdentifiers.map((identifier, index) => (
              <li
                key={`i-${index}`}
                className="flex flex-wrap items-center gap-3 rounded border border-dashed border-border bg-white/20 px-4 py-3 glass"
              >
                <span className="font-mono text-sm text-muted">{identifier.platform}</span>
                <span className="text-sm text-foreground">{identifier.value}</span>
                <Badge tone="neutral">兼容账号记录</Badge>
                {identifier.account_type && (
                  <span className="text-xs text-muted">（{identifier.account_type}）</span>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>

      <section aria-labelledby="subject-events-heading" className="mt-10">
        <h2 id="subject-events-heading" className="text-xl font-semibold">
          公开事件轨迹
        </h2>
        {events.length === 0 ? (
          <EmptyState message="当前没有可公开展示的事件记录。这不代表对该对象作出绝对安全判断。" />
        ) : (
          <ol className="mt-4 space-y-4 border-l border-border pl-4">
            {events.map((event) => {
              const eventStatus = eventStatusPresentation(event.status)
              const severity = severityPresentation(event.severity)
              return (
                <li
                  key={event.id}
                  className="rounded border border-border bg-white/40 px-4 py-4 glass"
                >
                  <div className="flex flex-wrap items-center gap-3">
                    <Link
                      href={`/events/${event.id}`}
                      className="text-lg font-semibold text-primary hover:underline"
                    >
                      {event.title}
                    </Link>
                    <Badge tone={eventStatus.tone}>{eventStatus.label}</Badge>
                  </div>
                  <p className="mt-2 text-sm text-muted">
                    严重程度 {severity.label} {severity.value}/5
                  </p>
                  {event.details && (
                    <p className="mt-2 line-clamp-2 text-sm text-foreground">{event.details}</p>
                  )}
                  <Link
                    href={`/events/${event.id}`}
                    className="mt-3 inline-block text-sm text-primary hover:underline"
                  >
                    查看事件与证据
                  </Link>
                </li>
              )
            })}
          </ol>
        )}
      </section>
    </main>
  )
}

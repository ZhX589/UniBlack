'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import type { EventItem, Subject } from '@/lib/types'
import { Badge } from '@/components/ui/badge'
import { Panel } from '@/components/ui/panel'
import { LoadingState } from '@/components/ui/loading-state'
import { ErrorState } from '@/components/ui/error-state'
import { EmptyState } from '@/components/ui/empty-state'

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
          返回列表
        </Link>
      </div>
    )
  }

  const events: EventItem[] = subject.events || []

  return (
    <div className="mx-auto max-w-3xl py-8">
      <Link href="/subjects" className="text-sm text-primary hover:underline">
        ← 返回列表
      </Link>
      <Panel className="mt-4">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold">{subject.display_name}</h1>
            <p className="mt-1 font-mono text-sm text-muted">{subject.public_id || subject.id}</p>
          </div>
          <Badge tone={subject.status === 'active' ? 'danger' : subject.status === 'cleared' ? 'success' : 'neutral'}>
            {subject.status}
          </Badge>
        </div>
        <p className="mt-4 text-sm text-muted">风险等级：{subject.risk_level ?? 0}/5</p>
        {(subject.accounts?.length || subject.identifiers?.length) ? (
          <div className="mt-6">
            <h2 className="mb-2 font-semibold">关联账号</h2>
            <div className="flex flex-wrap gap-2">
              {(subject.accounts || []).map((account, index) => (
                <span key={`a-${index}`} className="rounded bg-background px-2 py-1 text-sm">
                  {account.platform}: {account.username || account.account_id}
                </span>
              ))}
              {(subject.identifiers || []).map((identifier, index) => (
                <span key={`i-${index}`} className="rounded bg-background px-2 py-1 text-sm">
                  {identifier.platform}: {identifier.value}
                </span>
              ))}
            </div>
          </div>
        ) : null}
      </Panel>

      <Panel className="mt-8">
        <h2 className="mb-4 text-xl font-semibold">公开事件时间线</h2>
        {events.length === 0 ? (
          <EmptyState message="暂无已公开事件" />
        ) : (
          <ul className="space-y-3">
            {events.map((event) => (
              <li key={event.id} className="rounded border border-border p-4">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <Link href={`/events/${event.id}`} className="font-medium text-primary hover:underline">
                    {event.title}
                  </Link>
                  <Badge tone={event.status === 'corrected' ? 'warning' : 'neutral'}>{event.status}</Badge>
                </div>
                {event.details && <p className="mt-1 line-clamp-2 text-sm text-muted">{event.details}</p>}
                <p className="mt-2 text-xs text-muted">严重程度 {event.severity ?? 1}/5</p>
              </li>
            ))}
          </ul>
        )}
      </Panel>
    </div>
  )
}

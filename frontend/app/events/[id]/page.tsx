'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import type { EventItem } from '@/lib/types'
import { Badge } from '@/components/ui/badge'
import { Panel } from '@/components/ui/panel'
import { LoadingState } from '@/components/ui/loading-state'
import { ErrorState } from '@/components/ui/error-state'

export default function EventDetailPage() {
  const params = useParams()
  const id = params?.id as string
  const [event, setEvent] = useState<EventItem | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!id) return
    const controller = new AbortController()
    setLoading(true)
    apiRequest<EventItem>(`/api/v1/events/${id}`, { signal: controller.signal })
      .then((data) => setEvent(data))
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
  if (error || !event) return <ErrorState message={error || '事件不存在或未公开'} />

  return (
    <div className="mx-auto max-w-3xl py-8">
      <Panel>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <h1 className="text-2xl font-bold">{event.title}</h1>
          <Badge tone={event.status === 'corrected' ? 'warning' : 'neutral'}>{event.status}</Badge>
        </div>
        <p className="mt-2 font-mono text-sm text-muted">{event.id}</p>
        {event.details && <p className="mt-4 whitespace-pre-wrap text-foreground">{event.details}</p>}
        <p className="mt-4 text-sm text-muted">严重程度 {event.severity ?? 1}/5</p>
        <div className="mt-6">
          <Link href="/search" className="text-primary hover:underline">
            返回查询
          </Link>
        </div>
      </Panel>
    </div>
  )
}

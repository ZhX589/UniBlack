'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/app/providers'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import type { Appeal } from '@/lib/types'
import { appealStatusPresentation, appealOutcomeLabel } from '@/lib/trustworthy-profile'
import { appealTarget } from './presentation'
import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { ErrorState } from '@/components/ui/error-state'
import { Select } from '@/components/ui/input'
import { LoadingState } from '@/components/ui/loading-state'
import { Panel } from '@/components/ui/panel'

const OUTCOMES = [
  { value: 'upheld', label: '维持（upheld）' },
  { value: 'corrected', label: '更正记录（corrected）' },
  { value: 'withdrawn', label: '撤回（withdrawn）' },
  { value: 'malicious_submission', label: '恶意提交（malicious_submission）' },
] as const

const FILTERS = [
  { value: 'pending', label: '待处理（pending）' },
  { value: 'approved', label: '已通过（approved）' },
  { value: 'rejected', label: '已驳回（rejected）' },
  { value: '', label: '全部状态' },
] as const

// List response shape from GET /api/appeals. The page only reads `appeals` for
// rendering but tracks `total` so the pending count is derived deterministically
// from the server response instead of locally mutating after an action.
type AppealListResponse = {
  appeals?: Appeal[]
  total?: number
  page?: number
  page_size?: number
}

type Feedback = { tone: 'success' | 'danger'; message: string } | null

export default function AdminAppealsPage() {
  const { status, hasRole } = useAuth()
  const router = useRouter()
  const [items, setItems] = useState<Appeal[]>([])
  const [total, setTotal] = useState<number | null>(null)
  // `null` means "pending count not yet known"; the header renders `待处理 — 项`
  // until the probe/list response fills it in, avoiding a misleading zero.
  const [pendingTotal, setPendingTotal] = useState<number | null>(null)
  // Default to the all-status view so the moderator can scan the full queue at
  // a glance; the pending count in the header always comes from a dedicated
  // `status=pending&page_size=1` probe so it is authoritative regardless of
  // which filter is active.
  const [filter, setFilter] = useState('')
  const [loadingItems, setLoadingItems] = useState(true)
  const [loadingPending, setLoadingPending] = useState(false)
  const [error, setError] = useState('')
  const [pendingError, setPendingError] = useState('')
  const [feedback, setFeedback] = useState<Feedback>(null)
  const [busyID, setBusyID] = useState('')
  const requestRef = useRef<{ generation: number; controller?: AbortController }>({ generation: 0 })
  const cancelRequests = useCallback(() => {
    requestRef.current.generation += 1
    requestRef.current.controller?.abort()
  }, [])

  const refresh = useCallback(async () => {
    if (status !== 'authenticated') return
    cancelRequests()
    const controller = new AbortController()
    const generation = requestRef.current.generation
    requestRef.current.controller = controller
    const current = () => !controller.signal.aborted && requestRef.current.generation === generation
    setBusyID('')
    setLoadingItems(true)
    setLoadingPending(true)
    setItems([])
    setTotal(null)
    setPendingTotal(null)
    setError('')
    setPendingError('')
    const qs = new URLSearchParams({ page: '1', page_size: '50' })
    if (filter) qs.set('status', filter)
    await Promise.all([
      apiRequest<AppealListResponse>(`/api/appeals?${qs}`, { auth: true, signal: controller.signal })
        .then((data) => {
          if (!current()) return
          setItems(data.appeals || [])
          setTotal(data.total ?? (data.appeals || []).length)
        })
        .catch((err) => {
          if (current()) setError(err instanceof ApiError ? err.message : '申诉列表加载失败')
        })
        .finally(() => { if (current()) setLoadingItems(false) }),
      apiRequest<AppealListResponse>('/api/appeals?page=1&page_size=1&status=pending', { auth: true, signal: controller.signal })
        .then((data) => {
          if (current()) setPendingTotal(data.total ?? 0)
        })
        .catch((err) => {
          if (current()) setPendingError(err instanceof ApiError ? err.message : '待处理计数加载失败')
        })
        .finally(() => { if (current()) setLoadingPending(false) }),
    ])
  }, [status, filter, cancelRequests])

  useEffect(() => {
    if (status === 'anonymous') router.replace('/login?next=/admin/appeals')
  }, [status, router])

  useEffect(() => {
    if (status !== 'authenticated') return
    void refresh()
    return cancelRequests
  }, [status, refresh, cancelRequests])

  async function resolve(id: string, outcome: string) {
    const reason = window.prompt(`裁决说明（outcome=${outcome}）`)
    if (reason === null) return
    const generation = requestRef.current.generation
    setBusyID(id)
    setFeedback(null)
    try {
      await apiRequest(`/api/appeals/${id}/review/resolve`, {
        auth: true,
        json: { outcome, reason: reason.trim() },
      })
      if (requestRef.current.generation !== generation) return
      setFeedback({ tone: 'success', message: '已裁决' })
      await refresh()
    } catch (err) {
      if (requestRef.current.generation !== generation) return
      setFeedback({
        tone: 'danger',
        message: err instanceof ApiError ? err.message : '裁决失败',
      })
    } finally {
      if (requestRef.current.generation === generation) setBusyID('')
    }
  }

  if (status === 'loading') return <LoadingState />
  if (status === 'anonymous') return null
  if (!hasRole('admin', 'moderator')) {
    return <ErrorState message="403：没有审核权限" />
  }

  const refreshing = loadingItems || loadingPending

  return (
    <main className="space-y-6 py-4">
      <div>
        <h1 className="text-3xl font-bold">事件申诉工作台</h1>
        <p className="mt-2 text-muted">逐项核对申诉理由和关联事件，不进行批量裁决。</p>
        <p className="mt-1 text-sm text-muted">
          {pendingTotal === null ? '待处理 — 项' : `待处理 ${pendingTotal} 项`}
        </p>
      </div>

      {/* Compact tooling area: filter + refresh. Wraps naturally on mobile. */}
      <div className="flex flex-wrap items-end gap-3">
        <label className="text-sm">
          <span className="sr-only">状态</span>
          <Select
            className="w-auto min-w-[12rem]"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          >
            {FILTERS.map((f) => (
              <option key={f.value} value={f.value}>
                {f.label}
              </option>
            ))}
          </Select>
        </label>
        <Button
          type="button"
          variant="secondary"
          onClick={() => void refresh()}
          disabled={refreshing}
        >
          {refreshing ? '刷新中…' : '刷新队列'}
        </Button>
      </div>

      {feedback && (
        <Alert tone={feedback.tone === 'success' ? 'success' : 'danger'}>{feedback.message}</Alert>
      )}
      {pendingError && <ErrorState message={`待处理计数：${pendingError}`} />}

      <Panel>
        {loadingItems ? (
          <LoadingState />
        ) : error ? (
          <ErrorState message={error} />
        ) : (
          <>
            {total !== null && items.length === 0 ? (
              <EmptyState message="当前筛选下无申诉" />
            ) : (
              <ul className="space-y-4">
                {items.map((item) => {
                  const target = appealTarget(item)
                  const statusPres = appealStatusPresentation(item.status)
                  const outcomeLabel = appealOutcomeLabel(item.outcome, true)
                  return (
                    <li
                      key={item.id}
                      className="space-y-2 rounded border border-border/50 p-3"
                    >
                      {/* 1. Event ID link / legacy Case label / unavailable. */}
                      <div className="flex flex-wrap items-center gap-2">
                        {target.kind === 'event' ? (
                          <Link
                            href={target.href}
                            className="font-mono text-sm text-primary hover:underline"
                          >
                            {target.id}
                          </Link>
                        ) : target.kind === 'legacy-case' ? (
                          <span className="font-mono text-sm text-text-secondary">
                            {target.id}{' '}
                            <span className="ml-1 text-xs">（兼容事项 legacy case）</span>
                          </span>
                        ) : (
                          <span className="text-sm text-text-secondary">
                            未关联事件/事项
                          </span>
                        )}
                        {/* 2. Translated appeal status with raw enum in auditLabel. */}
                        <Badge tone={statusPres.tone}>{statusPres.auditLabel}</Badge>
                      </div>

                      {/* 3. Full reason preserved with whitespace-pre-wrap. */}
                      <div>
                        <p className="text-xs text-muted">申诉理由</p>
                        <p className="whitespace-pre-wrap break-words text-sm">{item.reason}</p>
                      </div>

                      {/* 4. Translated outcome with raw enum + resolution reason. */}
                      {outcomeLabel && (
                        <div>
                          <p className="text-xs text-muted">已有裁决</p>
                          <p className="text-sm">{outcomeLabel}</p>
                          {item.resolution_reason && (
                            <p className="mt-1 whitespace-pre-wrap break-words text-sm text-text-secondary">
                              {item.resolution_reason}
                            </p>
                          )}
                        </div>
                      )}

                      {/* 5. Four single-item actions only when pending. */}
                      {item.status === 'pending' && (
                        <div className="flex flex-wrap gap-2 pt-1">
                          {OUTCOMES.map((o) => (
                            <Button
                              key={o.value}
                              type="button"
                              variant={o.value === 'malicious_submission' ? 'danger' : 'secondary'}
                              className="justify-start text-xs"
                              disabled={busyID === item.id}
                              onClick={() => resolve(item.id, o.value)}
                            >
                              {o.label}
                            </Button>
                          ))}
                        </div>
                      )}
                    </li>
                  )
                })}
              </ul>
            )}
          </>
        )}
      </Panel>
    </main>
  )
}

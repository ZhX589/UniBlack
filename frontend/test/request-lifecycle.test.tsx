import { act, cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SearchPage from '@/app/search/page'
import AdminAppealsPage from '@/app/admin/appeals/page'
import EventDetailPage from '@/app/events/[id]/page'
import { apiRequest, type ApiRequestOptions } from '@/lib/api'
import { ApiError } from '@/lib/api-error'

const route = vi.hoisted(() => ({ q: 'first', id: 'one', status: 'authenticated' }))
vi.mock('next/navigation', () => ({
  useSearchParams: () => new URLSearchParams({ q: route.q }),
  useParams: () => ({ id: route.id }),
  useRouter: () => ({ replace: vi.fn() }),
}))
vi.mock('@/app/providers', () => ({ useAuth: () => ({ status: route.status, user: { username: 'tester' }, hasRole: () => true }) }))
vi.mock('@/lib/api', () => ({ apiRequest: vi.fn() }))

type Pending = { path: string; signal?: AbortSignal; resolve: (data: unknown) => void; reject: (error: unknown) => void }
let requests: Pending[]
async function finish(request: Pending, data: unknown, reject = false) {
  await act(async () => { if (reject) request.reject(data); else request.resolve(data) })
}
const event = (id: string) => ({ id, title: `event-${id}`, status: 'published', severity: 1 })
const subject = (name: string) => ({ id: name, display_name: name, status: 'active' })
beforeEach(() => {
  route.q = 'first'; route.id = 'one'; route.status = 'authenticated'; requests = []
  vi.mocked(apiRequest).mockImplementation((path: string, options?: ApiRequestOptions) => new Promise((resolve, reject) => {
    requests.push({ path, signal: options?.signal, resolve, reject })
  }))
})
afterEach(() => { cleanup(); vi.clearAllMocks() })

describe('search request ownership', () => {
  it('aborts old URL queries and ignores late success, error and cancellation', async () => {
    const view = render(<SearchPage />)
    route.q = 'second'; view.rerender(<SearchPage />)
    expect(requests[0].signal?.aborted).toBe(true)
    await finish(requests[1], { results: [subject('new-result')] })
    await finish(requests[0], { results: [subject('old-result')] })
    expect(screen.getByText('new-result')).toBeInTheDocument()
    expect(screen.queryByText('old-result')).not.toBeInTheDocument()
    route.q = 'third'; view.rerender(<SearchPage />)
    route.q = 'fourth'; view.rerender(<SearchPage />)
    await finish(requests[2], new DOMException('aborted', 'AbortError'), true)
    expect(screen.getByText('正在查询...')).toBeInTheDocument()
    await finish(requests[3], { results: [] })
    expect(screen.getByText('未找到相关结果')).toBeInTheDocument()
    view.unmount(); expect(requests[3].signal?.aborted).toBe(true)
  })
  it('clears and cancels query when URL becomes empty; lookup shares request ownership', async () => {
    const view = render(<SearchPage />)
    fireEvent.change(screen.getByLabelText('搜索关键词'), { target: { value: 'lookup' } })
    fireEvent.submit(screen.getByRole('button', { name: '精确查询' }).closest('form')!)
    expect(requests[0].signal?.aborted).toBe(true)
    await finish(requests[1], subject('lookup-result'))
    await finish(requests[0], new ApiError(500, 'stale-error'), true)
    expect(screen.getByText('lookup-result')).toBeInTheDocument()
    route.q = ''; view.rerender(<SearchPage />)
    expect(requests[1].signal?.aborted).toBe(true)
    expect(screen.getByText('输入关键词后开始查询')).toBeInTheDocument()
  })
})

describe('appeal queue independent resources', () => {
  it('keeps list failure when count succeeds, with no misleading empty list', async () => {
    render(<AdminAppealsPage />)
    await finish(requests[0], new ApiError(500, 'list-failed'), true)
    await finish(requests[1], { total: 4 })
    expect(screen.getByText('list-failed')).toBeInTheDocument()
    expect(screen.getByText('待处理 4 项')).toBeInTheDocument()
    expect(screen.queryByText('当前筛选下无申诉')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '刷新队列' }))
    expect(requests[0].signal?.aborted).toBe(true)
    await finish(requests[2], { appeals: [], total: 0 })
    await finish(requests[3], new ApiError(500, 'count-failed'), true)
    expect(screen.getByText('当前筛选下无申诉')).toBeInTheDocument()
    expect(screen.getByText('待处理 — 项')).toBeInTheDocument()
    expect(screen.getByText('待处理计数：count-failed')).toBeInTheDocument()
  })
  it('aborts actual list/count signals on filter changes and ignores old settlements', async () => {
    const view = render(<AdminAppealsPage />)
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'pending' } })
    expect(requests[0].signal?.aborted).toBe(true)
    expect(requests[1].signal?.aborted).toBe(true)
    await finish(requests[0], new DOMException('cancelled', 'AbortError'), true)
    await finish(requests[1], { total: 99 })
    expect(screen.getByText('刷新中…')).toBeDisabled()
    await finish(requests[2], { appeals: [{ id: 'a', reason: 'new-appeal', status: 'approved' }], total: 1 })
    await finish(requests[3], { total: 1 })
    expect(screen.getByText('new-appeal')).toBeInTheDocument()
    expect(screen.getByText('待处理 1 项')).toBeInTheDocument()
    view.unmount(); expect(requests[2].signal?.aborted).toBe(true)
  })
})

describe('event detail and history', () => {
  it('shows history failure rather than empty while detail is usable and recovers on retry', async () => {
    render(<EventDetailPage />)
    await finish(requests[0], event('one'))
    expect(screen.getByText('event-one')).toBeInTheDocument()
    expect(screen.getByText('正在加载申诉历史…')).toBeInTheDocument()
    await finish(requests[1], new ApiError(403, 'history-failed'), true)
    expect(screen.getByText('history-failed')).toBeInTheDocument()
    expect(screen.queryByText('当前事件暂无申诉记录。')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '重试申诉历史' }))
    expect(requests[1].signal?.aborted).toBe(true)
    await finish(requests[2], [])
    expect(screen.queryByText('history-failed')).not.toBeInTheDocument()
    expect(screen.getByText('当前事件暂无申诉记录。')).toBeInTheDocument()
  })
  it('clears detail error on navigation and ignores superseded detail/history and finally', async () => {
    const view = render(<EventDetailPage />)
    await finish(requests[0], new ApiError(500, 'detail-failed'), true)
    route.id = 'two'; view.rerender(<EventDetailPage />)
    expect(requests[0].signal?.aborted).toBe(true)
    expect(requests[1].signal?.aborted).toBe(true)
    await finish(requests[1], new ApiError(500, 'old-history-error'), true)
    await finish(requests[2], event('two'))
    await finish(requests[3], [])
    expect(screen.getByText('event-two')).toBeInTheDocument()
    expect(screen.queryByText('detail-failed')).not.toBeInTheDocument()
    route.id = 'three'; view.rerender(<EventDetailPage />)
    route.id = 'four'; view.rerender(<EventDetailPage />)
    await finish(requests[4], event('three'))
    await finish(requests[5], [{ id: 'old', status: 'pending', reason: 'stale-history' }])
    expect(screen.queryByText('event-three')).not.toBeInTheDocument()
    await finish(requests[6], event('four'))
    await finish(requests[7], [])
    expect(screen.getByText('event-four')).toBeInTheDocument()
    expect(screen.queryByText('stale-history')).not.toBeInTheDocument()
    view.unmount(); expect(requests[6].signal?.aborted).toBe(true); expect(requests[7].signal?.aborted).toBe(true)
  })
})

import { ApiError } from '@/lib/api-error'

export type ApiRequestOptions = {
  method?: string
  body?: BodyInit | null
  headers?: HeadersInit
  auth?: boolean
  signal?: AbortSignal
  json?: unknown
}

type TokenGetter = () => string | null
type UnauthorizedHandler = () => void
type SessionOwner = { token: string | null; generation: number }

let getGeneration = () => 0
let onRefresh: (() => Promise<boolean>) | null = null
let refreshFlight: { owner: SessionOwner; promise: Promise<boolean> } | null = null
let unauthorizedOwner: SessionOwner | null = null

function sessionOwner(): SessionOwner {
  return { token: getToken(), generation: getGeneration() }
}

function sameOwner(a: SessionOwner, b: SessionOwner) {
  return a.token === b.token && a.generation === b.generation
}

function isCurrent(owner: SessionOwner) {
  return sameOwner(owner, sessionOwner())
}

let getToken: TokenGetter = () => {
  if (typeof window === 'undefined') return null
  return window.localStorage?.getItem('token') ?? null
}

let onUnauthorized: UnauthorizedHandler | null = null

export function configureApiClient(options: {
  getToken?: TokenGetter
  getGeneration?: () => number
  onUnauthorized?: UnauthorizedHandler | null
  onRefresh?: (() => Promise<boolean>) | null
}) {
  if (options.getToken) getToken = options.getToken
  if (options.getGeneration) getGeneration = options.getGeneration
  if (options.onUnauthorized !== undefined) {
    onUnauthorized = options.onUnauthorized
    unauthorizedOwner = null
  }
  if (options.onRefresh !== undefined) {
    onRefresh = options.onRefresh
    refreshFlight = null
  }
}

// Both bootstrap and concurrent safe reads share the same refresh for this session.
export function refreshApiSession(): Promise<boolean> {
  const owner = sessionOwner()
  if (refreshFlight && sameOwner(refreshFlight.owner, owner)) return refreshFlight.promise
  if (!onRefresh) return Promise.resolve(false)
  const handler = onRefresh
  const flight = { owner, promise: Promise.resolve(false) }
  flight.promise = Promise.resolve().then(() => isCurrent(owner) ? handler() : false).finally(() => {
    if (refreshFlight === flight) refreshFlight = null
  })
  refreshFlight = flight
  return flight.promise
}

export function resetApiClientForTests() {
  getToken = () => (typeof window === 'undefined' ? null : window.localStorage?.getItem('token') ?? null)
  onUnauthorized = null
  getGeneration = () => 0
  onRefresh = null
  refreshFlight = null
  unauthorizedOwner = null
}

async function parseBody(res: Response): Promise<unknown> {
  const text = await res.text()
  if (!text) return undefined
  try {
    return JSON.parse(text)
  } catch {
    return text
  }
}

function messageFromBody(body: unknown, fallback: string): string {
  if (!body) return fallback
  if (typeof body === 'string' && body.trim()) return body
  if (typeof body === 'object' && body !== null) {
    const record = body as Record<string, unknown>
    if (typeof record.message === 'string') return record.message
    if (typeof record.error === 'string') return record.error
  }
  return fallback
}

function buildRequest(path: string, options: ApiRequestOptions = {}, owner = sessionOwner()) {
  const headers = new Headers(options.headers || {})
  let body = options.body ?? null

  if (options.json !== undefined) {
    headers.set('Content-Type', 'application/json')
    body = JSON.stringify(options.json)
  } else if (typeof FormData !== 'undefined' && body instanceof FormData) {
    headers.delete('Content-Type')
  }

  if (options.auth) {
    const token = owner.token
    if (token) headers.set('Authorization', `Bearer ${token}`)
  }

  const url = path.startsWith('/api') ? path : `/api${path.startsWith('/') ? path : `/${path}`}`
  return {
    url,
    init: {
      method: options.method || (options.json !== undefined || body ? 'POST' : 'GET'),
      headers,
      body,
      signal: options.signal,
    } as RequestInit,
  }
}

function notifyUnauthorized(owner: SessionOwner) {
  if (owner.token && isCurrent(owner) && onUnauthorized &&
      (!unauthorizedOwner || !sameOwner(unauthorizedOwner, owner))) {
    unauthorizedOwner = owner
    onUnauthorized()
  }
}

async function requestResponse(path: string, options: ApiRequestOptions): Promise<Response> {
  const owner = sessionOwner()
  const { url, init } = buildRequest(path, options, owner)
  const res = await fetch(url, init)
  // Public/login failures cannot consume the current session's notification.
  if (res.status !== 401 || !options.auth || !owner.token || !isCurrent(owner)) return res
  const method = String(init.method).toUpperCase()
  if ((method === 'GET' || method === 'HEAD') && onRefresh) {
    const renewed = await refreshApiSession().catch(() => false)
    const current = sessionOwner()
    // Rotation is within a generation; a new login/logout must never replay this read.
    if (renewed && current.generation === owner.generation && current.token && !options.signal?.aborted) {
      const retry = buildRequest(path, options, current)
      const retried = await fetch(retry.url, retry.init)
      if (retried.status === 401) notifyUnauthorized(current)
      return retried
    }
  }
  // Mutations are never replayed, even if a refresh token is available.
  notifyUnauthorized(owner)
  return res
}

export async function apiRequest<T>(path: string, options: ApiRequestOptions = {}): Promise<T> {
  const res = await requestResponse(path, options)
  const parsed = await parseBody(res)
  if (!res.ok) {
    throw new ApiError(res.status, messageFromBody(parsed, res.statusText || 'request failed'), parsed)
  }
  return parsed as T
}

export async function apiBlob(path: string, options: ApiRequestOptions = {}): Promise<Blob> {
  const res = await requestResponse(path, options)
  if (!res.ok) {
    const parsed = await parseBody(res)
    throw new ApiError(res.status, messageFromBody(parsed, res.statusText || 'request failed'), parsed)
  }
  return res.blob()
}

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

let getToken: TokenGetter = () => {
  if (typeof window === 'undefined') return null
  return localStorage.getItem('token')
}

let onUnauthorized: UnauthorizedHandler | null = null
let unauthorizedNotified = false

export function configureApiClient(options: {
  getToken?: TokenGetter
  onUnauthorized?: UnauthorizedHandler | null
}) {
  if (options.getToken) getToken = options.getToken
  if (options.onUnauthorized !== undefined) {
    onUnauthorized = options.onUnauthorized
    unauthorizedNotified = false
  }
}

export function resetApiClientForTests() {
  getToken = () => (typeof window === 'undefined' ? null : localStorage.getItem('token'))
  onUnauthorized = null
  unauthorizedNotified = false
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

function buildRequest(path: string, options: ApiRequestOptions = {}) {
  const headers = new Headers(options.headers || {})
  let body = options.body ?? null

  if (options.json !== undefined) {
    headers.set('Content-Type', 'application/json')
    body = JSON.stringify(options.json)
  } else if (typeof FormData !== 'undefined' && body instanceof FormData) {
    headers.delete('Content-Type')
  }

  if (options.auth) {
    const token = getToken()
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

function notifyUnauthorized(status: number) {
  if (status === 401 && onUnauthorized && !unauthorizedNotified) {
    unauthorizedNotified = true
    onUnauthorized()
  }
}

export async function apiRequest<T>(path: string, options: ApiRequestOptions = {}): Promise<T> {
  const { url, init } = buildRequest(path, options)
  const res = await fetch(url, init)
  const parsed = await parseBody(res)
  if (!res.ok) {
    notifyUnauthorized(res.status)
    throw new ApiError(res.status, messageFromBody(parsed, res.statusText || 'request failed'), parsed)
  }
  return parsed as T
}

export async function apiBlob(path: string, options: ApiRequestOptions = {}): Promise<Blob> {
  const { url, init } = buildRequest(path, options)
  const res = await fetch(url, init)
  if (!res.ok) {
    notifyUnauthorized(res.status)
    const parsed = await parseBody(res)
    throw new ApiError(res.status, messageFromBody(parsed, res.statusText || 'request failed'), parsed)
  }
  return res.blob()
}

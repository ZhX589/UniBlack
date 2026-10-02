import { act, cleanup, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { Providers, useAuth } from '@/app/providers'
import { apiBlob, apiRequest, configureApiClient, refreshApiSession, resetApiClientForTests } from '@/lib/api'

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (error: unknown) => void
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no })
  return { promise, resolve, reject }
}
const response = (status: number, body: unknown = {}) => new Response(JSON.stringify(body), { status })
const token = (id: string, expired = false) => `header.${btoa(JSON.stringify({ user_id: id, username: id, roles: ['user'], exp: Math.floor(Date.now() / 1000) + (expired ? -100 : 3600) }))}.signature`
let auth: ReturnType<typeof useAuth>
function Probe() {
  auth = useAuth()
  return <div>{auth.status}:{auth.user?.username}</div>
}
function mount() { return render(<Providers><Probe /></Providers>) }
const fail = (promise: Promise<unknown>) => promise.catch((error) => error.status)
beforeEach(() => {
  const data = new Map<string, string>()
  vi.stubGlobal('localStorage', {
    getItem: (key: string) => data.get(key) ?? null,
    setItem: (key: string, value: string) => { data.set(key, value) },
    removeItem: (key: string) => { data.delete(key) },
    clear: () => data.clear(),
  })
  resetApiClientForTests()
})
afterEach(() => { cleanup(); resetApiClientForTests(); vi.unstubAllGlobals() })

describe('request session ownership', () => {
  it('anonymous/public/login 401 do not consume logout; each generation can logout once even with the same JWT', async () => {
    let access: string | null = null
    let generation = 0
    const logout = vi.fn()
    configureApiClient({ getToken: () => access, getGeneration: () => generation, onUnauthorized: logout })
    const anonymous = deferred<Response>()
    const fetch = vi.fn().mockReturnValueOnce(anonymous.promise).mockResolvedValue(response(401))
    vi.stubGlobal('fetch', fetch)
    const early = fail(apiRequest('/private', { auth: true }))
    access = 'same-token'; generation++
    anonymous.resolve(response(401)); await early
    await fail(apiRequest('/auth/login', { method: 'POST' }))
    expect(logout).not.toHaveBeenCalled()
    await fail(apiRequest('/private', { auth: true }))
    await fail(apiRequest('/private', { auth: true }))
    expect(logout).toHaveBeenCalledTimes(1)
    generation++
    await fail(apiRequest('/private', { auth: true }))
    expect(logout).toHaveBeenCalledTimes(2)
  })

  it('old token and old same-token generation late 401 cannot logout a new login (including blob)', async () => {
    let access = 'old'; let generation = 1
    const logout = vi.fn()
    configureApiClient({ getToken: () => access, getGeneration: () => generation, onUnauthorized: logout })
    const old = deferred<Response>(); const same = deferred<Response>()
    vi.stubGlobal('fetch', vi.fn().mockReturnValueOnce(old.promise).mockReturnValueOnce(same.promise))
    const first = fail(apiRequest('/private', { auth: true }))
    access = 'new'; generation++
    const second = fail(apiBlob('/download', { auth: true }))
    generation++
    old.resolve(response(401)); same.resolve(response(401))
    await Promise.all([first, second])
    expect(logout).not.toHaveBeenCalled()
  })

  it('single-flights concurrent GET refresh and retries each read once with rotated token', async () => {
    let access = 'old'
    const pending = deferred<boolean>()
    const refresh = vi.fn(async () => { await pending.promise; access = 'new'; return true })
    configureApiClient({ getToken: () => access, onRefresh: refresh })
    const fetch = vi.fn().mockResolvedValueOnce(response(401)).mockResolvedValueOnce(response(401))
      .mockImplementation(() => Promise.resolve(response(200, { renewed: true })))
    vi.stubGlobal('fetch', fetch)
    const first = apiRequest('/one', { auth: true }); const second = apiRequest('/two', { auth: true })
    await vi.waitFor(() => expect(refresh).toHaveBeenCalledTimes(1))
    pending.resolve(true)
    await expect(Promise.all([first, second])).resolves.toEqual([{ renewed: true }, { renewed: true }])
    expect(fetch).toHaveBeenCalledTimes(4)
    expect(fetch.mock.calls[2][1].headers.get('Authorization')).toBe('Bearer new')
    expect(fetch.mock.calls[3][1].headers.get('Authorization')).toBe('Bearer new')
  })

  it('does not replay a safe read into a new session while refresh is pending', async () => {
    let generation = 1; let access = 'old'
    const pending = deferred<boolean>(); const logout = vi.fn()
    configureApiClient({ getToken: () => access, getGeneration: () => generation, onRefresh: () => pending.promise, onUnauthorized: logout })
    const fetch = vi.fn().mockImplementation(() => Promise.resolve(response(401)))
    vi.stubGlobal('fetch', fetch)
    const request = fail(apiRequest('/private', { auth: true }))
    await act(async () => { await Promise.resolve() })
    generation++; access = 'new-login'
    pending.resolve(true)
    expect(await request).toBe(401)
    expect(fetch).toHaveBeenCalledTimes(1)
    expect(logout).not.toHaveBeenCalled()
  })

  it('a retried GET 401 notifies once without another refresh loop', async () => {
    let access = 'old'; const logout = vi.fn()
    const refresh = vi.fn(async () => { access = 'rotated'; return true })
    configureApiClient({ getToken: () => access, onRefresh: refresh, onUnauthorized: logout })
    const fetch = vi.fn().mockImplementation(() => Promise.resolve(response(401)))
    vi.stubGlobal('fetch', fetch)
    expect(await fail(apiRequest('/private', { auth: true }))).toBe(401)
    expect(fetch).toHaveBeenCalledTimes(2)
    expect(refresh).toHaveBeenCalledTimes(1)
    expect(logout).toHaveBeenCalledTimes(1)
  })

  it('refresh rejection logs out once and never loops; unsafe methods never refresh or replay', async () => {
    const pending = deferred<boolean>(); const logout = vi.fn(); const refresh = vi.fn(() => pending.promise)
    configureApiClient({ getToken: () => 'old', onRefresh: refresh, onUnauthorized: logout })
    const fetch = vi.fn().mockImplementation(() => Promise.resolve(response(401)))
    vi.stubGlobal('fetch', fetch)
    const first = fail(apiRequest('/one', { auth: true })); const second = fail(apiRequest('/two', { auth: true }))
    await vi.waitFor(() => expect(refresh).toHaveBeenCalledTimes(1))
    pending.reject(new Error('offline'))
    await Promise.all([first, second])
    expect(logout).toHaveBeenCalledTimes(1)
    for (const method of ['POST', 'PUT', 'PATCH', 'DELETE']) await fail(apiRequest('/write', { auth: true, method, json: { x: 1 } }))
    expect(fetch).toHaveBeenCalledTimes(6)
    expect(refresh).toHaveBeenCalledTimes(1)
  })
})

describe('provider refresh lifecycle', () => {
  function setupRefresh() {
    const pending = deferred<Response>()
    const fetch = vi.fn().mockImplementation((url: string) => url === '/api/auth/refresh'
      ? pending.promise : Promise.resolve(response(200)))
    vi.stubGlobal('fetch', fetch)
    return { pending, fetch }
  }
  it('bootstrap renews expired access with valid refresh, shares single-flight, and uses refreshed identity', async () => {
    window.localStorage.setItem('token', token('old', true)); window.localStorage.setItem('refresh_token', 'refresh-old')
    const { pending, fetch } = setupRefresh()
    mount()
    expect(screen.getByText('loading:')).toBeInTheDocument()
    const shared = refreshApiSession()
    await vi.waitFor(() => expect(fetch.mock.calls.filter(([url]) => url === '/api/auth/refresh')).toHaveLength(1))
    const fresh = token('fresh')
    await act(async () => { pending.resolve(response(200, { access_token: fresh, refresh_token: 'refresh-new' })); await shared })
    expect(screen.getByText('authenticated:fresh')).toBeInTheDocument()
    expect(window.localStorage.getItem('token')).toBe(fresh)
    expect(window.localStorage.getItem('refresh_token')).toBe('refresh-new')
  })
  it('bootstrap refresh failure clears both credentials', async () => {
    window.localStorage.setItem('token', token('old', true)); window.localStorage.setItem('refresh_token', 'invalid')
    const { pending } = setupRefresh(); mount()
    await act(async () => { pending.resolve(response(401)); await refreshApiSession() })
    expect(screen.getByText('anonymous:')).toBeInTheDocument()
    expect(window.localStorage.getItem('token')).toBeNull()
    expect(window.localStorage.getItem('refresh_token')).toBeNull()
  })
  it.each([200, 401])('late refresh %s cannot overwrite or logout a newer login', async (status) => {
    window.localStorage.setItem('token', token('old', true)); window.localStorage.setItem('refresh_token', 'old-refresh')
    const { pending } = setupRefresh(); mount()
    const oldFlight = refreshApiSession()
    await act(async () => { await Promise.resolve() })
    const fresh = token('new-login')
    act(() => auth.login(fresh, 'new-login-refresh'))
    await act(async () => { pending.resolve(response(status, { access_token: token('stale'), refresh_token: 'stale' })); await oldFlight })
    expect(screen.getByText('authenticated:new-login')).toBeInTheDocument()
    expect(window.localStorage.getItem('token')).toBe(fresh)
    expect(window.localStorage.getItem('refresh_token')).toBe('new-login-refresh')
  })
  it('consecutive login/401 cycles each logout while late old session 401 is ignored', async () => {
    const old = deferred<Response>()
    vi.stubGlobal('fetch', vi.fn().mockImplementation((url: string) => url === '/api/old' ? old.promise : Promise.resolve(response(url === '/api/write' ? 401 : 200))))
    mount()
    act(() => auth.login(token('first'), 'first-refresh'))
    const late = fail(apiRequest('/old', { auth: true, method: 'POST' }))
    await act(async () => { await fail(apiRequest('/write', { auth: true, method: 'POST' })) })
    expect(auth.status).toBe('anonymous')
    act(() => auth.login(token('second'), 'second-refresh'))
    await act(async () => { old.resolve(response(401)); await late })
    expect(auth.user?.username).toBe('second')
    await act(async () => { await fail(apiRequest('/write', { auth: true, method: 'POST' })) })
    expect(auth.status).toBe('anonymous')
  })
})

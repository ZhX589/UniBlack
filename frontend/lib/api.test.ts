import { afterEach, describe, expect, it, vi } from 'vitest'
import { apiRequest, configureApiClient, resetApiClientForTests } from '@/lib/api'
import { ApiError } from '@/lib/api-error'

describe('apiRequest', () => {
  afterEach(() => {
    resetApiClientForTests()
    vi.unstubAllGlobals()
  })

  it('parses JSON success responses', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        statusText: 'OK',
        text: async () => JSON.stringify({ subjects: 2 }),
      }),
    )
    await expect(apiRequest<{ subjects: number }>('/api/v1/statistics')).resolves.toEqual({ subjects: 2 })
  })

  it('adds session authorization and logs out once on 401', async () => {
    const logout = vi.fn()
    configureApiClient({
      getToken: () => 'token-123',
      onUnauthorized: logout,
    })
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      statusText: 'Unauthorized',
      text: async () => JSON.stringify({ message: 'expired' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    await expect(apiRequest('/private', { auth: true })).rejects.toMatchObject({
      status: 401,
      message: 'expired',
    })
    await expect(apiRequest('/private', { auth: true })).rejects.toBeInstanceOf(ApiError)
    expect(logout).toHaveBeenCalledTimes(1)
    expect(fetchMock.mock.calls[0][1].headers.get('Authorization')).toBe('Bearer token-123')
  })

  it('supports AbortSignal and non-JSON error bodies', async () => {
    const controller = new AbortController()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(async (_url, init) => {
        expect(init.signal).toBe(controller.signal)
        return {
          ok: false,
          status: 500,
          statusText: 'Server Error',
          text: async () => 'plain failure',
        }
      }),
    )
    await expect(apiRequest('/broken', { signal: controller.signal })).rejects.toMatchObject({
      status: 500,
      message: 'plain failure',
    })
  })
})

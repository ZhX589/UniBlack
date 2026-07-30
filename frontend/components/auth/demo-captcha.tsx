'use client'

import { useState } from 'react'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'

// Generic fallback shown for unknown (non-ApiError) failures — mirrors the
// historical page.tsx "发布失败" pattern but scoped to the captcha surface.
// Keeping the message Chinese-only follows the Task 1 audit label convention.
const GENERIC_CAPTCHA_ERROR = '人机验证失败，请重试'

export function DemoCaptcha({
  value,
  onChange,
  onError,
  purpose = 'register',
}: {
  value: string
  onChange: (value: string) => void
  // Optional error sink. When a verification request fails, the captcha
  // component reports a human-readable message via `onError` and leaves the
  // existing token untouched so the user's other draft input is preserved.
  // Callers that don't pass `onError` keep the historical behavior where
  // failures are swallowed (no `onChange('')` and no UI side effect).
  onError?: (message: string) => void
  purpose?: 'register' | 'submission' | 'appeal'
}) {
  const verified = value !== ''
  const [loading, setLoading] = useState(false)

  async function verify() {
    if (loading) return
    if (verified) return onChange('')
    setLoading(true)
    try {
      const path =
        purpose === 'submission' ? '/api/verification/demo/submission' : '/api/verification/demo/register'
      const data = await apiRequest<{ token: string }>(path, {
        method: 'POST',
        auth: purpose === 'submission',
        json: {},
      })
      onChange(data.token)
    } catch (err) {
      // Surface a recoverable message via onError; do NOT call onChange so the
      // caller's existing draft value is preserved (see [S9]).
      const message = err instanceof ApiError && err.message ? err.message : GENERIC_CAPTCHA_ERROR
      onError?.(message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <button
      type="button"
      aria-pressed={verified}
      onClick={verify}
      // While verification is in flight the button is disabled so the user
      // cannot trigger a second concurrent request; this matches the
      // historical "发布中..." publish-button affordance.
      disabled={loading}
      className={`min-h-touch w-full rounded-lg border p-4 text-left ${
        verified ? 'border-success bg-green-50 text-success' : 'border-border bg-background text-foreground'
      }`}
    >
      <strong>{verified ? '验证已完成' : loading ? '验证中...' : '我不是自动程序'}</strong>
      <span className="ml-2 text-sm text-muted">这是 UniBlack 内置演示验证，不会连接第三方服务。</span>
    </button>
  )
}

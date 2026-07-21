'use client'

import { useState } from 'react'
import { apiRequest } from '@/lib/api'

export function DemoCaptcha({
  value,
  onChange,
  purpose = 'register',
}: {
  value: string
  onChange: (value: string) => void
  purpose?: 'register' | 'submission' | 'appeal'
}) {
  const verified = value !== ''
  const [loading, setLoading] = useState(false)

  async function verify() {
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
    } finally {
      setLoading(false)
    }
  }

  return (
    <button
      type="button"
      aria-pressed={verified}
      onClick={verify}
      className={`min-h-touch w-full rounded-lg border p-4 text-left ${
        verified ? 'border-success bg-green-50 text-success' : 'border-border bg-background text-foreground'
      }`}
    >
      <strong>{verified ? '验证已完成' : loading ? '验证中...' : '我不是自动程序'}</strong>
      <span className="ml-2 text-sm text-muted">这是 UniBlack 内置演示验证，不会连接第三方服务。</span>
    </button>
  )
}

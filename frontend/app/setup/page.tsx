'use client'

import Link from 'next/link'
import { useEffect, useState, type FormEvent } from 'react'
import { useRouter } from 'next/navigation'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { LoadingState } from '@/components/ui/loading-state'
import { Panel } from '@/components/ui/panel'

export default function SetupPage() {
  const router = useRouter()
  const [initialized, setInitialized] = useState<boolean | null>(null)
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [siteName, setSiteName] = useState('UniBlack')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    apiRequest<{ initialized: boolean }>('/api/setup/check')
      .then((data) => setInitialized(!!data.initialized))
      .catch(() => setInitialized(false))
  }, [])

  const handleInitialize = async (e: FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    if (password !== confirmPassword) {
      setError('两次密码不一致')
      setLoading(false)
      return
    }
    if (password.length < 8) {
      setError('密码至少8位')
      setLoading(false)
      return
    }
    try {
      await apiRequest('/api/setup/initialize', {
        json: {
          admin_password: password,
          site_name: siteName,
        },
      })
      router.replace('/login')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '初始化失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }

  if (initialized === null) return <LoadingState message="检查中..." />

  if (initialized) {
    return (
      <div className="flex min-h-[70vh] items-center justify-center">
        <Panel className="w-full max-w-md text-center">
          <h1 className="mb-4 text-2xl font-bold">系统已初始化</h1>
          <p className="mb-4 text-muted">系统已经完成初始化，请直接登录。</p>
          <Link href="/login">
            <Button>去登录</Button>
          </Link>
        </Panel>
      </div>
    )
  }

  return (
    <div className="flex min-h-[70vh] items-center justify-center py-8">
      <Panel className="w-full max-w-md">
        <h1 className="mb-6 text-center text-2xl font-bold">系统初始化</h1>
        <p className="mb-6 text-center text-muted">欢迎使用 UniBlack！请设置管理员密码完成初始化。</p>
        {error && (
          <div className="mb-4">
            <Alert>{error}</Alert>
          </div>
        )}
        <form onSubmit={handleInitialize} className="space-y-4">
          <label className="block text-sm">
            系统名称
            <input
              className="mt-1 min-h-touch w-full rounded border border-border px-4 py-2"
              value={siteName}
              onChange={(e) => setSiteName(e.target.value)}
            />
          </label>
          <label className="block text-sm">
            管理员密码
            <input
              type="password"
              required
              minLength={8}
              className="mt-1 min-h-touch w-full rounded border border-border px-4 py-2"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </label>
          <label className="block text-sm">
            确认密码
            <input
              type="password"
              required
              minLength={8}
              className="mt-1 min-h-touch w-full rounded border border-border px-4 py-2"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
            />
          </label>
          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? '初始化中...' : '完成初始化'}
          </Button>
        </form>
      </Panel>
    </div>
  )
}

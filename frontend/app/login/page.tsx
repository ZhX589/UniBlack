'use client'

import Link from 'next/link'
import { useState, type FormEvent } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { useAuth } from '@/app/providers'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Panel } from '@/components/ui/panel'

type LoginResponse = {
  access_token: string
  refresh_token: string
}

export default function LoginPage() {
  const { login } = useAuth()
  const router = useRouter()
  const search = useSearchParams()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleLogin = async (event: FormEvent) => {
    event.preventDefault()
    setLoading(true)
    setError('')
    try {
      const data = await apiRequest<LoginResponse>('/api/auth/login', {
        json: { username, password },
      })
      login(data.access_token, data.refresh_token)
      const next = search.get('next') || '/'
      router.replace(next.startsWith('/') && !next.startsWith('//') ? next : '/')
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.status === 401 ? '用户名或密码错误' : err.message)
      } else {
        setError('无法连接后端 API，请确认后端已启动')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-[70vh] items-center justify-center py-8">
      <Panel className="w-full max-w-md">
        <h1 className="mb-6 text-center text-2xl font-bold">登录</h1>
        {error && (
          <div className="mb-4">
            <Alert>{error}</Alert>
          </div>
        )}
        <form onSubmit={handleLogin} className="space-y-4">
          <div>
            <label htmlFor="login-username" className="mb-2 block text-sm text-foreground">
              用户名
            </label>
            <input
              id="login-username"
              type="text"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              className="min-h-touch w-full rounded border border-border px-4 py-2"
              required
            />
          </div>
          <div>
            <label htmlFor="login-password" className="mb-2 block text-sm text-foreground">
              密码
            </label>
            <input
              id="login-password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              className="min-h-touch w-full rounded border border-border px-4 py-2"
              required
            />
          </div>
          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? '登录中...' : '登录'}
          </Button>
        </form>
        <p className="mt-4 text-center text-sm text-muted">
          还没有账号？
          <Link href="/register" className="text-primary hover:underline">
            注册
          </Link>
        </p>
      </Panel>
    </div>
  )
}

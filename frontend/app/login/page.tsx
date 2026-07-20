'use client'

import Link from 'next/link'
import { useState, type FormEvent } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { useAuth } from '@/app/providers'

export default function LoginPage() {
  const { login } = useAuth()
  const router = useRouter()
  const search = useSearchParams()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleLogin = async (e: FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    try {
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })

      if (res.ok) {
        const data = await res.json()
        login(data.access_token, data.refresh_token)
        router.replace(search.get('next') || '/')
        return
      }

      let message = '登录失败，请稍后重试'
      try {
        const data = await res.json()
        if (res.status === 401) {
          message = '用户名或密码错误'
        } else if (data?.error) {
          message = data.error
        } else if (res.status === 404 || res.status >= 500) {
          message = '无法连接后端 API，请确认后端已在 :8080 启动'
        }
      } catch {
        if (res.status === 404 || res.status >= 500) {
          message = '无法连接后端 API，请确认后端已在 :8080 启动'
        }
      }
      setError(message)
    } catch {
      setError('无法连接后端 API，请确认后端已在 :8080 启动')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="bg-white rounded-lg shadow-md p-8 w-full max-w-md">
        <h1 className="text-2xl font-bold text-center mb-6">登录</h1>

        {error && (
          <div className="bg-red-50 text-red-600 p-3 rounded-lg mb-4">
            {error}
          </div>
        )}

        <form onSubmit={handleLogin}>
          <div className="mb-4">
            <label className="block text-gray-700 mb-2">用户名</label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full border rounded-lg px-4 py-2"
              required
            />
          </div>

          <div className="mb-6">
            <label className="block text-gray-700 mb-2">密码</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full border rounded-lg px-4 py-2"
              required
            />
          </div>

          <button
            type="submit"
            className="w-full bg-blue-600 text-white py-2 rounded-lg hover:bg-blue-700"
            disabled={loading}
          >
            {loading ? '登录中...' : '登录'}
          </button>
        </form>

        <div className="mt-4 text-center text-gray-500">
          <p>还没有账号？<Link href="/register" className="text-blue-600 hover:underline">注册</Link></p>
        </div>
      </div>
    </div>
  )
}

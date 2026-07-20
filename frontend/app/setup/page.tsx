'use client'

import { useState, useEffect } from 'react'

export default function SetupPage() {
  const [initialized, setInitialized] = useState<boolean | null>(null)
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [siteName, setSiteName] = useState('UniBlack')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    checkSetup()
  }, [])

  const checkSetup = async () => {
    try {
      const res = await fetch('/api/setup/check')
      const data = await res.json()
      setInitialized(data.initialized)
    } catch (error) {
      console.error('Failed to check setup:', error)
    }
  }

  const handleInitialize = async (e: React.FormEvent) => {
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
      const res = await fetch('/api/setup/initialize', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          admin_password: password,
          site_name: siteName,
        }),
      })

      if (res.ok) {
        window.location.href = '/login'
      } else {
        const data = await res.json()
        setError(data.error || '初始化失败')
      }
    } catch (error) {
      setError('初始化失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }

  if (initialized === null) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">检查中...</div>
      </div>
    )
  }

  if (initialized) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="bg-white rounded-lg shadow-md p-8 max-w-md w-full text-center">
          <h1 className="text-2xl font-bold mb-4">系统已初始化</h1>
          <p className="text-gray-600 mb-4">系统已经完成初始化，请直接登录。</p>
          <a href="/login" className="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 inline-block">
            去登录
          </a>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="bg-white rounded-lg shadow-md p-8 max-w-md w-full">
        <h1 className="text-2xl font-bold text-center mb-6">系统初始化</h1>
        <p className="text-gray-600 mb-6 text-center">
          欢迎使用 UniBlack！请设置管理员密码完成初始化。
        </p>

        {error && (
          <div className="bg-red-50 text-red-600 p-3 rounded-lg mb-4">
            {error}
          </div>
        )}

        <form onSubmit={handleInitialize}>
          <div className="mb-4">
            <label className="block text-gray-700 mb-2">系统名称</label>
            <input
              type="text"
              value={siteName}
              onChange={(e) => setSiteName(e.target.value)}
              className="w-full border rounded-lg px-4 py-2"
            />
          </div>

          <div className="mb-4">
            <label className="block text-gray-700 mb-2">管理员密码</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full border rounded-lg px-4 py-2"
              required
              minLength={8}
            />
          </div>

          <div className="mb-6">
            <label className="block text-gray-700 mb-2">确认密码</label>
            <input
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className="w-full border rounded-lg px-4 py-2"
              required
              minLength={8}
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-blue-600 text-white py-2 rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? '初始化中...' : '完成初始化'}
          </button>
        </form>
      </div>
    </div>
  )
}

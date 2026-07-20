'use client'

import { useState, useEffect } from 'react'

export default function RegisterPage() {
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
    verificationCode: '',
    captchaToken: '',
  })
  const [settings, setSettings] = useState<any>({})
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const [codeSent, setCodeSent] = useState(false)

  useEffect(() => {
    fetchSettings()
  }, [])

  const fetchSettings = async () => {
    try {
      const res = await fetch('/api/settings/public')
      const data = await res.json()
      setSettings(data)
    } catch (error) {
      console.error('Failed to fetch settings:', error)
    }
  }

  const handleSendCode = async () => {
    if (!formData.email) {
      setError('请输入邮箱')
      return
    }

    try {
      const res = await fetch('/api/auth/send-verification-code', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: formData.email }),
      })

      if (res.ok) {
        setCodeSent(true)
        setError('')
      } else {
        setError('发送验证码失败')
      }
    } catch (error) {
      setError('发送验证码失败')
    }
  }

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    if (formData.password !== formData.confirmPassword) {
      setError('两次密码不一致')
      setLoading(false)
      return
    }

    if (formData.password.length < 8) {
      setError('密码至少8位')
      setLoading(false)
      return
    }

    try {
      const res = await fetch('/api/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          username: formData.username,
          email: formData.email,
          password: formData.password,
          verification_code: formData.verificationCode,
          captcha_token: formData.captchaToken,
        }),
      })

      if (res.ok) {
        setSuccess(true)
      } else {
        const data = await res.json()
        setError(data.error || '注册失败')
      }
    } catch (error) {
      setError('注册失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }

  if (!settings['auth.registration_enabled']) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="bg-white rounded-lg shadow-md p-8 max-w-md w-full text-center">
          <h1 className="text-2xl font-bold mb-4">注册已关闭</h1>
          <p className="text-gray-600">系统暂不开放注册，请联系管理员。</p>
          <a href="/login" className="text-blue-600 hover:underline mt-4 inline-block">返回登录</a>
        </div>
      </div>
    )
  }

  if (success) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="bg-white rounded-lg shadow-md p-8 max-w-md w-full text-center">
          <div className="text-green-600 text-5xl mb-4">✓</div>
          <h1 className="text-2xl font-bold mb-4">注册成功</h1>
          <p className="text-gray-600 mb-4">您的账号已创建成功，请登录。</p>
          <a href="/login" className="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 inline-block">
            去登录
          </a>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center py-8">
      <div className="bg-white rounded-lg shadow-md p-8 max-w-md w-full">
        <h1 className="text-2xl font-bold text-center mb-6">注册账号</h1>

        {error && (
          <div className="bg-red-50 text-red-600 p-3 rounded-lg mb-4">
            {error}
          </div>
        )}

        <form onSubmit={handleRegister}>
          <div className="mb-4">
            <label className="block text-gray-700 mb-2">用户名</label>
            <input
              type="text"
              value={formData.username}
              onChange={(e) => setFormData({ ...formData, username: e.target.value })}
              className="w-full border rounded-lg px-4 py-2"
              required
            />
          </div>

          <div className="mb-4">
            <label className="block text-gray-700 mb-2">邮箱</label>
            <div className="flex gap-2">
              <input
                type="email"
                value={formData.email}
                onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                className="flex-1 border rounded-lg px-4 py-2"
                required
              />
              {settings['security.email_verification'] && (
                <button
                  type="button"
                  onClick={handleSendCode}
                  className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 whitespace-nowrap"
                  disabled={codeSent}
                >
                  {codeSent ? '已发送' : '发送验证码'}
                </button>
              )}
            </div>
          </div>

          {settings['security.email_verification'] && codeSent && (
            <div className="mb-4">
              <label className="block text-gray-700 mb-2">验证码</label>
              <input
                type="text"
                value={formData.verificationCode}
                onChange={(e) => setFormData({ ...formData, verificationCode: e.target.value })}
                className="w-full border rounded-lg px-4 py-2"
                placeholder="请输入6位验证码"
              />
            </div>
          )}

          <div className="mb-4">
            <label className="block text-gray-700 mb-2">密码</label>
            <input
              type="password"
              value={formData.password}
              onChange={(e) => setFormData({ ...formData, password: e.target.value })}
              className="w-full border rounded-lg px-4 py-2"
              required
              minLength={8}
            />
          </div>

          <div className="mb-6">
            <label className="block text-gray-700 mb-2">确认密码</label>
            <input
              type="password"
              value={formData.confirmPassword}
              onChange={(e) => setFormData({ ...formData, confirmPassword: e.target.value })}
              className="w-full border rounded-lg px-4 py-2"
              required
            />
          </div>

          {settings['security.captcha_enabled'] && (
            <div className="mb-6">
              <div className="border rounded-lg p-4 text-center text-gray-500">
                人机验证区域
              </div>
            </div>
          )}

          <button
            type="submit"
            className="w-full bg-blue-600 text-white py-2 rounded-lg hover:bg-blue-700"
            disabled={loading}
          >
            {loading ? '注册中...' : '注册'}
          </button>
        </form>

        <div className="mt-4 text-center text-gray-500">
          <p>已有账号？<a href="/login" className="text-blue-600 hover:underline">登录</a></p>
        </div>
      </div>
    </div>
  )
}

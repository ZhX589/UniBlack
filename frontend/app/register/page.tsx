'use client'

import { useState, useEffect, useRef } from 'react'
import { settingBool, settingString } from '@/lib/settings'

declare global {
  interface Window {
    turnstile?: { render: (el: HTMLElement, opts: Record<string, unknown>) => string }
    grecaptcha?: { render: (el: HTMLElement, opts: Record<string, unknown>) => number; getResponse: (id?: number) => string }
    hcaptcha?: { render: (el: HTMLElement, opts: Record<string, unknown>) => string; getResponse: (id?: string) => string }
  }
}

export default function RegisterPage() {
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
    verificationCode: '',
    captchaToken: '',
  })
  const [settings, setSettings] = useState<Record<string, unknown> | null>(null)
  const [settingsError, setSettingsError] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const [codeSent, setCodeSent] = useState(false)
  const captchaRef = useRef<HTMLDivElement>(null)
  const captchaWidgetId = useRef<string | number | null>(null)

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const res = await fetch('/api/settings/public')
        if (!res.ok) throw new Error('failed')
        const data = await res.json()
        if (!cancelled) setSettings(data)
      } catch {
        if (!cancelled) {
          setSettingsError('无法加载注册配置，请确认后端已启动')
          // fail-open for registration_enabled so we don't flash "closed"
          setSettings({ 'auth.registration_enabled': true })
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  const captchaEnabled = settings ? settingBool(settings, 'security.captcha_enabled') : false
  const captchaProvider = settings ? settingString(settings, 'security.captcha_provider', 'turnstile') : 'turnstile'
  const captchaSiteKey = settings ? settingString(settings, 'security.captcha_site_key') : ''
  const emailVerification = settings ? settingBool(settings, 'security.email_verification') : false
  const registrationEnabled = settings ? settingBool(settings, 'auth.registration_enabled', true) : true

  useEffect(() => {
    if (!settings || !captchaEnabled || !captchaSiteKey || !captchaRef.current) return
    const el = captchaRef.current
    el.innerHTML = ''
    captchaWidgetId.current = null

    const onToken = (token: string) => {
      setFormData((prev) => ({ ...prev, captchaToken: token }))
    }

    if (captchaProvider === 'turnstile') {
      const scriptId = 'cf-turnstile-script'
      const render = () => {
        if (window.turnstile && captchaRef.current) {
          captchaWidgetId.current = window.turnstile.render(captchaRef.current, {
            sitekey: captchaSiteKey,
            callback: onToken,
          })
        }
      }
      if (!document.getElementById(scriptId)) {
        const s = document.createElement('script')
        s.id = scriptId
        s.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
        s.async = true
        s.onload = render
        document.body.appendChild(s)
      } else {
        render()
      }
    } else if (captchaProvider === 'recaptcha') {
      const scriptId = 'recaptcha-script'
      const render = () => {
        if (window.grecaptcha && captchaRef.current) {
          captchaWidgetId.current = window.grecaptcha.render(captchaRef.current, {
            sitekey: captchaSiteKey,
            callback: onToken,
          })
        }
      }
      if (!document.getElementById(scriptId)) {
        const s = document.createElement('script')
        s.id = scriptId
        s.src = 'https://www.google.com/recaptcha/api.js?render=explicit'
        s.async = true
        s.onload = render
        document.body.appendChild(s)
      } else {
        render()
      }
    } else if (captchaProvider === 'hcaptcha') {
      const scriptId = 'hcaptcha-script'
      const render = () => {
        if (window.hcaptcha && captchaRef.current) {
          captchaWidgetId.current = window.hcaptcha.render(captchaRef.current, {
            sitekey: captchaSiteKey,
            callback: onToken,
          })
        }
      }
      if (!document.getElementById(scriptId)) {
        const s = document.createElement('script')
        s.id = scriptId
        s.src = 'https://js.hcaptcha.com/1/api.js?render=explicit'
        s.async = true
        s.onload = render
        document.body.appendChild(s)
      } else {
        render()
      }
    }
  }, [settings, captchaEnabled, captchaProvider, captchaSiteKey])

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
        const data = await res.json().catch(() => ({}))
        setError(data.error || '发送验证码失败')
      }
    } catch {
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

    let captchaToken = formData.captchaToken
    if (captchaEnabled) {
      if (captchaProvider === 'recaptcha' && window.grecaptcha) {
        captchaToken = window.grecaptcha.getResponse(captchaWidgetId.current as number) || captchaToken
      }
      if (captchaProvider === 'hcaptcha' && window.hcaptcha) {
        captchaToken = window.hcaptcha.getResponse(String(captchaWidgetId.current ?? '')) || captchaToken
      }
      if (!captchaToken) {
        setError('请完成人机验证')
        setLoading(false)
        return
      }
    }
    if (emailVerification && !formData.verificationCode) {
      setError('请填写邮箱验证码')
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
          captcha_token: captchaToken,
        }),
      })
      if (res.ok) {
        setSuccess(true)
      } else {
        const data = await res.json().catch(() => ({}))
        setError(data.error || '注册失败')
      }
    } catch {
      setError('注册失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }

  if (settings === null) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-gray-500">加载注册配置...</div>
      </div>
    )
  }

  if (!registrationEnabled) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="bg-white rounded-lg shadow-md p-8 max-w-md w-full text-center">
          <h1 className="text-2xl font-bold mb-4">注册已关闭</h1>
          <p className="text-gray-600">系统暂不开放注册，请联系管理员。</p>
          <a href="/login" className="text-blue-600 hover:underline mt-4 inline-block">
            返回登录
          </a>
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
          <a
            href="/login"
            className="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 inline-block"
          >
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
        {settingsError && (
          <div className="bg-yellow-50 text-yellow-800 p-3 rounded-lg mb-4 text-sm">{settingsError}</div>
        )}
        {error && <div className="bg-red-50 text-red-600 p-3 rounded-lg mb-4">{error}</div>}

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
              {emailVerification && (
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

          {emailVerification && (
            <div className="mb-4">
              <label className="block text-gray-700 mb-2">验证码</label>
              <input
                type="text"
                value={formData.verificationCode}
                onChange={(e) => setFormData({ ...formData, verificationCode: e.target.value })}
                className="w-full border rounded-lg px-4 py-2"
                placeholder="请输入邮箱验证码"
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

          {captchaEnabled && (
            <div className="mb-6">
              {captchaSiteKey ? (
                <div ref={captchaRef} className="flex justify-center min-h-[78px]" />
              ) : (
                <div className="border rounded-lg p-4 text-center text-amber-700 bg-amber-50 text-sm">
                  已开启人机验证，但未配置 Site Key。请在管理控制台 → 安全配置中填写。
                </div>
              )}
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
          <p>
            已有账号？
            <a href="/login" className="text-blue-600 hover:underline">
              登录
            </a>
          </p>
        </div>
      </div>
    </div>
  )
}

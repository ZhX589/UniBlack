'use client'

import Link from 'next/link'
import { useEffect, useState, type FormEvent } from 'react'
import { DemoCaptcha } from '@/components/auth/demo-captcha'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import { settingBool } from '@/lib/settings'
import type { PublicSettings } from '@/lib/types'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { LoadingState } from '@/components/ui/loading-state'
import { Panel } from '@/components/ui/panel'

export default function RegisterPage() {
  const [settings, setSettings] = useState<PublicSettings | null>(null)
  const [form, setForm] = useState({
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
    verificationCode: '',
    captchaToken: '',
  })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [sent, setSent] = useState(false)
  const [success, setSuccess] = useState(false)

  useEffect(() => {
    apiRequest<PublicSettings>('/api/settings/public')
      .then(setSettings)
      .catch(() => setSettings({ 'auth.registration_enabled': true }))
  }, [])

  const registrationEnabled = settings ? settingBool(settings, 'auth.registration_enabled', true) : true
  const emailVerification = settings ? settingBool(settings, 'security.email_verification') : false
  const captchaEnabled = settings ? settingBool(settings, 'security.captcha_enabled') : false

  async function sendCode() {
    if (!form.email) return setError('请输入邮箱')
    try {
      await apiRequest('/api/auth/send-verification-code', {
        json: { email: form.email },
      })
      setSent(true)
      setError('')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '发送验证码失败')
    }
  }

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError('')
    if (form.password !== form.confirmPassword) return setError('两次密码不一致')
    if (captchaEnabled && !form.captchaToken) return setError('请完成人机验证')
    setLoading(true)
    try {
      await apiRequest('/api/auth/register', {
        json: {
          username: form.username,
          email: form.email,
          password: form.password,
          verification_code: form.verificationCode,
          captcha_token: form.captchaToken,
        },
      })
      setSuccess(true)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '注册失败')
    } finally {
      setLoading(false)
    }
  }

  if (!settings) return <LoadingState message="加载注册配置..." />
  if (!registrationEnabled) {
    return (
      <div className="py-12 text-center">
        <h1 className="text-2xl font-bold">注册已关闭</h1>
        <Link href="/login" className="text-primary hover:underline">
          返回登录
        </Link>
      </div>
    )
  }
  if (success) {
    return (
      <div className="py-12 text-center">
        <h1 className="text-2xl font-bold">注册成功</h1>
        <Link href="/login" className="text-primary hover:underline">
          去登录
        </Link>
      </div>
    )
  }

  return (
    <main className="mx-auto max-w-md py-8">
      <h1 className="mb-6 text-2xl font-bold">注册账号</h1>
      {error && (
        <div className="mb-4">
          <Alert>{error}</Alert>
        </div>
      )}
      <Panel>
        <form onSubmit={submit} className="space-y-4">
          <label className="block text-sm">
            用户名
            <input
              required
              className="mt-1 min-h-touch w-full rounded border border-border p-2"
              value={form.username}
              onChange={(e) => setForm({ ...form, username: e.target.value })}
            />
          </label>
          <div className="flex gap-2">
            <label className="block min-w-0 flex-1 text-sm">
              邮箱
              <input
                required
                type="email"
                className="mt-1 min-h-touch w-full rounded border border-border p-2"
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
              />
            </label>
            {emailVerification && (
              <Button type="button" variant="secondary" className="self-end" onClick={sendCode} disabled={sent}>
                {sent ? '已发送' : '发送验证码'}
              </Button>
            )}
          </div>
          {emailVerification && (
            <label className="block text-sm">
              验证码（开发环境为 123456）
              <input
                required
                className="mt-1 min-h-touch w-full rounded border border-border p-2"
                value={form.verificationCode}
                onChange={(e) => setForm({ ...form, verificationCode: e.target.value })}
              />
            </label>
          )}
          <label className="block text-sm">
            密码
            <input
              required
              type="password"
              minLength={8}
              className="mt-1 min-h-touch w-full rounded border border-border p-2"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
            />
          </label>
          <label className="block text-sm">
            确认密码
            <input
              required
              type="password"
              minLength={8}
              className="mt-1 min-h-touch w-full rounded border border-border p-2"
              value={form.confirmPassword}
              onChange={(e) => setForm({ ...form, confirmPassword: e.target.value })}
            />
          </label>
          {captchaEnabled && (
            <DemoCaptcha
              value={form.captchaToken}
              onChange={(captchaToken) => setForm({ ...form, captchaToken })}
            />
          )}
          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? '注册中...' : '注册'}
          </Button>
        </form>
      </Panel>
      <p className="mt-4 text-center text-sm">
        已有账号？
        <Link href="/login" className="text-primary hover:underline">
          登录
        </Link>
      </p>
    </main>
  )
}

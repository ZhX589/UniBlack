'use client'

import Link from 'next/link'
import { useEffect, useState, type FormEvent } from 'react'
import { DemoCaptcha } from '@/components/auth/demo-captcha'
import { settingBool } from '@/lib/settings'

export default function RegisterPage() {
  const [settings, setSettings] = useState<Record<string, unknown> | null>(null)
  const [form, setForm] = useState({ username: '', email: '', password: '', confirmPassword: '', verificationCode: '', captchaToken: '' })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [sent, setSent] = useState(false)
  const [success, setSuccess] = useState(false)

  useEffect(() => { fetch('/api/settings/public').then((r) => r.json()).then(setSettings).catch(() => setSettings({ 'auth.registration_enabled': true })) }, [])
  const registrationEnabled = settings ? settingBool(settings, 'auth.registration_enabled', true) : true
  const emailVerification = settings ? settingBool(settings, 'security.email_verification') : false
  const captchaEnabled = settings ? settingBool(settings, 'security.captcha_enabled') : false

  async function sendCode() {
    if (!form.email) return setError('请输入邮箱')
    const res = await fetch('/api/auth/send-verification-code', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ email: form.email }) })
    if (!res.ok) return setError('发送验证码失败')
    setSent(true); setError('')
  }
  async function submit(e: FormEvent) {
    e.preventDefault(); setError('')
    if (form.password !== form.confirmPassword) return setError('两次密码不一致')
    if (captchaEnabled && !form.captchaToken) return setError('请完成人机验证')
    setLoading(true)
    try {
      const res = await fetch('/api/auth/register', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username: form.username, email: form.email, password: form.password, verification_code: form.verificationCode, captcha_token: form.captchaToken }) })
      if (!res.ok) { const data = await res.json().catch(() => ({})); setError(data.error || '注册失败'); return }
      setSuccess(true)
    } finally { setLoading(false) }
  }
  if (!settings) return <div className="py-12 text-center text-gray-500">加载注册配置...</div>
  if (!registrationEnabled) return <div className="py-12 text-center"><h1 className="text-2xl font-bold">注册已关闭</h1><Link href="/login" className="text-blue-600">返回登录</Link></div>
  if (success) return <div className="py-12 text-center"><h1 className="text-2xl font-bold">注册成功</h1><Link href="/login" className="text-blue-600">去登录</Link></div>
  return <main className="mx-auto max-w-md py-8"><h1 className="mb-6 text-2xl font-bold">注册账号</h1>{error && <p className="mb-4 rounded bg-red-50 p-3 text-red-700">{error}</p>}<form onSubmit={submit} className="space-y-4 rounded-lg bg-white p-6 shadow"><input required placeholder="用户名" className="w-full rounded border p-2" value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })}/><div className="flex gap-2"><input required type="email" placeholder="邮箱" className="min-w-0 flex-1 rounded border p-2" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })}/>{emailVerification && <button type="button" onClick={sendCode} disabled={sent} className="rounded bg-blue-600 px-3 text-white">{sent ? '已发送' : '发送验证码'}</button>}</div>{emailVerification && <input required placeholder="验证码（开发环境为 123456）" className="w-full rounded border p-2" value={form.verificationCode} onChange={(e) => setForm({ ...form, verificationCode: e.target.value })}/>}<input required type="password" minLength={8} placeholder="密码" className="w-full rounded border p-2" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })}/><input required type="password" minLength={8} placeholder="确认密码" className="w-full rounded border p-2" value={form.confirmPassword} onChange={(e) => setForm({ ...form, confirmPassword: e.target.value })}/>{captchaEnabled && <DemoCaptcha value={form.captchaToken} onChange={(captchaToken) => setForm({ ...form, captchaToken })}/>}<button disabled={loading} className="w-full rounded bg-blue-600 p-2 text-white">{loading ? '注册中...' : '注册'}</button></form><p className="mt-4 text-center">已有账号？<Link href="/login" className="text-blue-600">登录</Link></p></main>
}

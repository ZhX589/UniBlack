'use client'

import Link from 'next/link'
import { useState, type FormEvent } from 'react'
import { DemoCaptcha } from '@/components/auth/demo-captcha'

type Account = { platform: string; username: string; account_id: string }
type EventInput = { title: string; details: string; severity: number }

export default function SubmitPage() {
  const [accounts, setAccounts] = useState<Account[]>([{ platform: 'qq', username: '', account_id: '' }])
  const [events, setEvents] = useState<EventInput[]>([{ title: '', details: '', severity: 1 }])
  const [displayName, setDisplayName] = useState('')
  const [verificationCode, setVerificationCode] = useState('')
  const [captchaToken, setCaptchaToken] = useState('')
  const [error, setError] = useState('')
  const [result, setResult] = useState<{ public_id: string } | null>(null)
  const [loading, setLoading] = useState(false)
  const token = typeof window === 'undefined' ? '' : localStorage.getItem('token') || ''

  if (!token) return <main className="py-8"><h1 className="text-3xl font-bold">提交对象与事件</h1><p className="mt-3 text-gray-600">登录并完成邮箱验证后可发布对象、账号和事件。提交后默认公开，错误或恶意内容可被申诉、修正、撤销，并可能导致提交权限处罚。</p><div className="mt-6 rounded-lg border border-dashed bg-gray-50 p-6 text-gray-500"><p>账号、事件、证据上传和发布操作在登录后可用。</p><Link href="/login?next=/submit" className="mt-4 inline-block rounded bg-gray-700 px-4 py-2 text-white">登录后提交</Link></div></main>

  async function publish(e: FormEvent) {
    e.preventDefault(); setError(''); setLoading(true)
    try {
      const res = await fetch('/api/subjects/publish', { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ display_name: displayName, accounts, events, verification_code: verificationCode, captcha_token: captchaToken }) })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) { setError(data.error || '发布失败'); return }
      setResult(data)
    } finally { setLoading(false) }
  }
  if (result) return <main className="py-8"><h1 className="text-2xl font-bold">对象与事件已发布</h1><p className="mt-2">公开对象 ID：<code>{result.public_id}</code></p><Link href={`/subjects/${result.public_id}`} className="mt-4 inline-block text-blue-600">查看对象档案</Link></main>
  return <main className="mx-auto max-w-3xl py-8"><h1 className="text-3xl font-bold">提交对象与事件</h1><p className="mt-2 text-gray-600">分段填写对象、账号、事件与验证信息。开发环境邮箱验证码为 123456。</p>{error && <p className="mt-4 rounded bg-red-50 p-3 text-red-700">{error}</p>}<form onSubmit={publish} className="mt-6 space-y-6">
    <section id="subject" className="rounded-lg bg-white p-5 shadow"><h2 className="font-semibold">1. 对象</h2><input className="mt-3 w-full rounded border p-2" placeholder="通用名（留空时使用第一条账号用户名）" value={displayName} onChange={(e) => setDisplayName(e.target.value)} /></section>
    <section id="accounts" className="rounded-lg bg-white p-5 shadow"><h2 className="font-semibold">2. 账号</h2>{accounts.map((a,i)=><div className="mt-3 flex gap-2" key={i}><input className="w-28 rounded border p-2" value={a.platform} onChange={(e)=>setAccounts(accounts.map((x,j)=>j===i?{...x,platform:e.target.value}:x))}/><input className="flex-1 rounded border p-2" placeholder="用户名" value={a.username} onChange={(e)=>setAccounts(accounts.map((x,j)=>j===i?{...x,username:e.target.value}:x))}/><input className="flex-1 rounded border p-2" placeholder="账号 ID" value={a.account_id} onChange={(e)=>setAccounts(accounts.map((x,j)=>j===i?{...x,account_id:e.target.value}:x))}/></div>)}<button type="button" className="mt-3 text-blue-600" onClick={()=>setAccounts([...accounts,{platform:'custom',username:'',account_id:''}])}>添加账号</button></section>
    <section id="events" className="rounded-lg bg-white p-5 shadow"><h2 className="font-semibold">3. 事件</h2>{events.map((v,i)=><div className="mt-3 space-y-2" key={i}><input required className="w-full rounded border p-2" placeholder="事件标题" value={v.title} onChange={(e)=>setEvents(events.map((x,j)=>j===i?{...x,title:e.target.value}:x))}/><textarea required className="w-full rounded border p-2" placeholder="事件详情" value={v.details} onChange={(e)=>setEvents(events.map((x,j)=>j===i?{...x,details:e.target.value}:x))}/></div>)}<button type="button" className="mt-3 text-blue-600" onClick={()=>setEvents([...events,{title:'',details:'',severity:1}])}>添加事件</button></section>
    <section id="verification" className="rounded-lg bg-white p-5 shadow"><h2 className="font-semibold">4. 验证与发布</h2><input className="mt-3 w-full rounded border p-2" placeholder="邮箱验证码" value={verificationCode} onChange={(e)=>setVerificationCode(e.target.value)}/><div className="mt-3"><DemoCaptcha value={captchaToken} onChange={setCaptchaToken} purpose="submission"/></div><button disabled={loading} className="mt-4 w-full rounded bg-red-700 p-2 text-white">{loading?'发布中...':'确认并公开发布'}</button></section>
  </form></main>
}

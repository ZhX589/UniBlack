'use client'

import { useState, type FormEvent } from 'react'

export default function SanctionsPage() {
  const [form, setForm] = useState({ user_id: '', type: 'warning', reason: '', ends_at: '' })
  const [message, setMessage] = useState('')
  async function submit(e: FormEvent) {
    e.preventDefault()
    const token = localStorage.getItem('token')
    if (!token) return (window.location.href = '/login')
    const res = await fetch('/api/admin/sanctions', { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ ...form, ends_at: form.ends_at || undefined }) })
    setMessage(res.ok ? '处罚已记录' : '保存失败')
  }
  return <main className="py-8"><h1 className="text-3xl font-bold">提交处罚</h1><p className="mt-2 text-gray-600">仅用于已核查的恶意、伪造或重复滥用提交。</p><form onSubmit={submit} className="mt-6 max-w-xl space-y-3 rounded bg-white p-6 shadow"><input required className="w-full border p-2" placeholder="用户 UUID" value={form.user_id} onChange={(e) => setForm({ ...form, user_id: e.target.value })}/><select className="w-full border p-2" value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value })}><option value="warning">警告</option><option value="submission_suspension">限期禁止提交</option><option value="submission_ban">永久禁止提交</option></select>{form.type === 'submission_suspension' && <input required type="datetime-local" className="w-full border p-2" value={form.ends_at} onChange={(e) => setForm({ ...form, ends_at: e.target.value })}/>}<textarea required className="w-full border p-2" placeholder="处罚原因" value={form.reason} onChange={(e) => setForm({ ...form, reason: e.target.value })}/><button className="rounded bg-red-700 px-4 py-2 text-white">记录处罚</button>{message && <p>{message}</p>}</form></main>
}

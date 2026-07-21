'use client'

import { useEffect, useState, type FormEvent } from 'react'

type Sanction = {
  id: string
  user_id: string
  type: string
  reason: string
  ends_at?: string
  revoked_at?: string
  created_at: string
}

export default function SanctionsPage() {
  const [form, setForm] = useState({ user_id: '', type: 'warning', reason: '', ends_at: '' })
  const [items, setItems] = useState<Sanction[]>([])
  const [message, setMessage] = useState('')
  const token = typeof window === 'undefined' ? '' : localStorage.getItem('token') || ''

  async function load() {
    if (!token) return (window.location.href = '/login')
    const res = await fetch('/api/admin/sanctions?page=1&page_size=50', {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!res.ok) return
    const data = await res.json()
    setItems(data.items || [])
  }

  useEffect(() => {
    load()
  }, [])

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!token) return (window.location.href = '/login')
    const body: Record<string, unknown> = { ...form }
    if (form.type !== 'submission_suspension') delete body.ends_at
    else if (form.ends_at) body.ends_at = new Date(form.ends_at).toISOString()
    const res = await fetch('/api/admin/sanctions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify(body),
    })
    setMessage(res.ok ? '处罚已记录' : '保存失败')
    if (res.ok) {
      setForm({ user_id: '', type: 'warning', reason: '', ends_at: '' })
      load()
    }
  }

  async function revoke(id: string) {
    const reason = window.prompt('撤销原因')
    if (!reason) return
    const res = await fetch(`/api/admin/sanctions/${id}/revoke`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ reason }),
    })
    setMessage(res.ok ? '处罚已撤销' : '撤销失败')
    if (res.ok) load()
  }

  return (
    <main className="space-y-8 py-4">
      <div>
        <h1 className="text-3xl font-bold">提交处罚</h1>
        <p className="mt-2 text-gray-600">仅用于已核查的恶意、伪造或重复滥用提交。所有操作写入审计记录。</p>
      </div>
      {message && <p className="rounded bg-blue-50 p-3 text-blue-800">{message}</p>}
      <form onSubmit={submit} className="max-w-xl space-y-3 rounded bg-white p-6 shadow">
        <input
          required
          className="w-full border p-2"
          placeholder="用户 UUID"
          value={form.user_id}
          onChange={(e) => setForm({ ...form, user_id: e.target.value })}
        />
        <select className="w-full border p-2" value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value })}>
          <option value="warning">警告</option>
          <option value="submission_suspension">限期禁止提交</option>
          <option value="submission_ban">永久禁止提交</option>
        </select>
        {form.type === 'submission_suspension' && (
          <input
            required
            type="datetime-local"
            className="w-full border p-2"
            value={form.ends_at}
            onChange={(e) => setForm({ ...form, ends_at: e.target.value })}
          />
        )}
        <textarea
          required
          className="w-full border p-2"
          placeholder="处罚原因"
          value={form.reason}
          onChange={(e) => setForm({ ...form, reason: e.target.value })}
        />
        <button className="rounded bg-red-700 px-4 py-2 text-white">记录处罚</button>
      </form>
      <section className="rounded bg-white p-6 shadow">
        <h2 className="font-semibold">最近处罚</h2>
        <div className="mt-3 overflow-x-auto">
          <table className="min-w-full text-left text-sm">
            <thead>
              <tr className="border-b">
                <th className="py-2 pr-4">用户</th>
                <th className="py-2 pr-4">类型</th>
                <th className="py-2 pr-4">原因</th>
                <th className="py-2 pr-4">状态</th>
                <th className="py-2">操作</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr key={item.id} className="border-b align-top">
                  <td className="py-2 pr-4 font-mono text-xs">{item.user_id}</td>
                  <td className="py-2 pr-4">{item.type}</td>
                  <td className="py-2 pr-4">{item.reason}</td>
                  <td className="py-2 pr-4">{item.revoked_at ? '已撤销' : '生效中'}</td>
                  <td className="py-2">
                    {!item.revoked_at && (
                      <button type="button" className="text-red-700" onClick={() => revoke(item.id)}>
                        撤销
                      </button>
                    )}
                  </td>
                </tr>
              ))}
              {items.length === 0 && (
                <tr>
                  <td colSpan={5} className="py-4 text-gray-500">
                    暂无处罚记录
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>
    </main>
  )
}
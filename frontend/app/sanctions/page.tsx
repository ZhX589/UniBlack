'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'

type Sanction = {
  id: string
  type: string
  reason: string
  ends_at?: string
  revoked_at?: string
  created_at: string
}

export default function MySanctionsPage() {
  const [items, setItems] = useState<Sanction[]>([])
  const [message, setMessage] = useState('')
  const token = typeof window === 'undefined' ? '' : localStorage.getItem('token') || ''

  async function load() {
    if (!token) {
      window.location.href = '/login?next=/sanctions'
      return
    }
    const res = await fetch('/api/sanctions/me', { headers: { Authorization: `Bearer ${token}` } })
    if (!res.ok) {
      setMessage('加载失败')
      return
    }
    const data = await res.json()
    setItems(data.items || [])
  }

  useEffect(() => {
    load()
  }, [])

  async function appeal(id: string) {
    const reason = window.prompt('请填写处罚申诉理由（每个处罚仅一次）')
    if (!reason) return
    const res = await fetch(`/api/sanctions/${id}/appeal`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ reason }),
    })
    const data = await res.json().catch(() => ({}))
    setMessage(res.ok ? '申诉已提交' : data.error || '申诉失败')
    if (res.ok) load()
  }

  return (
    <main className="mx-auto max-w-3xl py-8">
      <h1 className="text-3xl font-bold">我的处罚</h1>
      <p className="mt-2 text-gray-600">被处罚用户可对每条处罚提出一次申诉。申诉通过后处罚将被撤销。</p>
      {message && <p className="mt-4 rounded bg-blue-50 p-3 text-blue-800">{message}</p>}
      <div className="mt-6 space-y-3">
        {items.map((item) => (
          <div key={item.id} className="rounded border bg-white p-4 shadow-sm">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <strong>{item.type}</strong>
              <span className="text-sm text-gray-500">{item.revoked_at ? '已撤销' : '生效中'}</span>
            </div>
            <p className="mt-2 text-sm">{item.reason}</p>
            {!item.revoked_at && (
              <button type="button" className="mt-3 text-sm text-red-700" onClick={() => appeal(item.id)}>
                提出申诉
              </button>
            )}
          </div>
        ))}
        {items.length === 0 && <p className="text-gray-500">暂无处罚记录</p>}
      </div>
      <Link href="/" className="mt-6 inline-block text-blue-600">
        返回首页
      </Link>
    </main>
  )
}

'use client'

import { useState, type FormEvent } from 'react'

export default function ArchivesPage() {
  const [publicID, setPublicID] = useState('')
  const [message, setMessage] = useState('')
  const [preview, setPreview] = useState<Record<string, unknown> | null>(null)
  const token = typeof window === 'undefined' ? '' : localStorage.getItem('token') || ''

  async function exportArchive() {
    setMessage('')
    if (!publicID.trim()) return setMessage('请输入对象公开 ID')
    const res = await fetch(`/api/admin/exports/subjects/${encodeURIComponent(publicID.trim())}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      return setMessage(data.error || '导出失败')
    }
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${publicID.trim()}.zip`
    a.click()
    URL.revokeObjectURL(url)
    setMessage('导出已开始下载')
  }

  async function previewImport(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setMessage('')
    setPreview(null)
    const form = e.currentTarget
    const fileInput = form.elements.namedItem('archive') as HTMLInputElement
    if (!fileInput.files?.length) return setMessage('请选择归档 ZIP')
    const body = new FormData()
    body.append('archive', fileInput.files[0])
    const res = await fetch('/api/admin/imports/preview', {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
      body,
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) return setMessage(data.error || '预览失败')
    setPreview(data)
    setMessage(data.valid ? '预览通过，可确认导入' : '预览存在冲突，不能导入')
  }

  async function confirmImport(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setMessage('')
    const form = e.currentTarget
    const fileInput = form.elements.namedItem('archive') as HTMLInputElement
    if (!fileInput.files?.length) return setMessage('请选择归档 ZIP')
    const body = new FormData()
    body.append('archive', fileInput.files[0])
    const res = await fetch('/api/admin/imports', {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
      body,
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) return setMessage(data.error || '导入失败')
    setMessage(`导入成功：${data.public_id}`)
  }

  return (
    <main className="space-y-8 py-4">
      <div>
        <h1 className="text-3xl font-bold">对象归档</h1>
        <p className="mt-2 text-gray-600">导出可校验 ZIP，或预览/确认导入。确认导入会拒绝已存在的 public ID 与账号冲突。</p>
      </div>
      {message && <p className="rounded bg-blue-50 p-3 text-blue-800">{message}</p>}
      <section className="rounded bg-white p-6 shadow">
        <h2 className="font-semibold">导出对象</h2>
        <div className="mt-3 flex gap-2">
          <input
            className="min-w-0 flex-1 rounded border p-2"
            placeholder="UBS_..."
            value={publicID}
            onChange={(e) => setPublicID(e.target.value)}
          />
          <button type="button" onClick={exportArchive} className="rounded bg-gray-900 px-4 py-2 text-white">
            导出 ZIP
          </button>
        </div>
      </section>
      <section className="rounded bg-white p-6 shadow">
        <h2 className="font-semibold">导入预览</h2>
        <form onSubmit={previewImport} className="mt-3 space-y-3">
          <input name="archive" type="file" accept=".zip,application/zip" className="block w-full" />
          <button className="rounded bg-blue-600 px-4 py-2 text-white">预览</button>
        </form>
        {preview && (
          <pre className="mt-3 overflow-auto rounded bg-gray-50 p-3 text-xs">{JSON.stringify(preview, null, 2)}</pre>
        )}
      </section>
      <section className="rounded bg-white p-6 shadow">
        <h2 className="font-semibold">确认导入</h2>
        <form onSubmit={confirmImport} className="mt-3 space-y-3">
          <input name="archive" type="file" accept=".zip,application/zip" className="block w-full" />
          <button className="rounded bg-red-700 px-4 py-2 text-white">确认导入</button>
        </form>
      </section>
    </main>
  )
}

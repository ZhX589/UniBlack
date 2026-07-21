'use client'

import { useEffect, useState, type FormEvent } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/app/providers'
import { apiBlob, apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { LoadingState } from '@/components/ui/loading-state'
import { Panel } from '@/components/ui/panel'

export default function ArchivesPage() {
  const { status } = useAuth()
  const router = useRouter()
  const [publicID, setPublicID] = useState('')
  const [message, setMessage] = useState('')
  const [preview, setPreview] = useState<Record<string, unknown> | null>(null)

  useEffect(() => {
    if (status === 'anonymous') router.replace('/login?next=/admin/archives')
  }, [status, router])

  async function exportArchive() {
    setMessage('')
    if (!publicID.trim()) return setMessage('请输入对象公开 ID')
    try {
      const blob = await apiBlob(`/api/admin/exports/subjects/${encodeURIComponent(publicID.trim())}`, {
        auth: true,
      })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${publicID.trim()}.zip`
      a.click()
      URL.revokeObjectURL(url)
      setMessage('导出已开始下载')
    } catch (err) {
      setMessage(err instanceof ApiError ? err.message : '导出失败')
    }
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
    try {
      const data = await apiRequest<Record<string, unknown>>('/api/admin/imports/preview', {
        method: 'POST',
        auth: true,
        body,
      })
      setPreview(data)
      setMessage(data.valid ? '预览通过，可确认导入' : '预览存在冲突，不能导入')
    } catch (err) {
      setMessage(err instanceof ApiError ? err.message : '预览失败')
    }
  }

  async function confirmImport(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setMessage('')
    const form = e.currentTarget
    const fileInput = form.elements.namedItem('archive') as HTMLInputElement
    if (!fileInput.files?.length) return setMessage('请选择归档 ZIP')
    const body = new FormData()
    body.append('archive', fileInput.files[0])
    try {
      const data = await apiRequest<{ public_id?: string }>('/api/admin/imports', {
        method: 'POST',
        auth: true,
        body,
      })
      setMessage(`导入成功：${data.public_id || ''}`)
    } catch (err) {
      setMessage(err instanceof ApiError ? err.message : '导入失败')
    }
  }

  if (status === 'loading') return <LoadingState />
  if (status === 'anonymous') return null

  return (
    <main className="space-y-8 py-4">
      <div>
        <h1 className="text-3xl font-bold">对象归档</h1>
        <p className="mt-2 text-muted">
          导出可校验 ZIP，或预览/确认导入。确认导入会拒绝已存在的 public ID 与账号冲突。
        </p>
      </div>
      {message && <Alert tone="success">{message}</Alert>}
      <Panel>
        <h2 className="font-semibold">导出对象</h2>
        <div className="mt-3 flex flex-col gap-2 sm:flex-row">
          <input
            className="min-h-touch min-w-0 flex-1 rounded border border-border p-2"
            placeholder="UBS_..."
            value={publicID}
            onChange={(e) => setPublicID(e.target.value)}
          />
          <Button type="button" onClick={exportArchive}>
            导出 ZIP
          </Button>
        </div>
      </Panel>
      <Panel>
        <h2 className="font-semibold">导入预览</h2>
        <form onSubmit={previewImport} className="mt-3 space-y-3">
          <input name="archive" type="file" accept=".zip,application/zip" className="block w-full" />
          <Button type="submit" variant="secondary">
            预览
          </Button>
        </form>
        {preview && (
          <pre className="mt-3 overflow-auto rounded bg-background p-3 text-xs">{JSON.stringify(preview, null, 2)}</pre>
        )}
      </Panel>
      <Panel>
        <h2 className="font-semibold">确认导入</h2>
        <form onSubmit={confirmImport} className="mt-3 space-y-3">
          <input name="archive" type="file" accept=".zip,application/zip" className="block w-full" />
          <Button type="submit" variant="danger">
            确认导入
          </Button>
        </form>
      </Panel>
    </main>
  )
}

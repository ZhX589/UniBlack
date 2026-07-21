'use client'

import Link from 'next/link'
import { useCallback, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/app/providers'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import type { Sanction } from '@/lib/types'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { ErrorState } from '@/components/ui/error-state'
import { LoadingState } from '@/components/ui/loading-state'
import { Panel } from '@/components/ui/panel'

export default function MySanctionsPage() {
  const { status } = useAuth()
  const router = useRouter()
  const [items, setItems] = useState<Sanction[]>([])
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    if (status !== 'authenticated') return
    setLoading(true)
    setError('')
    try {
      const data = await apiRequest<{ items?: Sanction[] }>('/api/sanctions/me', { auth: true })
      setItems(data.items || [])
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }, [status])

  useEffect(() => {
    if (status === 'anonymous') router.replace('/login?next=/sanctions')
  }, [status, router])

  useEffect(() => {
    void load()
  }, [load])

  async function appeal(id: string) {
    const reason = window.prompt('请填写处罚申诉理由（每个处罚仅一次）')
    if (!reason) return
    try {
      await apiRequest(`/api/sanctions/${id}/appeal`, {
        auth: true,
        json: { reason },
      })
      setMessage('申诉已提交')
      await load()
    } catch (err) {
      setMessage(err instanceof ApiError ? err.message : '申诉失败')
    }
  }

  if (status === 'loading') return <LoadingState />
  if (status === 'anonymous') return null

  return (
    <main className="mx-auto max-w-3xl py-8">
      <h1 className="text-3xl font-bold">我的处罚</h1>
      <p className="mt-2 text-muted">被处罚用户可对每条处罚提出一次申诉。申诉通过后处罚将被撤销。</p>
      {message && (
        <div className="mt-4">
          <Alert tone="success">{message}</Alert>
        </div>
      )}
      {error && (
        <div className="mt-4">
          <ErrorState message={error} />
        </div>
      )}
      {loading ? (
        <LoadingState />
      ) : (
        <div className="mt-6 space-y-3">
          {items.map((item) => (
            <Panel key={item.id}>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <strong>{item.type}</strong>
                <span className="text-sm text-muted">{item.revoked_at ? '已撤销' : '生效中'}</span>
              </div>
              <p className="mt-2 text-sm">{item.reason}</p>
              {!item.revoked_at && (
                <Button type="button" variant="ghost" className="mt-3" onClick={() => appeal(item.id)}>
                  提出申诉
                </Button>
              )}
            </Panel>
          ))}
          {items.length === 0 && <EmptyState message="暂无处罚记录" />}
        </div>
      )}
      <Link href="/" className="mt-6 inline-block text-primary hover:underline">
        返回首页
      </Link>
    </main>
  )
}

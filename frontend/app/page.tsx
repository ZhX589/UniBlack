'use client'

import Link from 'next/link'
import { FormEvent, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { apiRequest } from '@/lib/api'
import type { Statistics } from '@/lib/types'
import { useSite } from '@/app/providers'
import { Button } from '@/components/ui/button'
import { Panel } from '@/components/ui/panel'
import { Alert } from '@/components/ui/alert'

export default function Home() {
  const { name, description } = useSite()
  const router = useRouter()
  const [query, setQuery] = useState('')
  const [stats, setStats] = useState<Statistics | null>(null)
  const [statsError, setStatsError] = useState(false)

  useEffect(() => {
    let cancelled = false
    apiRequest<Statistics>('/api/v1/statistics')
      .then((data) => {
        if (!cancelled) setStats(data)
      })
      .catch(() => {
        if (!cancelled) setStatsError(true)
      })
    return () => {
      cancelled = true
    }
  }, [])

  function onVerify(event: FormEvent) {
    event.preventDefault()
    const value = query.trim()
    if (!value) return
    router.push(`/search?q=${encodeURIComponent(value)}`)
  }

  return (
    <div className="mx-auto max-w-5xl py-8">
      <h1 className="mb-3 text-4xl font-bold text-foreground">{name}</h1>
      <p className="mb-8 max-w-3xl text-muted">
        {description || '社区维护的可复用云黑系统。核验账号、查看公开事件，并在登录后提交证据。'}
      </p>

      <Panel className="mb-8">
        <h2 className="mb-2 text-xl font-semibold">核验账号</h2>
        <p className="mb-4 text-sm text-muted">输入平台账号、用户名或公开 ID，立即查询是否存在公开记录。</p>
        <form onSubmit={onVerify} className="flex flex-col gap-3 sm:flex-row">
          <label className="sr-only" htmlFor="home-verify">
            核验账号
          </label>
          <input
            id="home-verify"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="QQ / Discord / Telegram / 邮箱..."
            className="min-h-touch flex-1 rounded border border-border bg-surface px-4 py-2"
          />
          <Button type="submit">开始核验</Button>
        </form>
        <div className="mt-4 flex flex-wrap gap-4 text-sm">
          <Link href="/search" className="text-primary hover:underline">
            高级查询
          </Link>
          <Link href="/submit" className="text-primary hover:underline">
            登录后提交
          </Link>
        </div>
      </Panel>

      <section>
        <h2 className="mb-3 text-lg font-semibold">公开统计</h2>
        {statsError && <Alert tone="warning">统计暂不可用，查询功能不受影响。</Alert>}
        <div className="mt-3 grid grid-cols-2 gap-4 md:grid-cols-2">
          <Panel className="text-center">
            <div className="text-3xl font-bold">{stats ? stats.subjects : '—'}</div>
            <div className="text-sm text-muted">活跃对象</div>
          </Panel>
          <Panel className="text-center">
            <div className="text-3xl font-bold">{stats ? stats.events : '—'}</div>
            <div className="text-sm text-muted">公开事件</div>
          </Panel>
        </div>
      </section>
    </div>
  )
}

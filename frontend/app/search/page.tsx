'use client'

import Link from 'next/link'
import { FormEvent, useEffect, useState } from 'react'
import { useSearchParams } from 'next/navigation'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import type { Subject } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { Panel } from '@/components/ui/panel'
import { Badge } from '@/components/ui/badge'
import { EmptyState } from '@/components/ui/empty-state'
import { ErrorState } from '@/components/ui/error-state'
import { LoadingState } from '@/components/ui/loading-state'

type SearchMode = 'idle' | 'loading' | 'success' | 'empty' | 'error'

export default function SearchPage() {
  const params = useSearchParams()
  const initial = params.get('q') || ''
  const [query, setQuery] = useState(initial)
  const [platform, setPlatform] = useState('qq')
  const [results, setResults] = useState<Subject[]>([])
  const [mode, setMode] = useState<SearchMode>(initial ? 'loading' : 'idle')
  const [error, setError] = useState('')

  async function runSearch(value: string) {
    setMode('loading')
    setError('')
    try {
      const data = await apiRequest<{ results?: Subject[] }>(`/api/v1/search?q=${encodeURIComponent(value)}`)
      const rows = data.results || []
      setResults(rows)
      setMode(rows.length ? 'success' : 'empty')
    } catch (err) {
      setResults([])
      setError(err instanceof ApiError ? err.message : '查询失败')
      setMode('error')
    }
  }

  async function runLookup(value: string) {
    setMode('loading')
    setError('')
    try {
      const data = await apiRequest<Subject>(`/api/v1/lookup?platform=${platform}&value=${encodeURIComponent(value)}`)
      setResults([data])
      setMode('success')
    } catch (err) {
      setResults([])
      if (err instanceof ApiError && err.status === 404) {
        setMode('empty')
        return
      }
      setError(err instanceof ApiError ? err.message : '精确查询失败')
      setMode('error')
    }
  }

  useEffect(() => {
    if (initial.trim()) {
      setQuery(initial)
      void runSearch(initial.trim())
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initial])

  function onSearch(event: FormEvent) {
    event.preventDefault()
    const value = query.trim()
    if (!value) return
    void runSearch(value)
  }

  function onLookup(event: FormEvent) {
    event.preventDefault()
    const value = query.trim()
    if (!value) return
    void runLookup(value)
  }

  return (
    <div className="mx-auto max-w-5xl py-8">
      <h1 className="mb-6 text-3xl font-bold">查询黑名单</h1>

      <Panel className="mb-6">
        <form onSubmit={onSearch} className="flex flex-col gap-3 sm:flex-row">
          <label className="sr-only" htmlFor="search-query">
            搜索关键词
          </label>
          <input
            id="search-query"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="输入QQ号、用户名、邮箱等"
            className="min-h-touch flex-1 rounded border border-border px-4 py-2"
          />
          <Button type="submit" disabled={mode === 'loading'}>
            {mode === 'loading' ? '搜索中...' : '搜索'}
          </Button>
        </form>
      </Panel>

      <Panel className="mb-6">
        <h2 className="mb-4 text-xl font-semibold">精确查询</h2>
        <form onSubmit={onLookup} className="flex flex-col gap-3 md:flex-row">
          <label className="sr-only" htmlFor="lookup-platform">
            平台
          </label>
          <select
            id="lookup-platform"
            value={platform}
            onChange={(event) => setPlatform(event.target.value)}
            className="min-h-touch rounded border border-border px-4 py-2"
          >
            <option value="qq">QQ</option>
            <option value="wechat">微信</option>
            <option value="bilibili">B站</option>
            <option value="douyin">抖音</option>
            <option value="x">X (Twitter)</option>
            <option value="telegram">Telegram</option>
            <option value="discord">Discord</option>
            <option value="steam">Steam</option>
            <option value="phone">手机号</option>
            <option value="email">邮箱</option>
          </select>
          <Button type="submit" variant="secondary" disabled={mode === 'loading'}>
            精确查询
          </Button>
        </form>
      </Panel>

      {mode === 'loading' && <LoadingState message="正在查询..." />}
      {mode === 'error' && <ErrorState message={error || '查询失败'} />}
      {mode === 'empty' && <EmptyState message="未找到相关结果" />}
      {mode === 'idle' && <EmptyState message="输入关键词后开始查询" />}
      {mode === 'success' && (
        <Panel>
          <h2 className="mb-4 text-xl font-semibold">查询结果</h2>
          <div className="space-y-4">
            {results.map((subject) => (
              <div key={subject.id} className="rounded border border-border p-4">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <h3 className="text-lg font-semibold">
                      <Link href={`/subjects/${subject.public_id || subject.id}`} className="text-primary hover:underline">
                        {subject.display_name}
                      </Link>
                    </h3>
                    {subject.public_id && <p className="font-mono text-sm text-muted">{subject.public_id}</p>}
                    <p className="text-muted">风险等级: {subject.risk_level ?? 0}/5</p>
                  </div>
                  <Badge tone={subject.status === 'active' ? 'danger' : subject.status === 'cleared' ? 'success' : 'neutral'}>
                    {subject.status === 'active' ? '黑名单' : subject.status === 'cleared' ? '已清除' : subject.status}
                  </Badge>
                </div>
                {(subject.accounts?.length || subject.identifiers?.length) && (
                  <div className="mt-3 flex flex-wrap gap-2">
                    {(subject.accounts || []).map((account, index) => (
                      <span key={`a-${index}`} className="rounded bg-background px-2 py-1 text-sm">
                        {account.platform}: {account.username || account.account_id}
                      </span>
                    ))}
                    {(subject.identifiers || []).map((identifier, index) => (
                      <span key={`i-${index}`} className="rounded bg-background px-2 py-1 text-sm">
                        {identifier.platform}: {identifier.value}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </Panel>
      )}
    </div>
  )
}

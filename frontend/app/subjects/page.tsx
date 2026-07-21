'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import type { Subject } from '@/lib/types'
import { Badge } from '@/components/ui/badge'
import { EmptyState } from '@/components/ui/empty-state'
import { ErrorState } from '@/components/ui/error-state'
import { LoadingState } from '@/components/ui/loading-state'
import { Panel } from '@/components/ui/panel'

export default function SubjectsPage() {
  const [subjects, setSubjects] = useState<Subject[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)
    apiRequest<{ subjects?: Subject[] }>('/api/v1/subjects?page=1&page_size=20', {
      signal: controller.signal,
    })
      .then((data) => setSubjects(data.subjects || []))
      .catch((err) => {
        if (controller.signal.aborted) return
        setError(err instanceof ApiError ? err.message : '加载名单失败')
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false)
      })
    return () => controller.abort()
  }, [])

  return (
    <div className="py-8">
      <h1 className="mb-6 text-3xl font-bold">黑名单列表</h1>
      {error && <ErrorState message={error} />}
      {loading ? (
        <LoadingState />
      ) : subjects.length === 0 ? (
        <EmptyState message="暂无黑名单记录" />
      ) : (
        <Panel className="overflow-x-auto p-0">
          <table className="w-full min-w-[40rem] text-sm">
            <thead className="bg-background text-left text-muted">
              <tr>
                <th className="px-4 py-3">名称</th>
                <th className="px-4 py-3">公开 ID</th>
                <th className="px-4 py-3">风险等级</th>
                <th className="px-4 py-3">状态</th>
              </tr>
            </thead>
            <tbody>
              {subjects.map((subject) => (
                <tr key={subject.id} className="border-t border-border">
                  <td className="px-4 py-3">
                    <Link
                      href={`/subjects/${subject.public_id || subject.id}`}
                      className="text-primary hover:underline"
                    >
                      {subject.display_name}
                    </Link>
                  </td>
                  <td className="px-4 py-3 font-mono text-xs text-muted">{subject.public_id || subject.id}</td>
                  <td className="px-4 py-3">{subject.risk_level ?? 0}/5</td>
                  <td className="px-4 py-3">
                    <Badge tone={subject.status === 'active' ? 'danger' : subject.status === 'cleared' ? 'success' : 'neutral'}>
                      {subject.status === 'active' ? '黑名单' : subject.status === 'cleared' ? '已清除' : subject.status}
                    </Badge>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Panel>
      )}
    </div>
  )
}

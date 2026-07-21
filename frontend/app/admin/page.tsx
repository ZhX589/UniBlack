'use client'

import { useCallback, useEffect, useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/app/providers'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { ErrorState } from '@/components/ui/error-state'
import { LoadingState } from '@/components/ui/loading-state'
import { Panel } from '@/components/ui/panel'

type Submission = { id: string; reason: string; status: string }
type LegacyCase = { id: string; title: string; status: string; severity: number }

export default function AdminPage() {
  const { status } = useAuth()
  const router = useRouter()
  const [activeTab, setActiveTab] = useState<'legacy-submissions' | 'legacy-cases'>('legacy-submissions')
  const [submissions, setSubmissions] = useState<Submission[]>([])
  const [cases, setCases] = useState<LegacyCase[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const fetchData = useCallback(async () => {
    if (status !== 'authenticated') return
    setLoading(true)
    setError('')
    try {
      if (activeTab === 'legacy-submissions') {
        const data = await apiRequest<{ submissions?: Submission[] }>('/api/submissions?page=1&page_size=20', {
          auth: true,
        })
        setSubmissions(data.submissions || [])
      } else {
        const data = await apiRequest<{ cases?: LegacyCase[] }>('/api/cases?page=1&page_size=20', { auth: true })
        setCases(data.cases || [])
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }, [activeTab, status])

  useEffect(() => {
    if (status === 'anonymous') router.replace('/login?next=/admin')
  }, [status, router])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const handleReview = async (id: string, reviewStatus: string) => {
    try {
      await apiRequest(`/api/submissions/${id}/review`, {
        auth: true,
        json: { status: reviewStatus, review_notes: '' },
      })
      await fetchData()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '审核失败')
    }
  }

  if (status === 'loading') return <LoadingState message="正在验证管理权限..." />
  if (status === 'anonymous') return null

  return (
    <div className="py-4">
      <h1 className="mb-2 text-3xl font-bold">内容治理</h1>
      <p className="mb-4 text-sm text-muted">
        新内容默认通过「提交对象与事件」直接发布。下列入口仅用于兼容窗口内的旧 Submission / Case 数据，Sunset 见{' '}
        <code>docs/api/case-event-migration.md</code>。
      </p>
      <div className="mb-4 flex flex-wrap gap-2">
        <Link href="/submit">
          <Button>Event 发布入口</Button>
        </Link>
        <Link href="/admin/sanctions">
          <Button variant="secondary">处罚与申诉裁决</Button>
        </Link>
        <Link href="/admin/archives">
          <Button variant="secondary">归档导入导出</Button>
        </Link>
      </div>

      <div className="mb-6 flex gap-2">
        <Button
          variant={activeTab === 'legacy-submissions' ? 'primary' : 'secondary'}
          onClick={() => setActiveTab('legacy-submissions')}
        >
          旧举报审核（兼容）
        </Button>
        <Button variant={activeTab === 'legacy-cases' ? 'primary' : 'secondary'} onClick={() => setActiveTab('legacy-cases')}>
          旧案件列表（兼容）
        </Button>
      </div>

      {error && <ErrorState message={error} />}
      {loading ? (
        <LoadingState />
      ) : activeTab === 'legacy-submissions' ? (
        <Panel className="overflow-x-auto p-0">
          {submissions.length === 0 ? (
            <div className="p-6">
              <EmptyState message="暂无旧举报记录" />
            </div>
          ) : (
            <table className="w-full min-w-[36rem] text-sm">
              <thead className="bg-background text-left text-muted">
                <tr>
                  <th className="px-4 py-3">原因</th>
                  <th className="px-4 py-3">状态</th>
                  <th className="px-4 py-3">操作</th>
                </tr>
              </thead>
              <tbody>
                {submissions.map((sub) => (
                  <tr key={sub.id} className="border-t border-border">
                    <td className="px-4 py-3">{sub.reason}</td>
                    <td className="px-4 py-3">
                      <Badge tone={sub.status === 'pending' ? 'warning' : sub.status === 'approved' ? 'success' : 'danger'}>
                        {sub.status}
                      </Badge>
                    </td>
                    <td className="px-4 py-3">
                      {sub.status === 'pending' && (
                        <div className="flex gap-2">
                          <Button variant="ghost" onClick={() => handleReview(sub.id, 'approved')}>
                            通过
                          </Button>
                          <Button variant="ghost" onClick={() => handleReview(sub.id, 'rejected')}>
                            驳回
                          </Button>
                        </div>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Panel>
      ) : (
        <Panel className="overflow-x-auto p-0">
          {cases.length === 0 ? (
            <div className="p-6">
              <EmptyState message="暂无旧案件记录" />
            </div>
          ) : (
            <table className="w-full min-w-[36rem] text-sm">
              <thead className="bg-background text-left text-muted">
                <tr>
                  <th className="px-4 py-3">标题</th>
                  <th className="px-4 py-3">状态</th>
                  <th className="px-4 py-3">严重程度</th>
                  <th className="px-4 py-3">兼容跳转</th>
                </tr>
              </thead>
              <tbody>
                {cases.map((item) => (
                  <tr key={item.id} className="border-t border-border">
                    <td className="px-4 py-3">{item.title}</td>
                    <td className="px-4 py-3">
                      <Badge>{item.status}</Badge>
                    </td>
                    <td className="px-4 py-3">{item.severity}/5</td>
                    <td className="px-4 py-3">
                      <Link href={`/events/${item.id}`} className="text-primary hover:underline">
                        尝试作为事件打开
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Panel>
      )}
    </div>
  )
}

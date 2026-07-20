'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'

type CaseDetail = {
  id: string
  subject_id: string
  title: string
  description?: string
  status: string
  severity: number
  verdict?: string
  created_at?: string
}

export default function CaseDetailPage() {
  const params = useParams()
  const id = params?.id as string
  const [data, setData] = useState<CaseDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!id) return
    let cancelled = false
    ;(async () => {
      setLoading(true)
      setError('')
      try {
        const res = await fetch(`/api/v1/cases/${id}`)
        const body = await res.json().catch(() => ({}))
        if (!res.ok) {
          throw new Error(body.error || '案件不存在或未公开')
        }
        if (!cancelled) setData(body)
      } catch (e: any) {
        if (!cancelled) setError(e.message || '加载失败')
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [id])

  if (loading) {
    return <div className="py-12 text-center text-gray-500">加载中...</div>
  }

  if (error || !data) {
    return (
      <div className="py-12 text-center">
        <p className="text-red-600 mb-4">{error || '未找到该案件'}</p>
        <p className="text-sm text-gray-500 mb-4">
          公开接口仅返回状态为 approved 或 closed 的案件。
        </p>
        <Link href="/subjects" className="text-blue-600 hover:underline">
          返回黑名单
        </Link>
      </div>
    )
  }

  return (
    <div className="py-8 max-w-2xl mx-auto">
      <Link
        href={`/subjects/${data.subject_id}`}
        className="text-sm text-blue-600 hover:underline"
      >
        ← 返回对象
      </Link>
      <article className="mt-4 bg-white rounded-lg shadow p-6">
        <h1 className="text-2xl font-bold">{data.title}</h1>
        <div className="mt-2 flex flex-wrap gap-3 text-sm text-gray-600">
          <span>状态：{data.status}</span>
          <span>严重程度：{data.severity}/5</span>
          {data.created_at && (
            <span>创建：{new Date(data.created_at).toLocaleString()}</span>
          )}
        </div>
        {data.description && (
          <div className="mt-6">
            <h2 className="font-semibold mb-2">描述</h2>
            <p className="text-gray-700 whitespace-pre-wrap">{data.description}</p>
          </div>
        )}
        {data.verdict && (
          <div className="mt-6">
            <h2 className="font-semibold mb-2">裁定</h2>
            <p className="text-gray-700 whitespace-pre-wrap">{data.verdict}</p>
          </div>
        )}
        <p className="mt-6 text-xs text-gray-400">案件 ID: {data.id}</p>
      </article>
    </div>
  )
}

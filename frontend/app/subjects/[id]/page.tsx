'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'

type Identifier = {
  platform: string
  account_type?: string
  value: string
  label?: string
  is_primary?: boolean
}

type CaseItem = {
  id: string
  title: string
  description?: string
  status: string
  severity: number
  created_at?: string
}

type Subject = {
  id: string
  display_name: string
  risk_level: number
  case_count: number
  status: string
  identifiers?: Identifier[]
  notes?: string
  created_at?: string
}

export default function SubjectDetailPage() {
  const params = useParams()
  const id = params?.id as string
  const [subject, setSubject] = useState<Subject | null>(null)
  const [cases, setCases] = useState<CaseItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!id) return
    let cancelled = false
    ;(async () => {
      setLoading(true)
      setError('')
      try {
        const [sRes, cRes] = await Promise.all([
          fetch(`/api/v1/subjects/${id}`),
          fetch(`/api/v1/subjects/${id}/cases`),
        ])
        if (!sRes.ok) {
          const body = await sRes.json().catch(() => ({}))
          throw new Error(body.error || '对象不存在')
        }
        const sData = await sRes.json()
        const cData = cRes.ok ? await cRes.json() : { cases: [] }
        if (!cancelled) {
          setSubject(sData)
          setCases(cData.cases || [])
        }
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

  if (error || !subject) {
    return (
      <div className="py-12 text-center">
        <p className="text-red-600 mb-4">{error || '未找到该对象'}</p>
        <Link href="/subjects" className="text-blue-600 hover:underline">
          返回列表
        </Link>
      </div>
    )
  }

  return (
    <div className="py-8 max-w-3xl mx-auto">
      <Link href="/subjects" className="text-sm text-blue-600 hover:underline">
        ← 返回列表
      </Link>
      <div className="mt-4 bg-white rounded-lg shadow p-6">
        <div className="flex justify-between items-start gap-4">
          <div>
            <h1 className="text-2xl font-bold">{subject.display_name}</h1>
            <p className="text-gray-500 text-sm mt-1">ID: {subject.id}</p>
          </div>
          <span
            className={`px-3 py-1 rounded-full text-sm ${
              subject.status === 'active'
                ? 'bg-red-100 text-red-800'
                : subject.status === 'cleared'
                  ? 'bg-green-100 text-green-800'
                  : 'bg-gray-100 text-gray-800'
            }`}
          >
            {subject.status}
          </span>
        </div>
        <div className="mt-4 grid grid-cols-2 gap-4 text-sm">
          <div>
            风险等级：<strong>{subject.risk_level}/5</strong>
          </div>
          <div>
            案件数：<strong>{subject.case_count}</strong>
          </div>
        </div>
        {subject.notes && (
          <p className="mt-4 text-gray-700 whitespace-pre-wrap">{subject.notes}</p>
        )}
        {subject.identifiers && subject.identifiers.length > 0 && (
          <div className="mt-6">
            <h2 className="font-semibold mb-2">关联账号</h2>
            <div className="flex flex-wrap gap-2">
              {subject.identifiers.map((id, i) => (
                <span key={i} className="bg-gray-100 px-2 py-1 rounded text-sm">
                  {id.platform}
                  {id.account_type ? `/${id.account_type}` : ''}: {id.value}
                </span>
              ))}
            </div>
          </div>
        )}
      </div>

      <div className="mt-8 bg-white rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold mb-4">公开案件</h2>
        {cases.length === 0 ? (
          <p className="text-gray-500 text-sm">暂无已公开（approved/closed）案件</p>
        ) : (
          <ul className="space-y-3">
            {cases.map((c) => (
              <li key={c.id} className="border rounded-lg p-4">
                <div className="flex justify-between gap-2">
                  <Link href={`/cases/${c.id}`} className="font-medium text-blue-600 hover:underline">
                    {c.title}
                  </Link>
                  <span className="text-xs text-gray-500">{c.status}</span>
                </div>
                {c.description && (
                  <p className="text-sm text-gray-600 mt-1 line-clamp-2">{c.description}</p>
                )}
                <p className="text-xs text-gray-400 mt-2">严重程度 {c.severity}/5</p>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

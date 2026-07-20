'use client'

import { useState, useEffect } from 'react'

export default function AdminPage() {
  const [activeTab, setActiveTab] = useState('submissions')
  const [submissions, setSubmissions] = useState<any[]>([])
  const [cases, setCases] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchData()
  }, [activeTab])

  const fetchData = async () => {
    setLoading(true)
    const token = localStorage.getItem('token')
    if (!token) {
      window.location.href = '/login'
      return
    }

    try {
      if (activeTab === 'submissions') {
        const res = await fetch('/api/submissions?page=1&page_size=20', {
          headers: { 'Authorization': `Bearer ${token}` },
        })
        const data = await res.json()
        setSubmissions(data.submissions || [])
      } else if (activeTab === 'cases') {
        const res = await fetch('/api/cases?page=1&page_size=20', {
          headers: { 'Authorization': `Bearer ${token}` },
        })
        const data = await res.json()
        setCases(data.cases || [])
      }
    } catch (error) {
      console.error('Failed to fetch data:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleReview = async (id: string, status: string) => {
    const token = localStorage.getItem('token')
    if (!token) return

    try {
      await fetch(`/api/submissions/${id}/review`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ status, review_notes: '' }),
      })
      fetchData()
    } catch (error) {
      console.error('Review failed:', error)
    }
  }

  return (
    <div className="py-8">
      <h1 className="text-3xl font-bold mb-6">管理后台</h1>

      <div className="flex gap-4 mb-6">
        <button
          onClick={() => setActiveTab('submissions')}
          className={`px-4 py-2 rounded-lg ${activeTab === 'submissions' ? 'bg-blue-600 text-white' : 'bg-gray-200'}`}
        >
          举报审核
        </button>
        <button
          onClick={() => setActiveTab('cases')}
          className={`px-4 py-2 rounded-lg ${activeTab === 'cases' ? 'bg-blue-600 text-white' : 'bg-gray-200'}`}
        >
          案件管理
        </button>
      </div>

      {loading ? (
        <div className="text-center py-8">加载中...</div>
      ) : activeTab === 'submissions' ? (
        <div className="bg-white rounded-lg shadow-md">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">原因</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">状态</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {submissions.map((sub) => (
                <tr key={sub.id}>
                  <td className="px-6 py-4">{sub.reason}</td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 rounded-full text-xs ${
                      sub.status === 'pending' ? 'bg-yellow-100 text-yellow-800' :
                      sub.status === 'approved' ? 'bg-green-100 text-green-800' :
                      'bg-red-100 text-red-800'
                    }`}>
                      {sub.status}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    {sub.status === 'pending' && (
                      <div className="flex gap-2">
                        <button
                          onClick={() => handleReview(sub.id, 'approved')}
                          className="text-green-600 hover:underline"
                        >
                          通过
                        </button>
                        <button
                          onClick={() => handleReview(sub.id, 'rejected')}
                          className="text-red-600 hover:underline"
                        >
                          驳回
                        </button>
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow-md">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">标题</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">状态</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">严重程度</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {cases.map((c) => (
                <tr key={c.id}>
                  <td className="px-6 py-4">{c.title}</td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 rounded-full text-xs ${
                      c.status === 'pending' ? 'bg-yellow-100 text-yellow-800' :
                      c.status === 'approved' ? 'bg-green-100 text-green-800' :
                      'bg-gray-100 text-gray-800'
                    }`}>
                      {c.status}
                    </span>
                  </td>
                  <td className="px-6 py-4">{c.severity}/5</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

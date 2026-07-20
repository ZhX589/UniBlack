'use client'

import { useState, useEffect } from 'react'

export default function SubjectsPage() {
  const [subjects, setSubjects] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchSubjects()
  }, [])

  const fetchSubjects = async () => {
    try {
      const res = await fetch('/api/v1/subjects?page=1&page_size=20')
      const data = await res.json()
      setSubjects(data.subjects || [])
    } catch (error) {
      console.error('Failed to fetch subjects:', error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="py-8">
      <h1 className="text-3xl font-bold mb-6">黑名单列表</h1>

      {loading ? (
        <div className="text-center py-8">加载中...</div>
      ) : subjects.length === 0 ? (
        <div className="bg-gray-50 rounded-lg p-8 text-center text-gray-500">
          暂无黑名单记录
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow-md">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">名称</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">风险等级</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">案件数</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">状态</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {subjects.map((subject) => (
                <tr key={subject.id}>
                  <td className="px-6 py-4">
                    <a href={`/subjects/${subject.id}`} className="text-blue-600 hover:underline">
                      {subject.display_name}
                    </a>
                  </td>
                  <td className="px-6 py-4">{subject.risk_level}/5</td>
                  <td className="px-6 py-4">{subject.case_count}</td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 rounded-full text-xs ${
                      subject.status === 'active' ? 'bg-red-100 text-red-800' :
                      subject.status === 'cleared' ? 'bg-green-100 text-green-800' :
                      'bg-gray-100 text-gray-800'
                    }`}>
                      {subject.status === 'active' ? '黑名单' :
                       subject.status === 'cleared' ? '已清除' : '已归档'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

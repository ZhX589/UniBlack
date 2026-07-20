'use client'

import { useState } from 'react'

export default function SearchPage() {
  const [query, setQuery] = useState('')
  const [platform, setPlatform] = useState('qq')
  const [results, setResults] = useState<any[]>([])
  const [loading, setLoading] = useState(false)

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)

    try {
      const res = await fetch(`/api/v1/search?q=${encodeURIComponent(query)}`)
      const data = await res.json()
      setResults(data.results || [])
    } catch (error) {
      console.error('Search failed:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleLookup = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)

    try {
      const res = await fetch(`/api/v1/lookup?platform=${platform}&value=${encodeURIComponent(query)}`)
      if (res.ok) {
        const data = await res.json()
        setResults([data])
      } else {
        setResults([])
      }
    } catch (error) {
      console.error('Lookup failed:', error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="py-8">
      <h1 className="text-3xl font-bold mb-6">查询黑名单</h1>

      <div className="bg-white rounded-lg shadow-md p-6 mb-6">
        <form onSubmit={handleSearch} className="flex gap-4">
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="输入QQ号、用户名、邮箱等"
            className="flex-1 border rounded-lg px-4 py-2"
          />
          <button
            type="submit"
            className="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700"
            disabled={loading}
          >
            {loading ? '搜索中...' : '搜索'}
          </button>
        </form>
      </div>

      <div className="bg-white rounded-lg shadow-md p-6 mb-6">
        <h2 className="text-xl font-semibold mb-4">精确查询</h2>
        <form onSubmit={handleLookup} className="flex gap-4">
          <select
            value={platform}
            onChange={(e) => setPlatform(e.target.value)}
            className="border rounded-lg px-4 py-2"
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
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="输入账号值"
            className="flex-1 border rounded-lg px-4 py-2"
          />
          <button
            type="submit"
            className="bg-green-600 text-white px-6 py-2 rounded-lg hover:bg-green-700"
            disabled={loading}
          >
            精确查询
          </button>
        </form>
      </div>

      {results.length > 0 && (
        <div className="bg-white rounded-lg shadow-md p-6">
          <h2 className="text-xl font-semibold mb-4">查询结果</h2>
          <div className="space-y-4">
            {results.map((subject, index) => (
              <div key={index} className="border rounded-lg p-4">
                <div className="flex justify-between items-start">
                  <div>
                    <h3 className="text-lg font-semibold">{subject.display_name}</h3>
                    <p className="text-gray-500">风险等级: {subject.risk_level}/5</p>
                    <p className="text-gray-500">案件数: {subject.case_count}</p>
                  </div>
                  <span className={`px-3 py-1 rounded-full text-sm ${
                    subject.status === 'active' ? 'bg-red-100 text-red-800' :
                    subject.status === 'cleared' ? 'bg-green-100 text-green-800' :
                    'bg-gray-100 text-gray-800'
                  }`}>
                    {subject.status === 'active' ? '黑名单' :
                     subject.status === 'cleared' ? '已清除' : '已归档'}
                  </span>
                </div>
                {subject.identifiers && (
                  <div className="mt-3">
                    <p className="text-sm text-gray-600">关联账号:</p>
                    <div className="flex flex-wrap gap-2">
                      {subject.identifiers.map((id: any, i: number) => (
                        <span key={i} className="bg-gray-100 px-2 py-1 rounded text-sm">
                          {id.platform}: {id.value}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {results.length === 0 && !loading && (
        <div className="bg-gray-50 rounded-lg p-8 text-center text-gray-500">
          未找到相关结果
        </div>
      )}
    </div>
  )
}

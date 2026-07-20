'use client'

import { useState, useEffect } from 'react'

export default function AccessListsPage() {
  const [entries, setEntries] = useState<any[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [type, setType] = useState('')
  const [loading, setLoading] = useState(true)
  const [showAdd, setShowAdd] = useState(false)
  const [newEntry, setNewEntry] = useState({ type: 'blacklist', target: 'ip', value: '', reason: '' })

  useEffect(() => {
    fetchEntries()
  }, [page, type])

  const fetchEntries = async () => {
    const token = localStorage.getItem('token')
    if (!token) {
      window.location.href = '/login'
      return
    }

    setLoading(true)
    try {
      const res = await fetch(`/api/admin/access-lists?page=${page}&page_size=20&type=${type}`, {
        headers: { 'Authorization': `Bearer ${token}` },
      })
      const data = await res.json()
      setEntries(data.entries || [])
      setTotal(data.total || 0)
    } catch (error) {
      console.error('Failed to fetch entries:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = async () => {
    const token = localStorage.getItem('token')
    if (!token) return

    try {
      await fetch('/api/admin/access-lists', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify(newEntry),
      })
      setShowAdd(false)
      setNewEntry({ type: 'blacklist', target: 'ip', value: '', reason: '' })
      fetchEntries()
    } catch (error) {
      console.error('Failed to add entry:', error)
    }
  }

  const handleDelete = async (id: string) => {
    const token = localStorage.getItem('token')
    if (!token) return

    if (!confirm('确定删除此条目？')) return

    try {
      await fetch(`/api/admin/access-lists/${id}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` },
      })
      fetchEntries()
    } catch (error) {
      console.error('Failed to delete entry:', error)
    }
  }

  return (
    <div className="py-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">名单管理</h1>
        <button
          onClick={() => setShowAdd(true)}
          className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700"
        >
          添加条目
        </button>
      </div>

      <div className="flex gap-4 mb-6">
        <button
          onClick={() => setType('')}
          className={`px-4 py-2 rounded-lg ${type === '' ? 'bg-blue-600 text-white' : 'bg-gray-200'}`}
        >
          全部
        </button>
        <button
          onClick={() => setType('whitelist')}
          className={`px-4 py-2 rounded-lg ${type === 'whitelist' ? 'bg-green-600 text-white' : 'bg-gray-200'}`}
        >
          白名单
        </button>
        <button
          onClick={() => setType('blacklist')}
          className={`px-4 py-2 rounded-lg ${type === 'blacklist' ? 'bg-red-600 text-white' : 'bg-gray-200'}`}
        >
          黑名单
        </button>
      </div>

      {showAdd && (
        <div className="bg-white rounded-lg shadow-md p-6 mb-6">
          <h2 className="text-xl font-semibold mb-4">添加条目</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-gray-700 mb-2">类型</label>
              <select
                value={newEntry.type}
                onChange={(e) => setNewEntry({ ...newEntry, type: e.target.value })}
                className="w-full border rounded-lg px-4 py-2"
              >
                <option value="blacklist">黑名单</option>
                <option value="whitelist">白名单</option>
              </select>
            </div>
            <div>
              <label className="block text-gray-700 mb-2">目标</label>
              <select
                value={newEntry.target}
                onChange={(e) => setNewEntry({ ...newEntry, target: e.target.value })}
                className="w-full border rounded-lg px-4 py-2"
              >
                <option value="ip">IP</option>
                <option value="email">邮箱</option>
                <option value="username">用户名</option>
              </select>
            </div>
            <div>
              <label className="block text-gray-700 mb-2">值</label>
              <input
                type="text"
                value={newEntry.value}
                onChange={(e) => setNewEntry({ ...newEntry, value: e.target.value })}
                className="w-full border rounded-lg px-4 py-2"
                placeholder="IP/邮箱/用户名"
              />
            </div>
            <div>
              <label className="block text-gray-700 mb-2">原因</label>
              <input
                type="text"
                value={newEntry.reason}
                onChange={(e) => setNewEntry({ ...newEntry, reason: e.target.value })}
                className="w-full border rounded-lg px-4 py-2"
                placeholder="可选"
              />
            </div>
          </div>
          <div className="flex gap-2 mt-4">
            <button
              onClick={handleAdd}
              className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700"
            >
              添加
            </button>
            <button
              onClick={() => setShowAdd(false)}
              className="bg-gray-200 px-4 py-2 rounded-lg hover:bg-gray-300"
            >
              取消
            </button>
          </div>
        </div>
      )}

      <div className="bg-white rounded-lg shadow-md">
        {loading ? (
          <div className="p-8 text-center">加载中...</div>
        ) : (
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">类型</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">目标</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">值</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">原因</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">创建时间</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {entries.map((entry) => (
                <tr key={entry.id}>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 rounded-full text-xs ${
                      entry.type === 'whitelist' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                    }`}>
                      {entry.type === 'whitelist' ? '白名单' : '黑名单'}
                    </span>
                  </td>
                  <td className="px-6 py-4">{entry.target}</td>
                  <td className="px-6 py-4 font-mono">{entry.value}</td>
                  <td className="px-6 py-4">{entry.reason || '-'}</td>
                  <td className="px-6 py-4 text-sm text-gray-500">
                    {new Date(entry.created_at).toLocaleString()}
                  </td>
                  <td className="px-6 py-4">
                    <button
                      onClick={() => handleDelete(entry.id)}
                      className="text-red-600 hover:underline"
                    >
                      删除
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        <div className="p-4 border-t flex justify-between items-center">
          <div className="text-gray-500">共 {total} 条记录</div>
          <div className="flex gap-2">
            <button
              onClick={() => setPage(Math.max(1, page - 1))}
              disabled={page === 1}
              className="px-3 py-1 border rounded disabled:opacity-50"
            >
              上一页
            </button>
            <span className="px-3 py-1">{page}</span>
            <button
              onClick={() => setPage(page + 1)}
              disabled={entries.length < 20}
              className="px-3 py-1 border rounded disabled:opacity-50"
            >
              下一页
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

'use client'

import { useState, useEffect } from 'react'

export default function AdminUsersPage() {
  const [users, setUsers] = useState<any[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchUsers()
  }, [page, search])

  const fetchUsers = async () => {
    const token = localStorage.getItem('token')
    if (!token) {
      window.location.href = '/login'
      return
    }

    setLoading(true)
    try {
      const res = await fetch(`/api/admin/users?page=${page}&page_size=20&search=${search}`, {
        headers: { 'Authorization': `Bearer ${token}` },
      })
      const data = await res.json()
      setUsers(data.users || [])
      setTotal(data.total || 0)
    } catch (error) {
      console.error('Failed to fetch users:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleToggleActive = async (userId: string, active: boolean) => {
    const token = localStorage.getItem('token')
    if (!token) return

    try {
      await fetch(`/api/admin/users/${userId}/active`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ active }),
      })
      fetchUsers()
    } catch (error) {
      console.error('Failed to toggle user active:', error)
    }
  }

  return (
    <div className="py-8">
      <h1 className="text-3xl font-bold mb-6">用户管理</h1>

      <div className="bg-white rounded-lg shadow-md">
        <div className="p-4 border-b">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="搜索用户名或邮箱..."
            className="w-full border rounded-lg px-4 py-2"
          />
        </div>

        {loading ? (
          <div className="p-8 text-center">加载中...</div>
        ) : (
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">用户名</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">邮箱</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">角色</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">状态</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">最后登录</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {users.map((user) => (
                <tr key={user.id}>
                  <td className="px-6 py-4">{user.username}</td>
                  <td className="px-6 py-4">{user.email}</td>
                  <td className="px-6 py-4">
                    <div className="flex gap-1">
                      {user.roles.map((role: string, i: number) => (
                        <span key={i} className="px-2 py-1 rounded-full text-xs bg-blue-100 text-blue-800">
                          {role}
                        </span>
                      ))}
                      {user.roles.length === 0 && (
                        <span className="text-gray-400">无角色</span>
                      )}
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 rounded-full text-xs ${
                      user.is_active ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                    }`}>
                      {user.is_active ? '活跃' : '禁用'}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500">
                    {user.last_login_at ? new Date(user.last_login_at).toLocaleString() : '从未登录'}
                  </td>
                  <td className="px-6 py-4">
                    <button
                      onClick={() => handleToggleActive(user.id, !user.is_active)}
                      className={`text-sm ${user.is_active ? 'text-red-600 hover:underline' : 'text-green-600 hover:underline'}`}
                    >
                      {user.is_active ? '禁用' : '启用'}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        <div className="p-4 border-t flex justify-between items-center">
          <div className="text-gray-500">共 {total} 个用户</div>
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
              disabled={users.length < 20}
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

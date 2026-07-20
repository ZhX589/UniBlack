'use client'

import { useState } from 'react'

export default function SubmitPage() {
  const [formData, setFormData] = useState({
    identifiers: [{ platform: 'qq', account_type: 'id', value: '' }],
    reason: '',
  })
  const [loading, setLoading] = useState(false)
  const [success, setSuccess] = useState(false)
  const [error, setError] = useState('')

  const handleIdentifierChange = (index: number, field: string, value: string) => {
    const newIdentifiers = [...formData.identifiers]
    newIdentifiers[index] = { ...newIdentifiers[index], [field]: value }
    setFormData({ ...formData, identifiers: newIdentifiers })
  }

  const addIdentifier = () => {
    setFormData({
      ...formData,
      identifiers: [...formData.identifiers, { platform: 'qq', account_type: 'id', value: '' }],
    })
  }

  const removeIdentifier = (index: number) => {
    const newIdentifiers = formData.identifiers.filter((_, i) => i !== index)
    setFormData({ ...formData, identifiers: newIdentifiers })
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    const token = localStorage.getItem('token')
    if (!token) {
      setError('请先登录')
      setLoading(false)
      return
    }

    try {
      const res = await fetch('/api/submissions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify(formData),
      })

      if (res.ok) {
        setSuccess(true)
      } else {
        const data = await res.json()
        setError(data.error || '提交失败')
      }
    } catch (error) {
      setError('提交失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }

  if (success) {
    return (
      <div className="py-8 text-center">
        <div className="bg-green-50 text-green-600 p-6 rounded-lg">
          <h2 className="text-2xl font-bold mb-2">提交成功</h2>
          <p>您的举报已提交，等待管理员审核。</p>
          <a href="/" className="text-blue-600 hover:underline mt-4 inline-block">返回首页</a>
        </div>
      </div>
    )
  }

  return (
    <div className="py-8">
      <h1 className="text-3xl font-bold mb-6">提交举报</h1>

      {error && (
        <div className="bg-red-50 text-red-600 p-3 rounded-lg mb-4">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow-md p-6">
        <div className="mb-6">
          <label className="block text-gray-700 mb-2">被举报人账号信息</label>
          {formData.identifiers.map((identifier, index) => (
            <div key={index} className="flex gap-2 mb-2">
              <select
                value={identifier.platform}
                onChange={(e) => handleIdentifierChange(index, 'platform', e.target.value)}
                className="border rounded-lg px-3 py-2"
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
              <select
                value={identifier.account_type}
                onChange={(e) => handleIdentifierChange(index, 'account_type', e.target.value)}
                className="border rounded-lg px-3 py-2"
              >
                <option value="id">ID</option>
                <option value="username">用户名</option>
                <option value="nickname">昵称</option>
              </select>
              <input
                type="text"
                value={identifier.value}
                onChange={(e) => handleIdentifierChange(index, 'value', e.target.value)}
                placeholder="账号值"
                className="flex-1 border rounded-lg px-3 py-2"
                required
              />
              {formData.identifiers.length > 1 && (
                <button
                  type="button"
                  onClick={() => removeIdentifier(index)}
                  className="text-red-600 hover:text-red-800"
                >
                  删除
                </button>
              )}
            </div>
          ))}
          <button
            type="button"
            onClick={addIdentifier}
            className="text-blue-600 hover:underline"
          >
            + 添加更多账号
          </button>
        </div>

        <div className="mb-6">
          <label className="block text-gray-700 mb-2">举报原因</label>
          <textarea
            value={formData.reason}
            onChange={(e) => setFormData({ ...formData, reason: e.target.value })}
            className="w-full border rounded-lg px-4 py-2 h-32"
            placeholder="请详细描述举报原因..."
            required
          />
        </div>

        <button
          type="submit"
          className="w-full bg-red-600 text-white py-2 rounded-lg hover:bg-red-700"
          disabled={loading}
        >
          {loading ? '提交中...' : '提交举报'}
        </button>
      </form>
    </div>
  )
}

'use client'

import { useState } from 'react'

export function DemoCaptcha({ value, onChange, purpose = 'register', token }: { value: string; onChange: (value: string) => void; purpose?: 'register' | 'submission' | 'appeal'; token?: string }) {
	const verified = value !== ''
	const [loading, setLoading] = useState(false)
	async function verify() {
		if (verified) return onChange('')
		setLoading(true)
		try {
			const path = purpose === 'submission' ? '/api/verification/demo/submission' : '/api/verification/demo/register'
			const response = await fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) } })
			const data = await response.json()
			if (response.ok) onChange(data.token)
		} finally { setLoading(false) }
	}
	return (
    <button
      type="button"
      aria-pressed={verified}
		onClick={verify}
      className={`w-full rounded-lg border p-4 text-left ${verified ? 'border-green-600 bg-green-50 text-green-800' : 'border-gray-300 bg-gray-50 text-gray-700'}`}
    >
		<strong>{verified ? '验证已完成' : loading ? '验证中...' : '我不是自动程序'}</strong>
      <span className="ml-2 text-sm">这是 UniBlack 内置演示验证，不会连接第三方服务。</span>
    </button>
  )
}

'use client'

import { useState, useEffect } from 'react'
import { adminSettingsToMap, flattenAdminSettings, parseSettingValue } from '@/lib/settings'

export default function AdminSettingsPage() {
  const [rows, setRows] = useState<Array<{ key: string; value: string }>>([])
  const [map, setMap] = useState<Record<string, unknown>>({})
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState('basic')
  const [message, setMessage] = useState('')

  useEffect(() => {
    fetchSettings()
  }, [])

  const fetchSettings = async () => {
    const token = localStorage.getItem('token')
    if (!token) {
      window.location.href = '/login'
      return
    }
    try {
      const res = await fetch('/api/admin/settings', {
        headers: { Authorization: `Bearer ${token}` },
      })
      const data = await res.json()
      const flat = flattenAdminSettings(data)
      setRows(flat)
      setMap(adminSettingsToMap(flat))
    } catch (error) {
      console.error('Failed to fetch settings:', error)
    } finally {
      setLoading(false)
    }
  }

  const displayString = (key: string, fallback = '') => {
    const v = map[key]
    if (v === undefined || v === null) return fallback
    return String(v)
  }

  const displayBool = (key: string) => {
    const v = map[key]
    if (typeof v === 'boolean') return v
    return v === true || v === 'true'
  }

  const handleSave = async (key: string, value: unknown) => {
    const token = localStorage.getItem('token')
    if (!token) return
    setMessage('')
    try {
      const res = await fetch('/api/admin/settings', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify([{ key, value }]),
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        setMessage(body.error || '保存失败')
        return
      }
      setMap((prev) => ({ ...prev, [key]: value }))
      setMessage('已保存')
    } catch {
      setMessage('保存失败')
    }
  }

  const tabs = [
    { id: 'basic', label: '基础配置' },
    { id: 'security', label: '安全配置' },
    { id: 'auth', label: '登录配置' },
  ]

  if (loading) {
    return <div className="py-8 text-center">加载中...</div>
  }

  return (
    <div className="py-8">
      <h1 className="text-3xl font-bold mb-6">系统配置</h1>
      <p className="text-sm text-gray-500 mb-4">
        邮箱验证 / 人机验证均通过接口与控制台配置，适配可复用部署（不硬编码供应商）。
      </p>
      {message && <div className="mb-4 text-sm text-green-700 bg-green-50 p-2 rounded">{message}</div>}

      <div className="flex gap-4 mb-6 flex-wrap">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`px-4 py-2 rounded-lg ${activeTab === tab.id ? 'bg-blue-600 text-white' : 'bg-gray-200'}`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="bg-white rounded-lg shadow-md p-6 space-y-4">
        {activeTab === 'basic' && (
          <>
            <h2 className="text-xl font-semibold">基础配置</h2>
            <Field
              label="项目名称"
              defaultValue={displayString('site.name', 'UniBlack')}
              onBlur={(v) => handleSave('site.name', v)}
            />
            <Field
              label="项目描述"
              textarea
              defaultValue={displayString('site.description')}
              onBlur={(v) => handleSave('site.description', v)}
            />
            <div>
              <label className="block text-gray-700 mb-2">主题色</label>
              <input
                type="color"
                defaultValue={displayString('site.theme_color', '#3B82F6')}
                onChange={(e) => handleSave('site.theme_color', e.target.value)}
                className="border rounded-lg px-2 py-1"
              />
            </div>
            <Field
              label="联系邮箱"
              defaultValue={displayString('site.contact_email')}
              onBlur={(v) => handleSave('site.contact_email', v)}
            />
          </>
        )}

        {activeTab === 'security' && (
          <>
            <h2 className="text-xl font-semibold">安全配置</h2>
            <Toggle
              label="邮箱验证"
              desc="注册时要求邮箱验证码（需配置 SMTP）"
              checked={displayBool('security.email_verification')}
              onChange={(v) => handleSave('security.email_verification', v)}
            />
            <Field
              label="SMTP Host"
              defaultValue={displayString('security.smtp_host')}
              onBlur={(v) => handleSave('security.smtp_host', v)}
            />
            <Field
              label="SMTP Port"
              defaultValue={displayString('security.smtp_port', '587')}
              onBlur={(v) => handleSave('security.smtp_port', parseInt(v, 10) || 587)}
            />
            <Field
              label="SMTP Username"
              defaultValue={displayString('security.smtp_username')}
              onBlur={(v) => handleSave('security.smtp_username', v)}
            />
            <Field
              label="SMTP Password"
              type="password"
              placeholder="留空表示不修改"
              defaultValue=""
              onBlur={(v) => {
                if (v && v !== '••••••••') handleSave('security.smtp_password', v)
              }}
            />
            <Field
              label="SMTP From"
              defaultValue={displayString('security.smtp_from')}
              onBlur={(v) => handleSave('security.smtp_from', v)}
            />
            <Toggle
              label="人机验证"
              desc="注册时要求 Turnstile / reCAPTCHA / hCaptcha"
              checked={displayBool('security.captcha_enabled')}
              onChange={(v) => handleSave('security.captcha_enabled', v)}
            />
            <div>
              <label className="block text-gray-700 mb-2">人机验证提供商</label>
              <select
                defaultValue={displayString('security.captcha_provider', 'turnstile')}
                onChange={(e) => handleSave('security.captcha_provider', e.target.value)}
                className="w-full border rounded-lg px-4 py-2"
              >
                <option value="turnstile">Cloudflare Turnstile</option>
                <option value="recaptcha">Google reCAPTCHA</option>
                <option value="hcaptcha">hCaptcha</option>
                <option value="none">无</option>
              </select>
            </div>
            <Field
              label="Captcha Site Key（公开）"
              defaultValue={displayString('security.captcha_site_key')}
              onBlur={(v) => handleSave('security.captcha_site_key', v)}
            />
            <Field
              label="Captcha Secret Key（仅服务端）"
              type="password"
              placeholder="留空表示不修改"
              defaultValue=""
              onBlur={(v) => {
                if (v && v !== '••••••••') handleSave('security.captcha_secret_key', v)
              }}
            />
            <Field
              label="公开 API 限速 (req/s)"
              defaultValue={displayString('security.rate_limit_public', '20')}
              onBlur={(v) => handleSave('security.rate_limit_public', parseInt(v, 10) || 20)}
            />
            <Field
              label="认证 API 限速 (req/s)"
              defaultValue={displayString('security.rate_limit_auth', '10')}
              onBlur={(v) => handleSave('security.rate_limit_auth', parseInt(v, 10) || 10)}
            />
          </>
        )}

        {activeTab === 'auth' && (
          <>
            <h2 className="text-xl font-semibold">登录配置</h2>
            <Toggle
              label="开放注册"
              desc="关闭后注册页显示「注册已关闭」"
              checked={displayBool('auth.registration_enabled')}
              onChange={(v) => handleSave('auth.registration_enabled', v)}
            />
            <Toggle
              label="GitHub 登录"
              desc="预留 OAuth（Client Secret 不回传前端）"
              checked={displayBool('auth.oauth_github_enabled')}
              onChange={(v) => handleSave('auth.oauth_github_enabled', v)}
            />
            <Field
              label="GitHub Client ID"
              defaultValue={displayString('auth.oauth_github_client_id')}
              onBlur={(v) => handleSave('auth.oauth_github_client_id', v)}
            />
            <Field
              label="GitHub Client Secret"
              type="password"
              placeholder="留空表示不修改"
              defaultValue=""
              onBlur={(v) => {
                if (v && v !== '••••••••') handleSave('auth.oauth_github_client_secret', v)
              }}
            />
          </>
        )}
      </div>

      <details className="mt-8 text-sm text-gray-500">
        <summary className="cursor-pointer">原始配置键列表（{rows.length}）</summary>
        <pre className="mt-2 overflow-auto bg-gray-50 p-3 rounded text-xs">
          {JSON.stringify(map, null, 2)}
        </pre>
      </details>
    </div>
  )
}

function Field({
  label,
  defaultValue,
  onBlur,
  textarea,
  type = 'text',
  placeholder,
}: {
  label: string
  defaultValue: string
  onBlur: (v: string) => void
  textarea?: boolean
  type?: string
  placeholder?: string
}) {
  return (
    <div>
      <label className="block text-gray-700 mb-2">{label}</label>
      {textarea ? (
        <textarea
          defaultValue={defaultValue}
          onBlur={(e) => onBlur(e.target.value)}
          className="w-full border rounded-lg px-4 py-2 h-24"
          placeholder={placeholder}
        />
      ) : (
        <input
          type={type}
          defaultValue={defaultValue}
          onBlur={(e) => onBlur(e.target.value)}
          className="w-full border rounded-lg px-4 py-2"
          placeholder={placeholder}
        />
      )}
    </div>
  )
}

function Toggle({
  label,
  desc,
  checked,
  onChange,
}: {
  label: string
  desc: string
  checked: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <div className="flex items-center justify-between p-4 bg-gray-50 rounded-lg">
      <div>
        <div className="font-medium">{label}</div>
        <div className="text-sm text-gray-500">{desc}</div>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          checked={checked}
          onChange={(e) => onChange(e.target.checked)}
          className="sr-only peer"
        />
        <div className="w-11 h-6 bg-gray-200 rounded-full peer peer-checked:bg-blue-600 after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
      </label>
    </div>
  )
}

'use client'

import { useState, useEffect } from 'react'

export default function AdminSettingsPage() {
  const [settings, setSettings] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState('basic')
  const [saving, setSaving] = useState(false)

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
        headers: { 'Authorization': `Bearer ${token}` },
      })
      const data = await res.json()
      setSettings(data)
    } catch (error) {
      console.error('Failed to fetch settings:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSave = async (key: string, value: any) => {
    setSaving(true)
    const token = localStorage.getItem('token')
    if (!token) return

    try {
      await fetch('/api/admin/settings', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify([{ key, value }]),
      })
    } catch (error) {
      console.error('Failed to save setting:', error)
    } finally {
      setSaving(false)
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

      <div className="flex gap-4 mb-6">
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

      <div className="bg-white rounded-lg shadow-md p-6">
        {activeTab === 'basic' && (
          <div className="space-y-4">
            <h2 className="text-xl font-semibold mb-4">基础配置</h2>
            <div>
              <label className="block text-gray-700 mb-2">项目名称</label>
              <input
                type="text"
                defaultValue={settings.find(s => s.key === 'site.name')?.value || ''}
                onBlur={(e) => handleSave('site.name', e.target.value)}
                className="w-full border rounded-lg px-4 py-2"
              />
            </div>
            <div>
              <label className="block text-gray-700 mb-2">项目描述</label>
              <textarea
                defaultValue={settings.find(s => s.key === 'site.description')?.value || ''}
                onBlur={(e) => handleSave('site.description', e.target.value)}
                className="w-full border rounded-lg px-4 py-2 h-24"
              />
            </div>
            <div>
              <label className="block text-gray-700 mb-2">主题色</label>
              <input
                type="color"
                defaultValue={settings.find(s => s.key === 'site.theme_color')?.value || '#3B82F6'}
                onChange={(e) => handleSave('site.theme_color', e.target.value)}
                className="border rounded-lg px-4 py-2"
              />
            </div>
            <div>
              <label className="block text-gray-700 mb-2">联系邮箱</label>
              <input
                type="email"
                defaultValue={settings.find(s => s.key === 'site.contact_email')?.value || ''}
                onBlur={(e) => handleSave('site.contact_email', e.target.value)}
                className="w-full border rounded-lg px-4 py-2"
              />
            </div>
          </div>
        )}

        {activeTab === 'security' && (
          <div className="space-y-4">
            <h2 className="text-xl font-semibold mb-4">安全配置</h2>
            <div className="flex items-center justify-between p-4 bg-gray-50 rounded-lg">
              <div>
                <div className="font-medium">邮箱验证</div>
                <div className="text-sm text-gray-500">注册时要求邮箱验证</div>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  defaultChecked={settings.find(s => s.key === 'security.email_verification')?.value === 'true'}
                  onChange={(e) => handleSave('security.email_verification', e.target.checked)}
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
              </label>
            </div>

            <div className="flex items-center justify-between p-4 bg-gray-50 rounded-lg">
              <div>
                <div className="font-medium">人机验证</div>
                <div className="text-sm text-gray-500">注册时要求人机验证</div>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  defaultChecked={settings.find(s => s.key === 'security.captcha_enabled')?.value === 'true'}
                  onChange={(e) => handleSave('security.captcha_enabled', e.target.checked)}
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
              </label>
            </div>

            <div>
              <label className="block text-gray-700 mb-2">人机验证提供商</label>
              <select
                defaultValue={settings.find(s => s.key === 'security.captcha_provider')?.value || 'turnstile'}
                onChange={(e) => handleSave('security.captcha_provider', e.target.value)}
                className="w-full border rounded-lg px-4 py-2"
              >
                <option value="turnstile">Cloudflare Turnstile</option>
                <option value="recaptcha">Google reCAPTCHA</option>
                <option value="hcaptcha">hCaptcha</option>
              </select>
            </div>

            <div>
              <label className="block text-gray-700 mb-2">Captcha Site Key</label>
              <input
                type="text"
                defaultValue={settings.find(s => s.key === 'security.captcha_site_key')?.value || ''}
                onBlur={(e) => handleSave('security.captcha_site_key', e.target.value)}
                className="w-full border rounded-lg px-4 py-2"
              />
            </div>

            <div>
              <label className="block text-gray-700 mb-2">公开API限速 (req/s)</label>
              <input
                type="number"
                defaultValue={settings.find(s => s.key === 'security.rate_limit_public')?.value || 20}
                onBlur={(e) => handleSave('security.rate_limit_public', parseInt(e.target.value))}
                className="w-full border rounded-lg px-4 py-2"
              />
            </div>
          </div>
        )}

        {activeTab === 'auth' && (
          <div className="space-y-4">
            <h2 className="text-xl font-semibold mb-4">登录配置</h2>
            <div className="flex items-center justify-between p-4 bg-gray-50 rounded-lg">
              <div>
                <div className="font-medium">开放注册</div>
                <div className="text-sm text-gray-500">允许新用户注册</div>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  defaultChecked={settings.find(s => s.key === 'auth.registration_enabled')?.value === 'true'}
                  onChange={(e) => handleSave('auth.registration_enabled', e.target.checked)}
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
              </label>
            </div>

            <div className="flex items-center justify-between p-4 bg-gray-50 rounded-lg">
              <div>
                <div className="font-medium">GitHub 登录</div>
                <div className="text-sm text-gray-500">允许使用 GitHub 账号登录</div>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  defaultChecked={settings.find(s => s.key === 'auth.oauth_github_enabled')?.value === 'true'}
                  onChange={(e) => handleSave('auth.oauth_github_enabled', e.target.checked)}
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
              </label>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

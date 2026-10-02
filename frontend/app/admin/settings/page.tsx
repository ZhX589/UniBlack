'use client'

import { useCallback, useEffect, useId, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/app/providers'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import { adminSettingsToMap, flattenAdminSettings } from '@/lib/settings'
import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input, Select, Textarea } from '@/components/ui/input'
import { LoadingState } from '@/components/ui/loading-state'
import { Panel } from '@/components/ui/panel'

const reservedSecretKeys = [
  'security.smtp_password',
  'security.captcha_secret_key',
  'auth.oauth_github_client_secret',
]
const secretMask = '••••••••'

type AdminSettingsResponse = {
  schema?: Array<{ key: string; secret?: boolean; type?: string }>
  values?: Record<string, unknown>
  settings?: Array<{ key: string; value: unknown; secret?: boolean }>
}

export default function AdminSettingsPage() {
  const { status } = useAuth()
  const router = useRouter()
  const [rowCount, setRowCount] = useState(0)
  // Display values never contain secrets; password drafts live only in SecretField.
  const [map, setMap] = useState<Record<string, unknown>>({})
  const [secretKeys, setSecretKeys] = useState(() => new Set(reservedSecretKeys))
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState('basic')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  const fetchSettings = useCallback(async () => {
    if (status !== 'authenticated') return
    setLoading(true)
    setError('')
    try {
      const data = await apiRequest<AdminSettingsResponse>('/api/admin/settings', { auth: true })
      const secrets = new Set(reservedSecretKeys)
      for (const meta of data?.schema ?? []) {
        if (meta.secret || meta.type === 'secret') secrets.add(meta.key)
      }
      for (const row of data?.settings ?? []) {
        if (row.secret) secrets.add(row.key)
      }
      const flat = flattenAdminSettings(data)
      const source = data?.values ?? adminSettingsToMap(flat)
      const values: Record<string, unknown> = {}
      for (const [key, value] of Object.entries(source)) {
        // Redaction metadata and masks are display-only, never editable values.
        if (value && typeof value === 'object' && 'redacted' in value) secrets.add(key)
        if (secrets.has(key) || value === secretMask) continue
        values[key] = value
      }
      setSecretKeys(secrets)
      setRowCount(flat.length)
      setMap(values)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '加载配置失败')
    } finally {
      setLoading(false)
    }
  }, [status])

  useEffect(() => {
    if (status === 'anonymous') router.replace('/login?next=/admin/settings')
  }, [status, router])

  useEffect(() => {
    void fetchSettings()
  }, [fetchSettings])

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

  const handleSave = async (key: string, value: unknown): Promise<boolean> => {
    // Keep the backend's existing no-change semantics; do not introduce clearing.
    if (secretKeys.has(key) && (value === '' || value === secretMask)) return false
    setMessage('')
    setError('')
    try {
      await apiRequest('/api/admin/settings', {
        method: 'PUT',
        auth: true,
        json: [{ key, value }],
      })
      // Never copy a submitted secret (or a response mask) into generic display state.
      if (!secretKeys.has(key)) setMap((prev) => ({ ...prev, [key]: value }))
      setMessage('已保存')
      return true
    } catch (err) {
      setMessage(err instanceof ApiError ? err.message : '保存失败')
      return false
    }
  }

  const tabs = [
    { id: 'basic', label: '基础配置' },
    { id: 'security', label: '安全配置' },
    { id: 'auth', label: '登录配置' },
  ]

  if (status === 'loading' || loading) return <LoadingState />
  if (status === 'anonymous') return null

  return (
    <div className="py-4">
      <h1 className="mb-2 text-3xl font-bold">系统配置</h1>
      <p className="mb-4 text-sm text-muted">
        邮箱验证 / 人机验证均通过接口与控制台配置。运行时当前使用演示 captcha；控制台保留 provider 契约。
      </p>
      {message && (
        <div className="mb-4">
          <Alert tone={message === '已保存' ? 'success' : 'danger'}>{message}</Alert>
        </div>
      )}
      {error && (
        <div className="mb-4">
          <Alert>{error}</Alert>
        </div>
      )}

      <div className="mb-6 flex flex-wrap gap-2">
        {tabs.map((tab) => (
          <Button key={tab.id} variant={activeTab === tab.id ? 'primary' : 'secondary'} onClick={() => setActiveTab(tab.id)}>
            {tab.label}
          </Button>
        ))}
      </div>

      <Panel className="space-y-4">
        {activeTab === 'basic' && (
          <>
            <h2 className="text-xl font-semibold">基础配置</h2>
            <Field label="项目名称" defaultValue={displayString('site.name', 'UniBlack')} onBlur={(v) => handleSave('site.name', v)} />
            <Field
              label="项目描述"
              textarea
              defaultValue={displayString('site.description')}
              onBlur={(v) => handleSave('site.description', v)}
            />
            <label className="block text-sm">
              主题色
              <Input
                type="color"
                defaultValue={displayString('site.theme_color', '#2563EB')}
                onChange={(e) => handleSave('site.theme_color', e.target.value)}
                className="mt-1 block w-auto rounded border border-black/10 px-2 py-1"
              />
            </label>
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
            <Field label="SMTP Host" defaultValue={displayString('security.smtp_host')} onBlur={(v) => handleSave('security.smtp_host', v)} />
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
            <SecretField label="SMTP Password" onSave={(v) => handleSave('security.smtp_password', v)} />
            <Field label="SMTP From" defaultValue={displayString('security.smtp_from')} onBlur={(v) => handleSave('security.smtp_from', v)} />
            <Toggle
              label="人机验证"
              desc="演示模式下不会调用第三方；保留 provider 配置契约"
              checked={displayBool('security.captcha_enabled')}
              onChange={(v) => handleSave('security.captcha_enabled', v)}
            />
            <label className="block text-sm">
              人机验证提供商
              <Select
                defaultValue={displayString('security.captcha_provider', 'turnstile')}
                onChange={(e) => handleSave('security.captcha_provider', e.target.value)}
              >
                <option value="turnstile">Cloudflare Turnstile</option>
                <option value="recaptcha">Google reCAPTCHA</option>
                <option value="hcaptcha">hCaptcha</option>
                <option value="none">无</option>
              </Select>
            </label>
            <Field
              label="Captcha Site Key（公开）"
              defaultValue={displayString('security.captcha_site_key')}
              onBlur={(v) => handleSave('security.captcha_site_key', v)}
            />
            <SecretField label="Captcha Secret Key（仅服务端）" onSave={(v) => handleSave('security.captcha_secret_key', v)} />
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
            <SecretField label="GitHub Client Secret" onSave={(v) => handleSave('auth.oauth_github_client_secret', v)} />
          </>
        )}
      </Panel>

      <details className="mt-8 text-sm text-muted">
        <summary className="cursor-pointer">原始配置键列表（{rowCount}）</summary>
        <pre className="mt-2 overflow-auto rounded bg-background p-3 text-xs">{JSON.stringify(map, null, 2)}</pre>
      </details>
    </div>
  )
}

function SecretField({ label, onSave }: { label: string; onSave: (value: string) => Promise<boolean> }) {
  const [draft, setDraft] = useState('')
  const [saving, setSaving] = useState(false)

  const save = async () => {
    if (!draft || draft === secretMask || saving) return
    setSaving(true)
    try {
      if (await onSave(draft)) setDraft('')
    } finally {
      setSaving(false)
    }
  }

  return (
    <label className="block text-sm">
      {label}
      <Input
        type="password"
        value={draft}
        placeholder="留空表示不修改"
        autoComplete="new-password"
        disabled={saving}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={() => void save()}
      />
    </label>
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
    <label className="block text-sm">
      {label}
      {textarea ? (
        <Textarea
          defaultValue={defaultValue}
          onBlur={(e) => onBlur(e.target.value)}
          placeholder={placeholder}
          className="h-24"
        />
      ) : (
        <Input
          type={type}
          defaultValue={defaultValue}
          onBlur={(e) => onBlur(e.target.value)}
          placeholder={placeholder}
        />
      )}
    </label>
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
  const id = useId()
  return (
    <div className="flex items-center justify-between rounded bg-background p-4">
      <div>
        <div id={`${id}-label`} className="font-medium">{label}</div>
        <div id={`${id}-description`} className="text-sm text-muted">{desc}</div>
      </div>
      <label className="relative inline-flex min-h-touch cursor-pointer items-center">
        <input type="checkbox" aria-labelledby={`${id}-label`} aria-describedby={`${id}-description`} checked={checked} onChange={(e) => onChange(e.target.checked)} className="peer sr-only" />
        <div className="h-6 w-11 rounded-full bg-border after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:bg-white after:transition-all peer-checked:bg-primary peer-checked:after:translate-x-full peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2" />
      </label>
    </div>
  )
}

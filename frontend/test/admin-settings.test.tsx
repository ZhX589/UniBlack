import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AdminSettingsPage from '@/app/admin/settings/page'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'

vi.mock('next/navigation', () => ({ useRouter: () => ({ replace: vi.fn() }) }))
vi.mock('@/app/providers', () => ({ useAuth: () => ({ status: 'authenticated' }) }))
vi.mock('@/lib/api', () => ({ apiRequest: vi.fn() }))

const mask = '••••••••'
const secretKeys = ['security.smtp_password', 'security.captcha_secret_key', 'auth.oauth_github_client_secret']
const settings = () => ({
  schema: secretKeys.map((key) => ({ key, type: 'secret', secret: true })),
  values: {
    'site.name': 'UniBlack',
    'security.email_verification': false,
    'security.captcha_enabled': false,
    'security.captcha_site_key': 'public-site-key',
    'auth.registration_enabled': true,
    'auth.oauth_github_enabled': false,
    ...Object.fromEntries(secretKeys.map((key) => [key, { configured: true, redacted: true }])),
  },
  settings: secretKeys.map((key) => ({ key, value: JSON.stringify(mask), secret: true, configured: true })),
})

beforeEach(() => {
  vi.mocked(apiRequest).mockImplementation(async (_path, options) => options?.method === 'PUT'
    ? { values: Object.fromEntries(secretKeys.map((key) => [key, mask])) }
    : settings())
})
afterEach(() => { cleanup(); vi.resetAllMocks() })

async function openTab(tab: string) {
  render(<AdminSettingsPage />)
  fireEvent.click(await screen.findByRole('button', { name: tab }))
}
function rawConfig() { return document.querySelector('pre')!.textContent! }
function puts() { return vi.mocked(apiRequest).mock.calls.filter(([, options]) => options?.method === 'PUT') }

describe('admin settings secret isolation', () => {
  it.each([
    ['安全配置', 'SMTP Password', 'security.smtp_password'],
    ['安全配置', 'Captcha Secret Key（仅服务端）', 'security.captcha_secret_key'],
    ['登录配置', 'GitHub Client Secret', 'auth.oauth_github_client_secret'],
  ])('saves %s / %s without exposing the submitted secret or response mask', async (tab, label, key) => {
    await openTab(tab)
    const input = screen.getByLabelText(label)
    expect(input).toHaveValue('')
    const secret = `test-secret-${key}`
    fireEvent.change(input, { target: { value: secret } })
    fireEvent.blur(input)
    await screen.findByText('已保存')
    expect(puts()).toHaveLength(1)
    expect(puts()[0][1]).toMatchObject({ auth: true, json: [{ key, value: secret }] })
    expect(input).toHaveValue('')
    expect(document.body.textContent).not.toContain(secret)
    expect(document.body.innerHTML).not.toContain(secret)
    expect(screen.queryByDisplayValue(secret)).not.toBeInTheDocument()
    expect(rawConfig()).not.toContain(key)
    expect(rawConfig()).not.toContain(mask)
    fireEvent.blur(input)
    expect(puts()).toHaveLength(1)
    fireEvent.click(screen.getByRole('button', { name: tab === '安全配置' ? '登录配置' : '安全配置' }))
    fireEvent.click(screen.getByRole('button', { name: tab }))
    expect(screen.getByLabelText(label)).toHaveValue('')
  })

  it('preserves empty/mask no-change semantics and permits replacement, not clearing', async () => {
    await openTab('安全配置')
    const input = screen.getByLabelText('SMTP Password')
    fireEvent.blur(input)
    fireEvent.change(input, { target: { value: mask } })
    fireEvent.blur(input)
    fireEvent.change(input, { target: { value: '' } })
    fireEvent.blur(input)
    expect(puts()).toHaveLength(0)
    fireEvent.change(input, { target: { value: 'replacement' } })
    fireEvent.blur(input)
    await screen.findByText('已保存')
    expect(input).toHaveValue('')
    expect(puts()[0][1]?.json).toEqual([{ key: 'security.smtp_password', value: 'replacement' }])
  })

  it('retains only the isolated draft on failed save and permits retry', async () => {
    await openTab('安全配置')
    vi.mocked(apiRequest).mockRejectedValueOnce(new ApiError(500, '保存失败测试'))
    const input = screen.getByLabelText('SMTP Password')
    fireEvent.change(input, { target: { value: 'retry-secret' } })
    fireEvent.blur(input)
    await screen.findByText('保存失败测试')
    expect(input).toHaveValue('retry-secret')
    expect(rawConfig()).not.toContain('retry-secret')
    fireEvent.blur(input)
    await screen.findByText('已保存')
    expect(input).toHaveValue('')
  })

  it('excludes schema/row secrets, redaction metadata and reserved secrets from initial JSON', async () => {
    const data = settings()
    data.schema.push({ key: 'custom.schema_secret', secret: true, type: 'string' })
    const values = {
      ...data.values,
      ...Object.fromEntries(secretKeys.map((key) => [key, `server-secret-${key}`])),
      'custom.schema_secret': 'schema-secret',
      'custom.type_secret': 'type-secret',
      'custom.row_secret': 'row-secret',
      'custom.redacted': { configured: true, redacted: true },
    }
    vi.mocked(apiRequest).mockResolvedValueOnce({ ...data, values,
      schema: [...data.schema, { key: 'custom.type_secret', type: 'secret' }],
      settings: [...data.settings, { key: 'custom.row_secret', value: 'row-secret', secret: true }],
    })
    await openTab('安全配置')
    expect(rawConfig()).toContain('public-site-key')
    expect(rawConfig()).not.toMatch(/server-secret|schema-secret|type-secret|row-secret|redacted|configured/)
    expect(screen.getByLabelText('SMTP Password')).toHaveValue('')
    expect(screen.getByLabelText('Captcha Secret Key（仅服务端）')).toHaveValue('')
  })

  it('filters reserved secrets from legacy settings rows without schema metadata', async () => {
    vi.mocked(apiRequest).mockResolvedValueOnce({ settings: [
      { key: 'site.name', value: '"Legacy"' },
      ...secretKeys.map((key) => ({ key, value: '"legacy-secret"' })),
    ] })
    await openTab('安全配置')
    expect(rawConfig()).toContain('Legacy')
    expect(rawConfig()).not.toContain('legacy-secret')
    for (const key of secretKeys) expect(rawConfig()).not.toContain(key)
  })
})

describe('admin settings accessible toggles', () => {
  it('finds security checkboxes by Chinese name and exposes descriptions', async () => {
    await openTab('安全配置')
    const email = screen.getByRole('checkbox', { name: '邮箱验证' })
    expect(email).toHaveAccessibleDescription('注册时要求邮箱验证码（需配置 SMTP）')
    const captcha = screen.getByRole('checkbox', { name: '人机验证' })
    expect(captcha).toHaveAccessibleDescription('演示模式下不会调用第三方；保留 provider 配置契约')
    fireEvent.click(email)
    await waitFor(() => expect(email).toBeChecked())
    expect(puts()[0][1]?.json).toEqual([{ key: 'security.email_verification', value: true }])
    expect(rawConfig()).toContain('"security.email_verification": true')
    fireEvent.click(screen.getByRole('button', { name: '登录配置' }))
    expect(screen.getByRole('checkbox', { name: '开放注册' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'GitHub 登录' })).not.toBeChecked()
  })
})

import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import SubmitPage from '@/app/submit/page'
import { apiRequest } from '@/lib/api'

vi.mock('@/app/providers', () => ({ useAuth: () => ({ status: 'authenticated', user: { email: 'owner@example.test' } }) }))
vi.mock('@/lib/api', () => ({ apiRequest: vi.fn() }))
vi.mock('@/components/auth/demo-captcha', () => ({ DemoCaptcha: () => null }))

beforeEach(() => {
  vi.mocked(apiRequest).mockResolvedValue({ 'security.email_verification': true, 'security.captcha_enabled': false })
})
afterEach(() => { cleanup(); vi.resetAllMocks() })

it('associates dynamic Chinese labels with unique focusable controls without submitting on add', async () => {
  render(<SubmitPage />)
  expect(await screen.findByLabelText('邮箱验证码')).toBeInTheDocument()
  const firstUsername = screen.getByLabelText('用户名 1')
  fireEvent.change(firstUsername, { target: { value: 'first-account' } })
  const accountButton = screen.getByRole('button', { name: '添加账号' })
  expect(accountButton).toHaveAttribute('type', 'button')
  fireEvent.click(accountButton)
  const eventButton = screen.getByRole('button', { name: '添加事件' })
  expect(eventButton).toHaveAttribute('type', 'button')
  fireEvent.click(eventButton)
  for (const n of [1, 2]) {
    for (const label of ['平台', '用户名', '账号 ID', '事件标题', '事件详情', '严重级别', '文本证据', '链接标题', '链接 URL', '文件证据']) {
      const control = screen.getByLabelText(`${label} ${n}`)
      expect(control.id).not.toBe('')
      expect(document.querySelectorAll(`[id="${control.id}"]`)).toHaveLength(1)
      control.focus()
      expect(control).toHaveFocus()
    }
  }
  expect(firstUsername).toHaveValue('first-account')
  expect(screen.getByLabelText('用户名 2')).toHaveValue('')
  expect(vi.mocked(apiRequest).mock.calls.every(([, options]) => !options?.method || options.method === 'GET')).toBe(true)
})

import { test, expect, type Page } from '@playwright/test'
import { expectNoHorizontalOverflow } from './helpers'

function fakeJwt(): string {
  const header = btoa(JSON.stringify({ alg: 'none', typ: 'JWT' })).replace(/=/g, '')
  const payload = btoa(
    JSON.stringify({
      user_id: 'user-1',
      username: 'reporter',
      email: 'reporter@example.test',
      roles: ['reporter'],
      exp: Math.floor(Date.now() / 1000) + 3600,
    }),
  ).replace(/=/g, '')
  return `${header}.${payload}.sig`
}

async function setupAuthAndSettings(page: Page) {
  await page.addInitScript((token) => {
    localStorage.setItem('token', token)
    localStorage.setItem('refresh_token', 'refresh')
  }, fakeJwt())
  await page.route('**/api/settings/public', (route) =>
    route.fulfill({
      json: { 'security.email_verification': true, 'security.captcha_enabled': true },
    }),
  )
  await page.route('**/api/verification/demo/submission', (route) =>
    route.fulfill({ json: { token: 'demo-captcha-token' } }),
  )
}

// ── (a) Anonymous /submit ──────────────────────────────────────────────

test('anonymous /submit shows login guidance, no active publish controls', async ({ page }) => {
  await page.route('**/api/settings/public', (route) =>
    route.fulfill({ json: {} }),
  )
  await page.goto('/submit')
  await expect(page.getByRole('heading', { name: '提交对象与事件' })).toBeVisible()
  await expect(page.getByRole('button', { name: '登录后提交' })).toBeVisible()
  await expect(page.getByRole('button', { name: '确认并公开发布' })).toHaveCount(0)
  await expect(page.locator('input[type="file"]')).toHaveCount(0)
  await expectNoHorizontalOverflow(page)
})

// ── (b) Authenticated task flow ────────────────────────────────────────

test('authenticated task flow: three sections, completion badges, no-evidence text', async ({ page }) => {
  await setupAuthAndSettings(page)
  await page.goto('/submit')

  // Wait for authenticated view.
  await expect(page.getByRole('heading', { name: '1 对象账号' })).toBeVisible()
  await expect(page.getByRole('heading', { name: '2 事件事实' })).toBeVisible()
  await expect(page.getByRole('heading', { name: '3 证据与确认' })).toBeVisible()

  // Wait for settings to load (captcha button appears).
  await page.waitForSelector('button:has-text("我不是自动程序")', { timeout: 15000 })

  // Three badges with 待补充 initially.
  await expect(page.getByText('待补充')).toHaveCount(3)

  // Fill platform + username → accounts section completes.
  await page.locator('[placeholder="用户名"]').fill('alice')
  await expect(page.locator('#publish-section-accounts span.inline-flex').getByText('已完成', { exact: true })).toBeVisible()

  // Fill event title + details + severity → facts section completes.
  await page.getByPlaceholder('事件标题').fill('fraud incident')
  await page.getByPlaceholder('事件详情').fill('scammed payment')
  await page.getByLabel(/严重级别/).selectOption('4')
  await expect(page.locator('#publish-section-facts span.inline-flex').getByText('已完成', { exact: true })).toBeVisible()

  // Fill verification code → evidence-confirmation verification step done.
  await page.getByPlaceholder('邮箱验证码').fill('123456')
  const captchaResponse = page.waitForResponse('**/api/verification/demo/submission', { timeout: 15000 })
  await page.getByRole('button', { name: '我不是自动程序' }).click()
  await captchaResponse
  await expect(page.locator('#publish-section-evidence-confirmation span.inline-flex').getByText('已完成', { exact: true })).toBeVisible({ timeout: 10000 })

  // No-evidence recommendation text visible.
  await expect(page.getByText(/当前未附证据/)).toBeVisible()

  await expectNoHorizontalOverflow(page)
})

// ── (c) Client validation ──────────────────────────────────────────────

test('client validation: no publish request, focus first broken section, values preserved', async ({ page }) => {
  let publishRequested = false
  await page.route('**/api/settings/public', (route) =>
    route.fulfill({
      json: { 'security.email_verification': true, 'security.captcha_enabled': true },
    }),
  )
  await page.route('**/api/verification/demo/submission', (route) =>
    route.fulfill({ json: { token: 'demo-captcha-token' } }),
  )
  await page.route('**/api/subjects/publish', (route) => {
    publishRequested = true
    route.fulfill({ json: { public_id: 'IGNORED' } })
  })

  await page.addInitScript((token) => {
    localStorage.setItem('token', token)
    localStorage.setItem('refresh_token', 'token')
  }, fakeJwt())

  await page.goto('/submit')
  await expect(page.getByRole('heading', { name: '1 对象账号' })).toBeVisible()
  await page.waitForSelector('button:has-text("我不是自动程序")', { timeout: 15000 })

  // Fill display name, event title, details, verification code, captcha.
  await page.getByPlaceholder('留空时使用第一条账号用户名').fill('Scam Person')
  await page.getByPlaceholder('事件标题').fill('bad event')
  await page.getByPlaceholder('事件详情').fill('did bad things')
  await page.getByLabel(/严重级别/).selectOption('2')
  await page.getByPlaceholder('邮箱验证码').fill('123456')
  const captchaResponse = page.waitForResponse('**/api/verification/demo/submission', { timeout: 15000 })
  await page.getByRole('button', { name: '我不是自动程序' }).click()
  await captchaResponse

  // Clear the default platform to make accounts invalid.
  await page.locator('[aria-label="平台 1"]').clear()
  // Now accounts are invalid → click publish.
  await page.getByRole('button', { name: '确认并公开发布' }).click()

  // No request to /api/subjects/publish was made.
  await page.waitForTimeout(500)
  expect(publishRequested).toBe(false)

  // Focus moved to publish-section-accounts.
  await page.waitForFunction(() => document.activeElement?.id === 'publish-section-accounts')

  // All previously filled values are preserved.
  await expect(page.getByPlaceholder('留空时使用第一条账号用户名')).toHaveValue('Scam Person')
  await expect(page.getByPlaceholder('事件标题')).toHaveValue('bad event')
  await expect(page.getByPlaceholder('事件详情')).toHaveValue('did bad things')
  await expect(page.getByPlaceholder('邮箱验证码')).toHaveValue('123456')

  await expectNoHorizontalOverflow(page)
})

// ── (d) JSON server failure ────────────────────────────────────────────

test('json server failure: error visible, focus on evidence section, values preserved, button restores', async ({ page }) => {
  await page.route('**/api/settings/public', (route) =>
    route.fulfill({
      json: { 'security.email_verification': true, 'security.captcha_enabled': true },
    }),
  )
  await page.route('**/api/verification/demo/submission', (route) =>
    route.fulfill({ json: { token: 'demo-captcha-token' } }),
  )
  let publishRequested = false
  await page.route('**/api/subjects/publish', (route) => {
    publishRequested = true
    route.fulfill({ status: 400, json: { error: 'invalid verification code' } })
  })

  await page.addInitScript((token) => {
    localStorage.setItem('token', token)
    localStorage.setItem('refresh_token', 'token')
  }, fakeJwt())

  await page.goto('/submit')
  await expect(page.getByRole('heading', { name: '1 对象账号' })).toBeVisible()
  await page.waitForSelector('button:has-text("我不是自动程序")', { timeout: 15000 })

  // Fill everything validly (no file → JSON path).
  await page.locator('[placeholder="用户名"]').fill('alice')
  await page.getByPlaceholder('账号 ID').fill('123')
  await page.getByPlaceholder('留空时使用第一条账号用户名').fill('Scam X')
  await page.getByPlaceholder('事件标题').fill('fraud')
  await page.getByPlaceholder('事件详情').fill('details here')
  await page.getByLabel(/严重级别/).selectOption('3')
  await page.getByPlaceholder('邮箱验证码').fill('123456')
  const captchaResponse = page.waitForResponse('**/api/verification/demo/submission', { timeout: 15000 })
  await page.getByRole('button', { name: '我不是自动程序' }).click()
  await captchaResponse

  const publishResponse = page.waitForResponse('**/api/subjects/publish', { timeout: 15000 })
  await page.getByRole('button', { name: '确认并公开发布' }).click()
  await publishResponse

  // Error visible.
  await expect(page.getByText('invalid verification code')).toBeVisible()

  // Focus moves to evidence-confirmation section.
  await page.waitForFunction(() => document.activeElement?.id === 'publish-section-evidence-confirmation')

  // All form values preserved.
  await expect(page.getByPlaceholder('留空时使用第一条账号用户名')).toHaveValue('Scam X')
  await expect(page.getByPlaceholder('事件标题')).toHaveValue('fraud')
  await expect(page.getByPlaceholder('事件详情')).toHaveValue('details here')
  await expect(page.getByPlaceholder('邮箱验证码')).toHaveValue('123456')

  // Button text restores and is re-enabled.
  const btn = page.getByRole('button', { name: '确认并公开发布' })
  await expect(btn).toBeVisible()
  await expect(btn).toBeEnabled()

  expect(publishRequested).toBe(true)
  await expectNoHorizontalOverflow(page)
})

// ── (e) Multipart success ──────────────────────────────────────────────

test('multipart publish: file upload shows success page with public ID and link', async ({ page }) => {
  let publishRequested = false
  await page.route('**/api/settings/public', (route) =>
    route.fulfill({
      json: { 'security.email_verification': true, 'security.captcha_enabled': true },
    }),
  )
  await page.route('**/api/verification/demo/submission', (route) =>
    route.fulfill({ json: { token: 'demo-captcha-token' } }),
  )
  await page.route('**/api/subjects/publish', (route) => {
    publishRequested = true
    route.fulfill({ json: { public_id: 'UBS_TEST001' } })
  })

  await page.addInitScript((token) => {
    localStorage.setItem('token', token)
    localStorage.setItem('refresh_token', 'token')
  }, fakeJwt())

  await page.goto('/submit')
  await expect(page.getByRole('heading', { name: '1 对象账号' })).toBeVisible()
  await page.waitForSelector('button:has-text("我不是自动程序")', { timeout: 15000 })

  // Fill all required fields.
  await page.locator('[placeholder="用户名"]').fill('bob')
  await page.getByPlaceholder('事件标题').fill('file test')
  await page.getByPlaceholder('事件详情').fill('test with file evidence')
  await page.getByLabel(/严重级别/).selectOption('2')
  await page.getByPlaceholder('邮箱验证码').fill('123456')
  const captchaResponse = page.waitForResponse('**/api/verification/demo/submission', { timeout: 15000 })
  await page.getByRole('button', { name: '我不是自动程序' }).click()
  await captchaResponse

  // Attach a file.
  const fileInput = page.locator('input[type="file"]').first()
  await fileInput.setInputFiles({
    name: 'record.txt',
    mimeType: 'text/plain',
    buffer: Buffer.from('evidence content'),
  })

  const publishResponse = page.waitForResponse('**/api/subjects/publish', { timeout: 15000 })
  await page.getByRole('button', { name: '确认并公开发布' }).click()
  await publishResponse

  expect(publishRequested).toBe(true)

  // Success page shows public ID.
  await expect(page.getByText('UBS_TEST001')).toBeVisible()
  await expect(page.getByRole('link', { name: '查看对象档案' })).toBeVisible()
  await expect(page.getByRole('link', { name: '查看对象档案' })).toHaveAttribute('href', '/subjects/UBS_TEST001')

  await expectNoHorizontalOverflow(page)
})

// ── (f) Captcha failure ────────────────────────────────────────────────

test('captcha failure: error in evidence section, inputs preserved, button re-enables', async ({ page }) => {
  await page.route('**/api/settings/public', (route) =>
    route.fulfill({
      json: { 'security.email_verification': true, 'security.captcha_enabled': true },
    }),
  )
  // Captcha returns 400.
  await page.route('**/api/verification/demo/submission', (route) =>
    route.fulfill({ status: 400, json: { message: 'captcha verification failed' } }),
  )

  await page.addInitScript((token) => {
    localStorage.setItem('token', token)
    localStorage.setItem('refresh_token', 'token')
  }, fakeJwt())

  await page.goto('/submit')
  await expect(page.getByRole('heading', { name: '1 对象账号' })).toBeVisible()
  await page.waitForSelector('button:has-text("我不是自动程序")', { timeout: 15000 })

  // Fill accounts and events.
  await page.locator('[placeholder="用户名"]').fill('eve')
  await page.getByPlaceholder('留空时使用第一条账号用户名').fill('Eve Smith')
  await page.getByPlaceholder('事件标题').fill('captcha fail')
  await page.getByPlaceholder('事件详情').fill('test captcha failure')
  await page.getByLabel(/严重级别/).selectOption('1')
  await page.getByPlaceholder('邮箱验证码').fill('123456')

  const captchaResponse = page.waitForResponse('**/api/verification/demo/submission', { timeout: 15000 })
  await page.getByRole('button', { name: '我不是自动程序' }).click()
  await captchaResponse

  // Error visible in evidence-confirmation section.
  await expect(page.getByText('captcha verification failed').first()).toBeVisible()

  // Previously filled values retained.
  await expect(page.getByPlaceholder('留空时使用第一条账号用户名')).toHaveValue('Eve Smith')
  await expect(page.getByPlaceholder('事件标题')).toHaveValue('captcha fail')
  await expect(page.getByPlaceholder('邮箱验证码')).toHaveValue('123456')

  // Captcha button re-enables.
  const captchaBtn = page.getByRole('button', { name: '我不是自动程序' })
  await expect(captchaBtn).toBeEnabled()

  await expectNoHorizontalOverflow(page)
})

import { test, expect } from '@playwright/test'
import { expectNoAdminLink, loginAs } from './helpers'

const adminUser = process.env.E2E_ADMIN || 'admin'
const adminPassword = process.env.E2E_ADMIN_PASSWORD || 'admin123'
const normalUser = process.env.E2E_USER || 'testuser'
const normalPassword = process.env.E2E_USER_PASSWORD || 'password123'

test('normal user does not see admin link after login', async ({ page }) => {
  test.skip(!process.env.E2E_USER && !process.env.E2E_ALLOW_DEFAULT_USERS, 'set E2E_USER or E2E_ALLOW_DEFAULT_USERS=1')
  await loginAs(page, normalUser, normalPassword)
  await expectNoAdminLink(page)
  await expect(page.getByRole('link', { name: '我的处罚' })).toBeVisible()
})

test('admin sees management navigation and settings', async ({ page }) => {
  test.skip(!process.env.E2E_ADMIN && !process.env.E2E_ALLOW_DEFAULT_USERS, 'set E2E_ADMIN or E2E_ALLOW_DEFAULT_USERS=1')
  await loginAs(page, adminUser, adminPassword)
  await expect(page.getByRole('link', { name: '管理' })).toBeVisible()
  await page.goto('/admin')
  await expect(page.getByRole('link', { name: '站点与配置' })).toBeVisible()
  await expect(page.getByText(/兼容窗口|内容治理|旧举报|旧案件/)).toBeVisible()
})

test('login and logout update shell without hard redirect artifacts', async ({ page }) => {
  test.skip(!process.env.E2E_ADMIN && !process.env.E2E_ALLOW_DEFAULT_USERS, 'set E2E_ADMIN or E2E_ALLOW_DEFAULT_USERS=1')
  await loginAs(page, adminUser, adminPassword)
  await page.getByRole('button', { name: '退出' }).click()
  await expect(page.getByRole('link', { name: '登录' })).toBeVisible()
  await expectNoAdminLink(page)
})

import { test, expect } from '@playwright/test'
import { loginAs } from './helpers'

const adminUser = process.env.E2E_ADMIN || 'admin'
const adminPassword = process.env.E2E_ADMIN_PASSWORD || 'admin123'

test('legacy case detail soft-redirects toward event route', async ({ page }) => {
  const legacyId = '00000000-0000-0000-0000-000000000001'
  await page.goto(`/cases/${legacyId}`)
  await expect(page).toHaveURL(new RegExp(`/events/${legacyId}`))
})

test('event detail shows appeal-related UI chrome without admin nav for guests', async ({ page }) => {
  await page.goto('/events/does-not-exist')
  await expect(page.getByRole('link', { name: '管理', exact: true })).toHaveCount(0)
  await expect(page.locator('body')).toContainText(/不存在|未公开|加载失败|事件/i)
})

test('authenticated admin appeal queue lists status filter', async ({ page }) => {
  test.skip(!process.env.E2E_ADMIN && !process.env.E2E_ALLOW_DEFAULT_USERS, 'set E2E_ADMIN or E2E_ALLOW_DEFAULT_USERS=1')
  await loginAs(page, adminUser, adminPassword)
  await page.goto('/admin/appeals')
  await expect(page.getByRole('heading', { name: '事件申诉队列' })).toBeVisible()
  const statusSelect = page.locator('select')
  await expect(statusSelect).toBeVisible()
  await statusSelect.selectOption('')
  await expect(page.getByText(/当前筛选下无申诉|理由|事件/)).toBeVisible()
})

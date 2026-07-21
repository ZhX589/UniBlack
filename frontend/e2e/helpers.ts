import { expect, type Page } from '@playwright/test'

export async function loginAs(page: Page, username: string, password: string) {
  await page.goto('/login')
  await page.getByLabel('用户名').fill(username)
  await page.getByLabel('密码').fill(password)
  await page.getByRole('button', { name: '登录' }).click()
  await expect(page.getByRole('button', { name: '退出' })).toBeVisible({ timeout: 15000 })
}

export async function expectNoAdminLink(page: Page) {
  await expect(page.getByRole('link', { name: '管理', exact: true })).toHaveCount(0)
}

export async function expectNoHorizontalOverflow(page: Page) {
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1,
  )
  expect(overflow).toBeFalsy()
}

export const viewports = {
  mobile: { width: 375, height: 812 },
  tablet: { width: 768, height: 1024 },
  desktop: { width: 1280, height: 800 },
}

import { test, expect } from '@playwright/test'
import { expectNoHorizontalOverflow, viewports } from './helpers'

for (const [name, size] of Object.entries(viewports)) {
  test(`public home and search usable at ${name}`, async ({ page }) => {
    await page.setViewportSize(size)
    await page.goto('/')
    await expect(page.getByRole('heading').first()).toBeVisible()
    await expect(page.getByLabel(/核验账号/i)).toBeVisible()
    await expectNoHorizontalOverflow(page)

    await page.goto('/search')
    await expect(page.getByRole('heading', { name: '查询黑名单' })).toBeVisible()
    await expect(page.getByLabel(/搜索关键词/i)).toBeVisible()
    await expectNoHorizontalOverflow(page)
  })
}

test('anonymous visitor cannot see management navigation', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('link', { name: '管理' })).toHaveCount(0)
  await expect(page.getByRole('link', { name: '查询' })).toBeVisible()
})

import { test, expect } from '@playwright/test'
import { expectNoHorizontalOverflow, viewports } from './helpers'

// Deterministic fixtures used across the trustworthy profile scenarios.

const SUBJECT_WITH_ACCOUNTS_AND_EVENTS = {
  id: 'UBS_PROFILE_TEST',
  public_id: 'UBS_PROFILE_TEST',
  display_name: '示例对象',
  status: 'active',
  risk_level: 4,
  accounts: [
    { id: 'acc-1', platform: 'github', username: 'alice', is_primary: true },
    { id: 'acc-2', platform: 'twitter', username: '', account_id: 'tw-99' },
    { id: 'acc-3', platform: 'discord', username: '   ', account_id: '   ' },
  ],
  identifiers: [
    { platform: 'legacy', value: 'legacy-001' },
    { platform: 'empty', value: '' },
  ],
  events: [
    {
      id: 'event-1',
      title: '事件标题甲',
      status: 'published',
      severity: 3,
      details: '已公开事件的事实摘要。',
      created_at: '2026-07-20T00:00:00Z',
    },
    {
      id: 'event-2',
      title: '事件标题乙',
      status: 'corrected',
      severity: 4,
      details: '已更正的事件说明。',
      created_at: '2026-07-22T00:00:00Z',
    },
  ],
}

const SUBJECT_WITHOUT_EVENTS = {
  id: 'UBS_PROFILE_EMPTY',
  public_id: 'UBS_PROFILE_EMPTY',
  display_name: '空白对象',
  status: 'cleared',
  risk_level: 0,
  accounts: [{ id: 'acc-1', platform: 'github', username: 'nobody' }],
  identifiers: [],
  events: [],
}

const EVENT_WITH_MIXED_EVIDENCE = {
  id: 'event-1',
  title: '混合证据事件',
  status: 'published',
  severity: 2,
  details: '第一行事实\n第二行事实',
  created_at: '2026-07-20T00:00:00Z',
  evidence: [
    {
      id: 'ev-link',
      type: 'link',
      title: '来源报道',
      description: '第三方报道链接',
      url: 'https://example.test/report',
    },
    {
      id: 'ev-file-url',
      type: 'file',
      title: '存档文件',
      description: '带有可访问 URL 的文件',
      url: 'https://example.test/doc.pdf',
      original_filename: 'doc.pdf',
      file_size: 1024,
      mime_type: 'application/pdf',
    },
    {
      id: 'ev-file-no-url',
      type: 'file',
      title: '无链接文件',
      description: '没有可访问 URL 的文件证据',
      url: null,
      original_filename: 'secret.txt',
      file_size: 256,
      mime_type: 'text/plain',
    },
    {
      id: 'ev-unknown',
      type: 'custom_type',
      title: '自定义类型',
      description: '未知证据类型',
      url: null,
    },
  ],
}

const EVENT_WITHOUT_EVIDENCE = {
  id: 'event-1',
  title: '无证据事件',
  status: 'published',
  severity: 1,
  details: '事件事实描述。',
  created_at: '2026-07-20T00:00:00Z',
  evidence: [],
}

// Fake JWT token so the AuthProvider reports status='authenticated'. The
// providers.tsx decodeToken helper only inspects user_id / exp / roles, so a
// minimal payload is enough — no signature verification happens client-side.
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

// Subject profile ----------------------------------------------------------

for (const [name, size] of Object.entries(viewports)) {
  test(`subject profile renders archive hierarchy at ${name}`, async ({ page }) => {
    await page.route('**/api/v1/subjects/UBS_PROFILE_TEST', (route) =>
      route.fulfill({ json: SUBJECT_WITH_ACCOUNTS_AND_EVENTS }),
    )

    await page.setViewportSize(size)
    await page.goto('/subjects/UBS_PROFILE_TEST')

    await expect(page.getByRole('heading', { name: '对象档案' })).toBeVisible()
    await expect(page.getByText('UBS_PROFILE_TEST')).toBeVisible()
    // Risk must carry both text label and numeric scale (header row only).
    await expect(page.getByText('风险等级：较高 4/5')).toBeVisible()
    // Empty-valued account is filtered; primary account is labelled but never "verified".
    await expect(page.getByText('主标识')).toBeVisible()
    await expect(page.getByText('已验证')).toHaveCount(0)
    // First identifier is shown and labelled as legacy compatibility.
    await expect(page.getByText('兼容账号记录')).toBeVisible()
    await expect(page.getByText('legacy-001')).toBeVisible()
    // Event trajectory entry links into event detail.
    await expect(
      page.getByRole('link', { name: '查看事件与证据' }).first(),
    ).toHaveAttribute('href', '/events/event-1')
    await expectNoHorizontalOverflow(page)
  })
}

test('subject without events does not imply safe and avoids absolute safety wording', async ({
  page,
}) => {
  await page.route('**/api/v1/subjects/UBS_PROFILE_TEST', (route) =>
    route.fulfill({ json: SUBJECT_WITHOUT_EVENTS }),
  )

  await page.goto('/subjects/UBS_PROFILE_TEST')
  await expect(page.getByText('不代表对该对象作出绝对安全判断')).toBeVisible()
  await expect(page.getByText('安全对象')).toHaveCount(0)
  await expectNoHorizontalOverflow(page)
})

// Event detail -------------------------------------------------------------

test('event with mixed evidence types renders four layers and respects url rules', async ({
  page,
}) => {
  await page.route('**/api/v1/events/event-1', (route) =>
    route.fulfill({ json: EVENT_WITH_MIXED_EVIDENCE }),
  )

  await page.goto('/events/event-1')

  await expect(page.getByRole('heading', { name: '事实描述' })).toBeVisible()
  await expect(page.getByRole('heading', { name: '证据材料' })).toBeVisible()
  await expect(page.getByRole('heading', { name: '申诉与纠错' })).toBeVisible()

  // External link evidence opens in a new tab without referrer.
  const sourceLink = page.getByRole('link', { name: '打开来源链接' })
  await expect(sourceLink).toHaveAttribute('target', '_blank')
  await expect(sourceLink).toHaveAttribute('rel', 'noreferrer')

  // File evidence without a real URL must NOT render an action link.
  await expect(page.getByRole('link', { name: '打开来源链接' })).toHaveCount(1)

  // Unknown evidence type maps to the shared "其他材料" label.
  await expect(page.getByText('其他材料（custom_type）')).toBeVisible()

  // Community record note is present alongside the facts.
  await expect(page.getByText(/社区公开记录/)).toBeVisible()

  await expectNoHorizontalOverflow(page)
})

for (const [name, size] of Object.entries(viewports)) {
  test(`event with empty evidence shows restrained empty state at ${name}`, async ({ page }) => {
    await page.route('**/api/v1/events/event-1', (route) =>
      route.fulfill({ json: EVENT_WITHOUT_EVIDENCE }),
    )

    await page.setViewportSize(size)
    await page.goto('/events/event-1')
    await expect(
      page.getByText('当前公开响应中没有可展示的证据材料'),
    ).toBeVisible()
    await expectNoHorizontalOverflow(page)
  })
}

test('appeal submission failure preserves reason and shows danger feedback', async ({ page }) => {
  await page.route('**/api/v1/events/event-1', (route) =>
    route.fulfill({ json: EVENT_WITHOUT_EVIDENCE }),
  )
  await page.route('**/api/events/event-1/appeals', (route) =>
    route.fulfill({ status: 400, json: { message: '原因不能为空或申诉已被处理' } }),
  )

  await page.addInitScript((token) => {
    localStorage.setItem('token', token)
    localStorage.setItem('refresh_token', 'refresh')
  }, fakeJwt())

  await page.goto('/events/event-1')
  await page.getByLabel(/申诉理由/).fill('保留这段申诉内容')
  await page.getByRole('button', { name: /提交事件申诉/ }).click()

  // Reason must be preserved on failure.
  await expect(page.getByLabel(/申诉理由/)).toHaveValue('保留这段申诉内容')
  // Danger feedback must be visible (exclude Next.js route-announcer).
  const dangerAlert = page
    .getByRole('alert')
    .filter({ hasText: /原因不能为空或申诉已被处理/ })
  await expect(dangerAlert).toBeVisible()
  await expectNoHorizontalOverflow(page)
})

// Error recovery -----------------------------------------------------------

test('subject detail 500 shows recoverable error state and back link', async ({ page }) => {
  await page.route('**/api/v1/subjects/UBS_PROFILE_TEST', (route) =>
    route.fulfill({ status: 500, json: { message: '服务暂时不可用' } }),
  )

  await page.goto('/subjects/UBS_PROFILE_TEST')
  await expect(page.getByText(/服务暂时不可用|加载失败/)).toBeVisible()
  await expect(page.getByRole('link', { name: /返回列表/ })).toBeVisible()
  await expectNoHorizontalOverflow(page)
})

test('event detail 500 shows recoverable error state', async ({ page }) => {
  await page.route('**/api/v1/events/event-1', (route) =>
    route.fulfill({ status: 500, json: { message: '服务暂时不可用' } }),
  )

  await page.goto('/events/event-1')
  await expect(page.getByText(/服务暂时不可用|加载失败/)).toBeVisible()
  await expectNoHorizontalOverflow(page)
})

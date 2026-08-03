import { test, expect } from '@playwright/test'
import { loginAs, expectNoHorizontalOverflow } from './helpers'

const adminUser = process.env.E2E_ADMIN || 'admin'
const adminPassword = process.env.E2E_ADMIN_PASSWORD || 'admin123'

test('legacy case detail soft-redirects toward event route', async ({ page }) => {
  const legacyId = '00000000-0000-0000-0000-000000000001'
  await page.goto(`/cases/${legacyId}`)
  await expect(page).toHaveURL(new RegExp(`/events/${legacyId}`))
})

test('event detail shows appeal-related UI chrome without admin nav for guests', async ({ page }) => {
  // Route-mock the event API so the guest scenario is deterministic and does
  // not depend on a live backend returning a localized 404 message.
  await page.route('**/api/v1/events/does-not-exist', (route) =>
    route.fulfill({ status: 404, json: { message: '事件不存在或未公开' } }),
  )
  await page.goto('/events/does-not-exist')
  await expect(page.getByRole('link', { name: '管理', exact: true })).toHaveCount(0)
  await expect(page.locator('body')).toContainText(/不存在|未公开|加载失败|事件/i)
  await expectNoHorizontalOverflow(page)
})

// --- Governance workbench (route-mocked) --------------------------------------
//
// The legacy admin queue test required a real auth/backend. It is replaced by
// the deterministic scenarios below, which inject a fake JWT via addInitScript
// and stub `/api/appeals` responses via a single page.route handler per glob
// (Playwright routes a request through handlers in registration order and the
// first handler that calls route.fulfill wins — so each test registers exactly
// one handler on `**/api/appeals?**` and branches internally on the query
// string, plus a separate handler for the resolve POST).

// Minimal fake JWT so the AuthProvider reports status='authenticated'. The
// providers.tsx decodeToken helper only inspects user_id / exp / roles, so a
// minimal payload is enough — no signature verification happens client-side.
function fakeAdminJwt(): string {
  const header = btoa(JSON.stringify({ alg: 'none', typ: 'JWT' })).replace(/=/g, '')
  const payload = btoa(
    JSON.stringify({
      user_id: 'admin-1',
      username: 'admin',
      email: 'admin@example.test',
      roles: ['admin'],
      exp: Math.floor(Date.now() / 1000) + 3600,
    }),
  ).replace(/=/g, '')
  return `${header}.${payload}.sig`
}

async function injectAdminAuth(page: import('@playwright/test').Page) {
  await page.addInitScript((token) => {
    localStorage.setItem('token', token)
    localStorage.setItem('refresh_token', 'refresh')
  }, fakeAdminJwt())
}

const PENDING_EVENT_APPEAL = {
  id: 'appeal-pending',
  event_id: 'event-pending',
  case_id: null,
  reason: '申诉理由甲\n第二行申诉理由',
  status: 'pending',
  outcome: null,
  resolution_reason: null,
  submitted_by: 'user-1',
  created_at: '2026-07-20T00:00:00Z',
}

const ADJUDICATED_EVENT_APPEAL = {
  id: 'appeal-done',
  event_id: 'event-done',
  case_id: null,
  reason: '申诉理由乙',
  status: 'approved',
  outcome: 'corrected',
  resolution_reason: '经核实，事实字段有误，予以更正。',
  submitted_by: 'user-2',
  created_at: '2026-07-21T00:00:00Z',
}

const LEGACY_CASE_APPEAL = {
  id: 'appeal-legacy',
  event_id: null,
  case_id: 'case-legacy-1',
  reason: '历史事项申诉',
  status: 'approved',
  outcome: 'upheld',
  resolution_reason: '维持原记录',
  submitted_by: 'user-3',
  created_at: '2026-07-19T00:00:00Z',
}

const WORKBENCH_LIST = {
  appeals: [PENDING_EVENT_APPEAL, ADJUDICATED_EVENT_APPEAL, LEGACY_CASE_APPEAL],
  total: 3,
  page: 1,
  page_size: 50,
}

// Pending-count probe response: exactly one pending appeal exists.
const PENDING_COUNT_ONE = { appeals: [PENDING_EVENT_APPEAL], total: 1, page: 1, page_size: 1 }

// After the moderator resolves the pending appeal, the list reloads with the
// item flipped to adjudicated state.
const RESOLVED_LIST = {
  appeals: [
    { ...PENDING_EVENT_APPEAL, status: 'approved', outcome: 'corrected', resolution_reason: '事实字段有误' },
    ADJUDICATED_EVENT_APPEAL,
    LEGACY_CASE_APPEAL,
  ],
  total: 3,
  page: 1,
  page_size: 50,
}

const EMPTY_LIST = { appeals: [], total: 0, page: 1, page_size: 50 }

// Shared helper: branch the list-route mock on the `status` + `page_size`
// query params. `listFixture` is the full-list payload returned for the main
// list request; the pending-count probe (`status=pending&page_size=1`) always
// returns PENDING_COUNT_ONE unless overridden via `pendingFixture`.
function listRoute(
  page: import('@playwright/test').Page,
  opts: {
    listFixture: typeof WORKBENCH_LIST | typeof EMPTY_LIST | typeof RESOLVED_LIST
    pendingFixture?: typeof PENDING_COUNT_ONE
    listStatus?: string // when set, only return listFixture for this status
    errorFirstList?: boolean // first main list call returns 500
  },
) {
  let mainListCalls = 0
  page.route('**/api/appeals?**', (route) => {
    const params = new URL(route.request().url()).searchParams
    const status = params.get('status') || ''
    const pageSize = params.get('page_size')
    // Pending-count probe: status=pending&page_size=1.
    if (status === 'pending' && pageSize === '1') {
      return route.fulfill({ json: opts.pendingFixture ?? PENDING_COUNT_ONE })
    }
    mainListCalls += 1
    // First main list call may be coerced to a 500 for the error test.
    if (opts.errorFirstList && mainListCalls === 1) {
      return route.fulfill({ status: 500, json: { message: '服务暂时不可用' } })
    }
    // When listStatus is set, only return listFixture for that status.
    if (opts.listStatus !== undefined && status !== opts.listStatus) {
      return route.fulfill({ json: EMPTY_LIST })
    }
    return route.fulfill({ json: opts.listFixture })
  })
}

test('governance workbench renders scan-first layout for mixed appeals', async ({ page }) => {
  await injectAdminAuth(page)
  listRoute(page, { listFixture: WORKBENCH_LIST })

  await page.goto('/admin/appeals')

  await expect(page.getByRole('heading', { name: '事件申诉工作台' })).toBeVisible()
  // Principle text.
  await expect(page.getByText('逐项核对申诉理由和关联事件，不进行批量裁决。')).toBeVisible()
  // Pending count expressed in the header.
  await expect(page.getByText('待处理 1 项')).toBeVisible()
  // Filter + refresh controls.
  await expect(page.locator('select')).toBeVisible()
  await expect(page.getByRole('button', { name: '刷新队列' })).toBeVisible()
  // Event ID link points at the event route.
  const eventLink = page.getByRole('link', { name: 'event-pending' })
  await expect(eventLink).toHaveAttribute('href', '/events/event-pending')
  // Translated status with raw enum preserved for audit. Scope to the list so
  // the assertion ignores the matching <option> in the filter <select>. Both
  // adjudicated rows share the approved badge, so use .first() to stay strict.
  const appealsList = page.getByRole('list')
  await expect(appealsList.getByText('待处理（pending）')).toBeVisible()
  await expect(appealsList.getByText('已通过（approved）').first()).toBeVisible()
  // Full reason preserved (whitespace-pre-wrap renders the newline).
  await expect(page.getByText('申诉理由甲\n第二行申诉理由')).toBeVisible()
  // Translated outcome with raw enum + resolution reason on adjudicated item.
  await expect(page.getByText('已更正记录（corrected）')).toBeVisible()
  await expect(page.getByText('经核实，事实字段有误，予以更正。')).toBeVisible()
  // Legacy Case target visible but unlinked (no href).
  await expect(page.getByText(/case-legacy-1/).first()).toBeVisible()
  await expect(page.getByText(/兼容事项/)).toBeVisible()
  // Four single-item resolve actions on the pending row.
  await expect(page.getByRole('button', { name: /维持（upheld）/ })).toBeVisible()
  await expect(page.getByRole('button', { name: /更正记录（corrected）/ })).toBeVisible()
  await expect(page.getByRole('button', { name: /撤回（withdrawn）/ })).toBeVisible()
  await expect(page.getByRole('button', { name: /恶意提交（malicious_submission）/ })).toBeVisible()
  // No batch controls.
  await expect(page.getByRole('checkbox')).toHaveCount(0)
  await expect(page.getByText('全选')).toHaveCount(0)
  await expect(page.getByRole('button', { name: /批量裁决/ })).toHaveCount(0)
  await expectNoHorizontalOverflow(page)
})

test('governance workbench filter to empty shows empty state and keeps pending count', async ({ page }) => {
  await injectAdminAuth(page)
  // The workbench defaults to the all-status filter, so the initial list shows
  // WORKBENCH_LIST. After selecting `已通过（approved）` we expect the empty
  // fixture for the approved-filtered main list.
  let mainListCalls = 0
  await page.route('**/api/appeals?**', (route) => {
    const params = new URL(route.request().url()).searchParams
    const status = params.get('status') || ''
    const pageSize = params.get('page_size')
    if (status === 'pending' && pageSize === '1') {
      return route.fulfill({ json: PENDING_COUNT_ONE })
    }
    mainListCalls += 1
    // First main list call (all-status) returns the full workbench list; later
    // main list calls (approved-filtered) return empty.
    if (mainListCalls === 1) return route.fulfill({ json: WORKBENCH_LIST })
    return route.fulfill({ json: EMPTY_LIST })
  })

  await page.goto('/admin/appeals')
  await expect(page.getByRole('heading', { name: '事件申诉工作台' })).toBeVisible()

  const select = page.locator('select')
  await select.selectOption('approved')

  await expect(page.getByText('当前筛选下无申诉')).toBeVisible()
  // Pending count still derived from the status=pending&page_size=1 probe.
  await expect(page.getByText('待处理 1 项')).toBeVisible()
  await expectNoHorizontalOverflow(page)
})

test('governance workbench prompt resolve submits outcome+reason and reloads', async ({ page }) => {
  await injectAdminAuth(page)

  let resolveCall: { outcome: string; reason: string } | null = null
  let mainListCalls = 0
  let resolved = false

  // Single handler for all list requests: main list returns WORKBENCH first,
  // then RESOLVED_LIST after the flag flips; the pending-count probe returns 1.
  await page.route('**/api/appeals?**', (route) => {
    const params = new URL(route.request().url()).searchParams
    const status = params.get('status') || ''
    const pageSize = params.get('page_size')
    if (status === 'pending' && pageSize === '1') {
      return route.fulfill({ json: PENDING_COUNT_ONE })
    }
    mainListCalls += 1
    return route.fulfill({ json: resolved ? RESOLVED_LIST : WORKBENCH_LIST })
  })

  await page.route('**/api/appeals/appeal-pending/review/resolve', async (route) => {
    const body = route.request().postDataJSON() as { outcome: string; reason: string }
    resolveCall = body
    await route.fulfill({ json: { ok: true } })
  })

  await page.goto('/admin/appeals')

  page.once('dialog', (dialog) => dialog.accept('事实字段有误'))

  await page.getByRole('button', { name: /更正记录（corrected）/ }).click()

  // Request body carries outcome + trimmed reason.
  await expect.poll(() => resolveCall).toEqual({ outcome: 'corrected', reason: '事实字段有误' })

  // Reload shows the adjudicated item: action buttons are gone and a success
  // Alert is visible.
  resolved = true
  // Expect at least one more main list call after resolving (the reload).
  await expect.poll(() => mainListCalls).toBeGreaterThan(1)
  await expect(page.getByText('已更正记录（corrected）').first()).toBeVisible()
  await expect(page.getByRole('button', { name: /更正记录（corrected）/ })).toHaveCount(0)
  await expect(page.getByRole('alert').filter({ hasText: /已裁决/ })).toBeVisible()
  await expectNoHorizontalOverflow(page)
})

test('governance workbench prompt dismiss sends no resolve request', async ({ page }) => {
  await injectAdminAuth(page)

  let resolveAttempts = 0

  await page.route('**/api/appeals?**', (route) => {
    const params = new URL(route.request().url()).searchParams
    const status = params.get('status') || ''
    const pageSize = params.get('page_size')
    if (status === 'pending' && pageSize === '1') {
      return route.fulfill({ json: PENDING_COUNT_ONE })
    }
    return route.fulfill({ json: WORKBENCH_LIST })
  })

  await page.route('**/api/appeals/*/review/resolve', async (route) => {
    resolveAttempts += 1
    await route.fulfill({ json: { ok: true } })
  })

  await page.goto('/admin/appeals')

  // Dismiss the native prompt — no resolve request should be sent.
  page.once('dialog', (dialog) => dialog.dismiss())
  await page.getByRole('button', { name: /恶意提交（malicious_submission）/ }).click()

  // Give the (absent) request a beat to land; it never should.
  await page.waitForTimeout(300)
  expect(resolveAttempts).toBe(0)
  // Item is still pending, action buttons remain.
  await expect(page.getByRole('button', { name: /更正记录（corrected）/ })).toBeVisible()
  await expectNoHorizontalOverflow(page)
})

test('governance workbench 500 error shows recoverable error state and refreshes', async ({ page }) => {
  await injectAdminAuth(page)

  let resolutionEnabled = false
  await page.route('**/api/appeals?**', (route) => {
    const params = new URL(route.request().url()).searchParams
    const status = params.get('status') || ''
    const pageSize = params.get('page_size')
    if (status === 'pending' && pageSize === '1') {
      // Pending-count probe also fails on first load; once the flag is set, it
      // succeeds so the refresh can repopulate the pending count.
      if (resolutionEnabled) return route.fulfill({ json: PENDING_COUNT_ONE })
      return route.fulfill({ status: 500, json: { message: '服务暂时不可用' } })
    }
    // Main list.
    if (resolutionEnabled) return route.fulfill({ json: WORKBENCH_LIST })
    return route.fulfill({ status: 500, json: { message: '服务暂时不可用' } })
  })

  await page.goto('/admin/appeals')

  // Recoverable error text is visible.
  await expect(page.getByText(/服务暂时不可用|加载失败/)).toBeVisible()
  // Refresh button is available after error.
  const refresh = page.getByRole('button', { name: '刷新队列' })
  await expect(refresh).toBeVisible()

  // Enable responses then click refresh.
  resolutionEnabled = true
  await refresh.click()

  // Second list call succeeds and renders items.
  await expect(page.getByRole('heading', { name: '事件申诉工作台' })).toBeVisible()
  await expect(page.getByText('申诉理由甲')).toBeVisible()
  await expectNoHorizontalOverflow(page)
})

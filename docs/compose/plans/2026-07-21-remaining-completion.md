# UniBlack Remaining Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use compose:subagent (recommended) or compose:execute to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `feature/next-development` 上完成最后 14–18% 收尾：提交已迁移页面、清零页面级 token/fetch、补齐 Playwright 角色与视口矩阵、跑通生产 Compose/Nginx smoke，并用证据更新进度台账。

**Architecture:** 不重做后端主路径；以前端共享 `apiRequest/apiBlob + Auth/Settings providers + navigation registry` 为唯一边界，先提交并验证当前未提交迁移，再收口两个残留管理页，然后用本地前后端进程跑 Playwright，最后在 Docker daemon 可用时完成生产 smoke。

**Tech Stack:** Next.js 14、React 18、TypeScript strict、Vitest、Playwright、Go/Echo、PostgreSQL、MinIO、Docker Compose、Nginx。

## Global Constraints

- 工作区固定：`/data/Projects/UniBlack/.worktrees/next-development`，分支 `feature/next-development`。
- 页面不得直接 `localStorage.getItem('token')`、拼接 `Authorization`、使用 `window.location` 登录跳转；统一走 `apiRequest` / `apiBlob` 与 `useAuth` / `router.replace`。
- 旧 Case/Submission 只允许兼容展示，不得新增依赖；Event 是公开主模型。
- 进度台账只有在实际测试/构建/smoke 证据后才能标“已验证”。
- Docker smoke 需要本机 Docker daemon；当前环境可能不可用，作为独立阻塞项处理，不阻塞前两个包。

---

## Completion Snapshot (2026-07-21)

### Overall

| 范围 | 完成度 | 说明 |
| --- | ---: | --- |
| 相对本轮“完成这些规划”目标 | **约 82–86%** | 核心后端 + 前端主路径已落地；剩页面收口、浏览器矩阵、部署 smoke |
| 后端 Phase 13 / 安全边界 | **约 95%** | 已提交并本地 `go test ./...` + build 通过 |
| 前端共享基础（API/导航/token） | **约 80–85%** | 已提交基础；多数页面已迁但未全部提交 |
| 浏览器验收 | **约 10%** | 仅有 Playwright 配置，无业务 e2e |
| 生产部署验收 | **约 40%** | Compose/Nginx/Dockerfile 有；缺真实 smoke |

### Already committed on `feature/next-development`

1. `a798f55` — Event/Case 弃用、S3、生产启动/访问控制、Event 治理闭环  
2. `7bbebe3` — 退休历史读取、Account-first 搜索、迁移兼容测试  
3. `26cc65a` — 前端 API client / navigation registry / Vitest  
4. `60b5f8b` — design tokens、首页/搜索/详情/登录 Event 化、CI 门禁  

### Uncommitted but already implemented (must land in Package A)

- `frontend/lib/api.ts`（含 `apiBlob` / FormData）
- `frontend/components/auth/demo-captcha.tsx`
- `frontend/app/submit/page.tsx`
- `frontend/app/subjects/page.tsx`
- `frontend/app/register/page.tsx`
- `frontend/app/setup/page.tsx`
- `frontend/app/sanctions/page.tsx`
- `frontend/app/admin/page.tsx`（兼容窗口说明 + 旧 Submission/Case）
- `frontend/app/admin/users/page.tsx`
- `frontend/app/admin/sanctions/page.tsx`
- `frontend/app/admin/archives/page.tsx`

### Still using page-level token/fetch (Package A residual)

- `frontend/app/admin/access-lists/page.tsx`
- `frontend/app/admin/settings/page.tsx`

### Known environment blockers

- Docker daemon 当前不可用（`docker info` 失败）→ Package C 可能 BLOCKED
- Playwright 业务场景尚未编写 → Package B
- 工作树 `.env.production` 可能不存在；根仓库有模板可参考

---

## Package Map

```text
Package A: 清零页面级 API / 提交已迁移页面 / 兼容 UI 收口
  -> Package B: Playwright 角色 + 375/768/1280 验收
  -> Package C: 生产 Compose/Nginx smoke + 文档台账
```

A 不依赖 Docker。  
B 可用本地 `go run` + `npm run dev`，不强制 Docker。  
C 强依赖 Docker daemon。

---

### Task 1: 提交并验证已迁移页面（Package A-1）

**Covers:** remaining-frontend-api-migration

**Files:**
- Modify (already dirty): `frontend/lib/api.ts`
- Modify: `frontend/components/auth/demo-captcha.tsx`
- Modify: `frontend/app/submit/page.tsx`
- Modify: `frontend/app/subjects/page.tsx`
- Modify: `frontend/app/register/page.tsx`
- Modify: `frontend/app/setup/page.tsx`
- Modify: `frontend/app/sanctions/page.tsx`
- Modify: `frontend/app/admin/page.tsx`
- Modify: `frontend/app/admin/users/page.tsx`
- Modify: `frontend/app/admin/sanctions/page.tsx`
- Modify: `frontend/app/admin/archives/page.tsx`
- Test: `frontend/lib/api.test.ts`

**Interfaces:**
- Consumes: `apiRequest<T>()`, `apiBlob()`, `configureApiClient()`, `useAuth()`, `useSite()`
- Produces: all migrated pages use shared client; no page-level Bearer headers in these files

- [ ] **Step 1: Freeze and scan remaining direct auth/API usage**

```bash
cd /data/Projects/UniBlack/.worktrees/next-development
rg -n "localStorage\.getItem\('token'\)|window\.location|fetch\(" frontend/app frontend/components -g'*.{ts,tsx}' | rg -v "providers\.tsx"
```

Expected: only `admin/access-lists` and `admin/settings` remain for Package A-2.

- [ ] **Step 2: Run current frontend verification on dirty tree**

```bash
cd frontend
npm run test:run
npm run typecheck
npm run lint
npm run build
```

Expected: tests pass; lint may still warn on access-lists hook deps until Task 2; build passes.

- [ ] **Step 3: Commit the already-migrated pages only**

```bash
git add frontend/lib/api.ts \
  frontend/components/auth/demo-captcha.tsx \
  frontend/app/submit/page.tsx \
  frontend/app/subjects/page.tsx \
  frontend/app/register/page.tsx \
  frontend/app/setup/page.tsx \
  frontend/app/sanctions/page.tsx \
  frontend/app/admin/page.tsx \
  frontend/app/admin/users/page.tsx \
  frontend/app/admin/sanctions/page.tsx \
  frontend/app/admin/archives/page.tsx
git commit -m "refactor: migrate remaining public and admin pages to shared API"
```

- [ ] **Step 4: Confirm admin page documents legacy compatibility window**

In `frontend/app/admin/page.tsx`, ensure UI copy states:
- new content uses `/submit` Event publish
- listed Submission/Case tables are compatibility-window only
- link to Event/sanction/archive governance entries

---

### Task 2: 迁移 access-lists 与 settings（Package A-2）

**Covers:** remaining-frontend-api-migration

**Files:**
- Modify: `frontend/app/admin/access-lists/page.tsx`
- Modify: `frontend/app/admin/settings/page.tsx`
- Optional: `frontend/components/ui/button.tsx`, `panel.tsx`, `alert.tsx` (reuse only)

**Interfaces:**
- Consumes: `apiRequest`, `useAuth`, `useRouter`
- Produces: zero page-level token/fetch outside providers

- [ ] **Step 1: Write a minimal failing grep gate for residual direct auth**

```bash
# This command should still match access-lists/settings before migration.
rg -n "localStorage\.getItem\('token'\)" frontend/app -g'*.tsx'
```

Expected: matches in `access-lists/page.tsx` and `settings/page.tsx`.

- [ ] **Step 2: Rewrite `admin/access-lists/page.tsx` with shared client**

Use this structure (minimal behavior-preserving rewrite):

```tsx
'use client'

import { useCallback, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/app/providers'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import type { AccessListEntry } from '@/lib/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { ErrorState } from '@/components/ui/error-state'
import { LoadingState } from '@/components/ui/loading-state'
import { Panel } from '@/components/ui/panel'

export default function AccessListsPage() {
  const { status } = useAuth()
  const router = useRouter()
  const [entries, setEntries] = useState<AccessListEntry[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [type, setType] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showAdd, setShowAdd] = useState(false)
  const [newEntry, setNewEntry] = useState({ type: 'blacklist', target: 'ip', value: '', reason: '' })

  const fetchEntries = useCallback(async () => {
    if (status !== 'authenticated') return
    setLoading(true)
    setError('')
    try {
      const data = await apiRequest<{ entries?: AccessListEntry[]; total?: number }>(
        `/api/admin/access-lists?page=${page}&page_size=20&type=${encodeURIComponent(type)}`,
        { auth: true },
      )
      setEntries(data.entries || [])
      setTotal(data.total || 0)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '加载名单失败')
    } finally {
      setLoading(false)
    }
  }, [page, status, type])

  useEffect(() => {
    if (status === 'anonymous') router.replace('/login?next=/admin/access-lists')
  }, [status, router])

  useEffect(() => {
    void fetchEntries()
  }, [fetchEntries])

  async function handleAdd() {
    try {
      await apiRequest('/api/admin/access-lists', { auth: true, json: newEntry })
      setShowAdd(false)
      setNewEntry({ type: 'blacklist', target: 'ip', value: '', reason: '' })
      await fetchEntries()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '添加失败')
    }
  }

  async function handleDelete(id: string) {
    if (!window.confirm('确定删除此条目？')) return
    try {
      await apiRequest(`/api/admin/access-lists/${id}`, { method: 'DELETE', auth: true })
      await fetchEntries()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '删除失败')
    }
  }

  if (status === 'loading') return <LoadingState />
  if (status === 'anonymous') return null

  return (
    <div className="py-4">
      {/* keep existing filter/add/table UX, but use Button/Panel/Badge/ErrorState */}
    </div>
  )
}
```

Preserve existing fields: type filter, add form (type/target/value/reason), table columns, pagination.

- [ ] **Step 3: Rewrite `admin/settings/page.tsx` with shared client**

Key replacements only:

```tsx
import { useRouter } from 'next/navigation'
import { useAuth } from '@/app/providers'
import { apiRequest } from '@/lib/api'
import { ApiError } from '@/lib/api-error'

// load
const data = await apiRequest<any>('/api/admin/settings', { auth: true })

// save
await apiRequest('/api/admin/settings', {
  method: 'PUT',
  auth: true,
  json: [{ key, value }],
})

// auth redirect
useEffect(() => {
  if (status === 'anonymous') router.replace('/login?next=/admin/settings')
}, [status, router])
```

Keep existing Field/Toggle UI and NewAPI-style `values/settings/schema` parsing via `lib/settings.ts`.

- [ ] **Step 4: Grep gate must pass**

```bash
rg -n "localStorage\.getItem\('token'\)|window\.location\.href\s*=\s*'/login" frontend/app -g'*.tsx'
```

Expected: no matches.

```bash
rg -n "fetch\(" frontend/app -g'*.tsx' | rg -v "providers\.tsx" || true
```

Expected: no page-level fetch matches.

- [ ] **Step 5: Verify and commit**

```bash
cd frontend
npm run test:run && npm run typecheck && npm run lint && npm run build
git add frontend/app/admin/access-lists/page.tsx frontend/app/admin/settings/page.tsx
git commit -m "refactor: migrate access-lists and settings to shared API client"
```

Expected: lint no longer reports access-lists exhaustive-deps from old pattern; build exit 0.

---

### Task 3: Playwright 角色与视口矩阵（Package B）

**Covers:** remaining-browser-acceptance

**Files:**
- Modify: `frontend/playwright.config.ts`
- Create: `frontend/e2e/navigation.spec.ts`
- Create: `frontend/e2e/public-search.spec.ts`
- Create: `frontend/e2e/auth-shell.spec.ts`
- Create: `frontend/e2e/helpers.ts`
- Modify: `.github/workflows/ci.yml` (optional e2e job if stable)
- Modify: `frontend/package.json` only if script missing

**Interfaces:**
- Consumes: running frontend at `PLAYWRIGHT_BASE_URL` (default `http://127.0.0.1:3000`)
- Consumes: backend at rewrite target (default `http://localhost:8080`)
- Produces: green e2e for anonymous/user/admin navigation and 375/768/1280 checks on key routes

- [ ] **Step 1: Install browsers once**

```bash
cd frontend
npx playwright install chromium
```

Expected: chromium installed.

- [ ] **Step 2: Create auth/navigation helpers**

```ts
// frontend/e2e/helpers.ts
import { expect, type Page } from '@playwright/test'

export async function loginAs(page: Page, username: string, password: string) {
  await page.goto('/login')
  await page.getByLabel('用户名').fill(username)
  await page.getByLabel('密码').fill(password)
  await page.getByRole('button', { name: '登录' }).click()
  await expect(page.getByRole('button', { name: '退出' })).toBeVisible()
}

export async function expectNoAdminLink(page: Page) {
  await expect(page.getByRole('link', { name: '管理' })).toHaveCount(0)
}

export const viewports = {
  mobile: { width: 375, height: 812 },
  tablet: { width: 768, height: 1024 },
  desktop: { width: 1280, height: 800 },
}
```

- [ ] **Step 3: Write public navigation + viewport tests**

```ts
// frontend/e2e/public-search.spec.ts
import { test, expect } from '@playwright/test'
import { viewports } from './helpers'

for (const [name, size] of Object.entries(viewports)) {
  test(`public home and search usable at ${name}`, async ({ page }) => {
    await page.setViewportSize(size)
    await page.goto('/')
    await expect(page.getByRole('textbox', { name: /核验账号|搜索关键词/i }).first()).toBeVisible()
    await expectNoHorizontalOverflow(page)
    await page.goto('/search')
    await expect(page.getByRole('heading', { name: '查询黑名单' })).toBeVisible()
    await expectNoHorizontalOverflow(page)
  })
}

async function expectNoHorizontalOverflow(page: import('@playwright/test').Page) {
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1)
  expect(overflow).toBeFalsy()
}
```

- [ ] **Step 4: Write role matrix tests**

```ts
// frontend/e2e/navigation.spec.ts
import { test, expect } from '@playwright/test'
import { expectNoAdminLink, loginAs } from './helpers'

test('anonymous visitor cannot see management navigation', async ({ page }) => {
  await page.goto('/')
  await expectNoAdminLink(page)
})

test('normal user does not see admin link after login', async ({ page }) => {
  await loginAs(page, process.env.E2E_USER || 'testuser', process.env.E2E_USER_PASSWORD || 'password123')
  await expectNoAdminLink(page)
  await expect(page.getByRole('link', { name: '我的处罚' })).toBeVisible()
})

test('admin sees management navigation', async ({ page }) => {
  await loginAs(page, process.env.E2E_ADMIN || 'admin', process.env.E2E_ADMIN_PASSWORD || 'admin123')
  await expect(page.getByRole('link', { name: '管理' })).toBeVisible()
  await page.goto('/admin')
  await expect(page.getByRole('link', { name: '站点与配置' })).toBeVisible()
})
```

Use development seed credentials already documented (`admin/admin123`) when `GO_ENV=development`.

- [ ] **Step 5: Run e2e against local stack**

Terminal A:

```bash
cd backend
GO_ENV=development go run ./cmd/server
```

Terminal B:

```bash
cd frontend
npm run dev
```

Terminal C:

```bash
cd frontend
PLAYWRIGHT_BASE_URL=http://127.0.0.1:3000 npm run test:e2e
```

Expected: all specs PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/e2e frontend/playwright.config.ts
git commit -m "test: add Playwright role and viewport acceptance matrix"
```

---

### Task 4: 生产 Compose/Nginx smoke 与文档收口（Package C）

**Covers:** remaining-deploy-smoke

**Files:**
- Modify: `docker-compose.prod.yml` (only if frontend needs relative `/api` rewrite base)
- Create or use: `.env.production` from root template
- Modify: `docs/implementation-gap-analysis.md`
- Modify: `docs/frontend-gap-analysis.md`
- Optional: `README.md` verification section only with verified commands

**Interfaces:**
- Consumes: production compose services postgres/minio/backend/frontend/nginx
- Produces: same-origin `/api` responses via Nginx; documented evidence

- [ ] **Step 1: Preflight Docker daemon**

```bash
docker info >/dev/null
```

If this fails: mark Package C **BLOCKED**, continue only docs for Packages A/B evidence, do not fake smoke.

- [ ] **Step 2: Prepare production env file**

```bash
cp /data/Projects/UniBlack/.env.production .env.production
# ensure non-default secrets and MinIO credentials are set
```

Required keys at minimum:

```env
POSTGRES_USER=uniblack
POSTGRES_PASSWORD=<non-default>
POSTGRES_DB=uniblack
JWT_SECRET=<non-default>
REFRESH_SECRET=<non-default>
MINIO_ROOT_USER=<non-default>
MINIO_ROOT_PASSWORD=<non-default>
MINIO_BUCKET=uniblack-evidence
API_BASE_URL=http://backend:8080
```

- [ ] **Step 3: Build and start production stack**

```bash
docker compose -f docker-compose.prod.yml --env-file .env.production up --build -d
docker compose -f docker-compose.prod.yml ps
```

Expected: postgres healthy, backend/frontend/nginx up.

- [ ] **Step 4: Smoke checks through Nginx same-origin**

```bash
curl -fsS http://127.0.0.1/ | head
curl -fsS http://127.0.0.1/api/settings/public
curl -fsS -X POST http://127.0.0.1/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"<prod-admin-password>"}'
curl -o /dev/null -w '%{http_code}\n' http://127.0.0.1/admin
```

Expected:
- `/` returns HTML
- `/api/settings/public` returns JSON
- login returns tokens or expected setup/init error if first-run not complete
- `/admin` returns frontend HTML (client guard), not raw 500

- [ ] **Step 5: Tear down cleanly**

```bash
docker compose -f docker-compose.prod.yml --env-file .env.production down
```

Expected: containers stop; named volumes retained unless intentionally removed.

- [ ] **Step 6: Update gap ledgers with evidence only**

Update `docs/implementation-gap-analysis.md` and `docs/frontend-gap-analysis.md`:
- mark shared API migration verified with commit hash + commands
- mark Playwright matrix verified with command output summary
- mark Docker smoke verified only if Step 4 passed
- leave unmet items explicit

- [ ] **Step 7: Final verification gate and commit**

```bash
cd backend && go test ./... && go build ./cmd/server
cd ../frontend && npm run test:run && npm run typecheck && npm run lint && npm run build
# e2e if servers available
git add docs frontend docker-compose.prod.yml .github
git commit -m "chore: verify production smoke and close remaining acceptance gaps"
```

---

## Definition of Done for “完成这些规划”

All must be true:

1. `rg` shows no page-level token/fetch/login hard redirects outside providers.
2. Frontend unit/typecheck/lint/build pass.
3. Playwright role + viewport specs pass against a running stack.
4. Backend `go test ./...` and `go build ./cmd/server` still pass.
5. Production smoke either:
   - fully passes via Nginx same-origin, or
   - is explicitly documented as **blocked by Docker daemon** with exact failing command.
6. Gap ledgers updated with evidence, not aspirational language.

## Out of Scope for this remaining plan

- OAuth provider implementation
- Full dark mode
- OpenAPI generator / API Key product decision
- Hard-deleting legacy Case tables before Sunset date
- HttpOnly cookie auth migration

## Recommended execution order and ETA

| Package | ETA | Dependency |
| --- | --- | --- |
| A 页面 API 清零 + 提交未提交迁移 | 0.5–1 day | none |
| B Playwright 矩阵 | 0.5–1 day | A preferred |
| C 生产 smoke + 文档 | 0.5 day | Docker daemon |

Total remaining: **1.5–2.5 working days**, excluding Docker-host recovery if daemon stays unavailable.

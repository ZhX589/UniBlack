# UniBlack Next Development Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use compose:subagent (recommended) or compose:execute to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不破坏已合入的 Subject-Account-Event 主路径的前提下，收束遗留 Case 双轨、完成生产存储与安全边界、补齐动态前端基础和验证矩阵，并以可发布的部署证据结束 Phase 12/13。

**Architecture:** 后端先稳定公开读模型、兼容弃用策略、对象存储和启动安全，再将前端从逐页直接请求迁移到统一 API、会话和导航边界。只有 API 契约和部署依赖稳定后，才迁移 Subject/Event 公开页与管理端；每一阶段独立验证并更新差距台账，旧 Case 读取在明确弃用窗口内保持兼容。

**Tech Stack:** Go、Echo、GORM、PostgreSQL 14、golang-migrate、MinIO/S3 SDK、Next.js 14 App Router、React 18、TypeScript strict、Tailwind CSS/PostCSS、Vitest/Testing Library、Playwright、Docker Compose、GitHub Actions。

## Global Constraints

- 项目、模块、镜像和环境变量统一使用 `UniBlack` / `uniblack`，不得引入 `CloudBan`。
- 后端保持 `handler -> service -> repository -> db/models` 单向依赖；迁移仅使用 `golang-migrate` SQL，禁止 GORM 建表。
- `Subject` 是核心治理对象；`Account` 与 `Event` 是新主模型。旧 `Identifier`、`Case`、`Submission` 只能作为有期限的兼容层，不得新增依赖它们的功能。
- 不直接删除 `cases`、`identifiers`、旧 API 或历史数据；先发布弃用响应头、迁移文档和替代端点，再在单独变更中决定删除日期。
- 证据二进制文件必须经 `Storage` 接口写入 LocalStorage 或 S3-compatible storage；数据库仅保存元数据和治理索引。
- 开发环境验证码固定为 `123456`；非开发环境缺少 SMTP 必须失败；运行时只使用 demo captcha，不接入第三方验证端点。
- 不迁移 localStorage JWT 到 HttpOnly Cookie，不引入全局状态库、CMS 或任意后端菜单。
- 前端站内跳转只使用 Next `Link` 或 router；页面不得直接读取 token、拼接 Bearer header 或自行处理 401。
- 所有新增或变更的数据页面必须有 loading、empty、error、unauthorized 和正常状态，并验证 375px、768px、1280px、键盘焦点和 reduced motion。
- 只有执行过对应自动化测试、构建或 smoke 测试后，才能在差距台账中将项目标记为已验证。

---

## Current Baseline

- `main` / 当前 HEAD 为 `085ae8c`；Phase 13 的 Subject-Account-Event、默认发布、多文件/文本证据、归档、处罚申诉和 demo captcha 主路径已合入。
- 当前分支 `feature/case-event-deprecation-and-hardening` 有未提交的 Case API 弃用和 S3 adapter 改动；本计划将其作为 Task 1 的在途实现，未经测试和审查前不计为完成。
- 后端的主要剩余缺口是旧 Case/Submission 双轨、Account 优先公开查询/统计、MinIO/S3 运行时接入、生产首启与安全控制面、链接证据和 Event-first 申诉。
- 前端的主要剩余缺口是统一 API client、typed DTO、navigation registry、动态品牌 token、最小 UI 状态组件、Event 浏览/治理页面、响应式/无障碍、Vitest/Playwright 与 Docker smoke。
- `docs/database-design.md` 仍描述 Identifier/Case 旧模型；它只能作为历史基线，实施时以 Phase 13 规格、迁移和差距台账为准，并在兼容窗口确定后修订文档。

## File Structure

### Existing boundaries to retain

- `backend/internal/storage/storage.go`: `Storage` 接口、LocalStorage 与稳定证据 key 规则。
- `backend/internal/service/event.go`: Subject/Account/Event 的新发布事务。
- `backend/internal/repository/subject.go`: 公开查询与 Account/Identifier 兼容读取的集中位置。
- `backend/internal/handler/public.go`: `/api/v1` Event-first 公共 DTO、统计和 legacy alias。
- `backend/cmd/server/main.go`: 依赖组装、路由、启动与健康检查门禁。
- `frontend/lib/`: API、认证、设置、导航与 DTO 的唯一共享边界。
- `frontend/components/shell/`: 公共、认证和管理壳层。
- `frontend/components/ui/`: 最小可访问基础组件和页面状态组件。
- `frontend/app/`: 只承担路由组合和页面特有交互，不直接管理认证协议。

### New/changed files by work package

- Task 1: `backend/internal/storage/s3.go`、`backend/internal/middleware/deprecation.go`、`backend/cmd/server/main.go`、`backend/internal/handler/public.go`、相应 Go 测试和 API 迁移文档。
- Task 2: `backend/internal/config/config.go`、启动/设置/访问名单中间件、setup handler/service、迁移测试与 CI migrate job。
- Task 3: Event/appeal/evidence/archive service、repository、handler、迁移及 HTTP/数据库集成测试。
- Task 4: `frontend/package.json`、Vitest/Playwright 配置、`frontend/lib/api.ts`、`lib/types.ts`、`lib/navigation.ts`、providers 与对应测试。
- Task 5: `frontend/app/globals.css`、`tailwind.config.js`、`components/ui/*`、`components/shell/*`、根布局和管理布局。
- Task 6: 首页、搜索、Subject/Event 详情、提交、认证和所有 `app/admin/*` 页面，以及它们的单元/E2E 测试。
- Task 7: Dockerfiles、Compose、Nginx 配置、CI、README、configuration、gap-analysis 与发布验证脚本。

---

### Task 1: 收束 Event API 与对象存储

**Covers:** S1, S2

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/handler/public.go`
- Create or complete: `backend/internal/middleware/deprecation.go`
- Create or complete: `backend/internal/storage/s3.go`
- Modify: `backend/internal/storage/storage_test.go`
- Create: `backend/internal/handler/public_test.go`
- Create: `docs/api/case-event-migration.md`

**Interfaces:**
- Consumes: `storage.Storage`, existing `EventService`, existing public `/api/v1` routes.
- Produces: Event-first public read endpoints, legacy Case response aliases with deprecation headers, and a selectable verified `Storage` implementation.

- [ ] **Step 1: Freeze and inspect the current in-flight diff**

Run: `git diff -- backend/cmd/server/main.go backend/go.mod backend/go.sum backend/internal/handler/public.go backend/internal/middleware/deprecation.go backend/internal/storage/s3.go`

Expected: every changed route, header, Go-version change and S3 behavior is understood before modifying it; unrelated worktree changes remain untouched.

- [ ] **Step 2: Write failing storage contract tests**

```go
func TestS3StorageRoundTrip(t *testing.T) {
	store := newMinIOTestStore(t)
	key := "subjects/UBS_01H00000000000000000000000/evidence/UBS_01H00000000000000000000000_E001_F001.txt"
	if _, err := store.Upload(t.Context(), key, strings.NewReader("evidence"), "text/plain"); err != nil {
		t.Fatal(err)
	}
	reader, err := store.Open(t.Context(), key)
	if err != nil { t.Fatal(err) }
	defer reader.Close()
	got, err := io.ReadAll(reader)
	if err != nil || string(got) != "evidence" { t.Fatalf("got %q, err %v", got, err) }
	if err := store.Delete(t.Context(), key); err != nil { t.Fatal(err) }
}
```

- [ ] **Step 3: Run the storage test against MinIO and confirm failure before implementation**

Run: `cd backend && go test ./internal/storage -run TestS3StorageRoundTrip -v`

Expected: FAIL because no completed S3 test adapter/harness exists yet.

- [ ] **Step 4: Implement the smallest S3 adapter and startup selection**

```go
func NewS3Storage(endpoint, accessKey, secretKey, bucket string, useSSL bool, publicBase string) (*S3Storage, error)

func (s *S3Storage) Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error)
func (s *S3Storage) Open(ctx context.Context, key string) (io.ReadCloser, error)
func (s *S3Storage) Delete(ctx context.Context, key string) error
func (s *S3Storage) GetURL(key string) string
func (s *S3Storage) Path(key string) (string, error)
```

Use MinIO's bucket-exists/create APIs in the constructor. Select S3 when `MINIO_ENDPOINT` is configured; retain LocalStorage only as an explicit development fallback. In production, storage initialization failure must stop startup rather than write to ephemeral container disk.

- [ ] **Step 5: Write failing public Event and legacy deprecation tests**

```go
func TestLegacyCaseRouteReturnsSunsetHeaders(t *testing.T) {
	e := newPublicTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/legacy-id", nil)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Header().Get("Deprecation") != "true" { t.Fatal("missing deprecation header") }
	if res.Header().Get("Sunset") == "" { t.Fatal("missing sunset header") }
}
```

- [ ] **Step 6: Implement Event-first reads and document the compatibility contract**

Expose canonical Event payload fields and retain only documented `case_count`, `cases`, `description` aliases on legacy Case routes. Apply RFC-compatible `Deprecation`, `Sunset`, `Link`, and `Warning` headers only to Case endpoints. Document replacement endpoints, response field mapping, Sunset date, and client migration examples in `docs/api/case-event-migration.md`.

- [ ] **Step 7: Run focused verification**

Run: `cd backend && go test ./internal/storage ./internal/handler -run 'Test(S3StorageRoundTrip|LegacyCaseRouteReturnsSunsetHeaders)' -v`

Expected: PASS with MinIO test service available; legacy route has all documented headers and Event route returns no legacy-only warning.

- [ ] **Step 8: Run full backend verification and commit**

Run: `cd backend && go test ./... && go build ./cmd/server`

Expected: PASS.

Commit:

```bash
git add backend docs/api/case-event-migration.md
```

### Task 2: 加固生产启动、初始化和访问控制

**Covers:** S3

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/handler/setup.go`
- Modify: `backend/internal/service/auth.go`
- Modify: `backend/internal/service/setting.go`
- Modify: `backend/internal/middleware/auth.go`
- Create: `backend/internal/middleware/access_list.go`
- Create: `backend/internal/handler/setup_test.go`
- Create: `backend/internal/db/migrate_test.go`
- Modify: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: settings OptionMap, access-list repository, JWT configuration and database migrations.
- Produces: fail-fast startup configuration, single-use production setup, configured rate limits and request-level access-list enforcement.

- [ ] **Step 1: Write failing startup and setup concurrency tests**

```go
func TestProductionRejectsDefaultJWTSecrets(t *testing.T) {
	cfg := config.Config{Environment: "production", JWTSecret: "change-me-access-secret", RefreshSecret: "change-me-refresh-secret"}
	if err := cfg.Validate(); err == nil { t.Fatal("expected production secret validation failure") }
}

func TestInitializeOnlySucceedsOnce(t *testing.T) {
	results := runConcurrentSetupRequests(t, 2)
	if results.successes != 1 { t.Fatalf("successes = %d, want 1", results.successes) }
}
```

- [ ] **Step 2: Run focused tests and confirm they fail**

Run: `cd backend && go test ./internal/config ./internal/handler -run 'Test(ProductionRejectsDefaultJWTSecrets|InitializeOnlySucceedsOnce)' -v`

Expected: FAIL because config validation and atomic initialization do not yet exist.

- [ ] **Step 3: Implement a single environment and fail-fast boundary**

Add one `Environment` field sourced from `APP_ENV` with `development` default. Validate required DB, JWT, refresh-token and production storage configuration before starting Echo. If DB connection, migration or required storage health check fails, exit non-zero. Development-only admin seeding and fixed verification behavior must use the same environment value.

- [ ] **Step 4: Make production setup atomic and scoped**

Implement setup as one database transaction with a conditional initialization state transition. Reject initialize requests after completion. Do not start a partially wired server when storage/database setup fails. Keep the setup page/API contract explicit rather than silently creating production credentials.

- [ ] **Step 5: Write and implement access-control tests**

```go
func TestWhitelistSkipsConfiguredRateLimit(t *testing.T) { /* request from listed IP reaches handler */ }
func TestBlacklistRejectsRequestBeforeHandler(t *testing.T) { /* request from listed IP receives 403 */ }
func TestPublicAndAuthLimitsUseSettings(t *testing.T) { /* distinct configured limits are observed */ }
```

Read `security.rate_limit_public` and `security.rate_limit_auth` through the OptionMap for each request or safe reload boundary. Apply blacklist enforcement before handlers and whitelist rate-limit bypass without using menu-level logic as security.

- [ ] **Step 6: Add migration up/down CI verification**

Add a CI job that starts an empty PostgreSQL database, performs all migration `up`, repeats `up`, performs a controlled `down`/`up` cycle, and fails on schema errors. Keep the Go version in `go.mod`, CI and backend Dockerfile identical before merging.

- [ ] **Step 7: Run verification and commit**

Run: `cd backend && go test ./... && go build ./cmd/server`

Expected: PASS; configuration failures are covered without starting a listener.

Commit:

```bash
git add backend .github/workflows/ci.yml
```

### Task 3: 完成 Event 治理数据闭环

**Covers:** S4

**Files:**
- Modify: `backend/internal/service/event.go`
- Modify: `backend/internal/service/evidence.go`
- Modify: `backend/internal/service/appeal.go`
- Modify: `backend/internal/service/submission.go`
- Modify: `backend/internal/repository/event.go`
- Modify: `backend/internal/repository/subject.go`
- Modify: `backend/internal/export/archive.go`
- Modify: `backend/internal/handler/event.go`
- Modify: `backend/internal/handler/appeal.go`
- Create: `backend/internal/migrations/000011_event_governance.up.sql`
- Create: `backend/internal/migrations/000011_event_governance.down.sql`
- Create: `backend/internal/service/event_governance_test.go`

**Interfaces:**
- Consumes: Event publish transaction, Storage, sanction service, archive manifest schema.
- Produces: link evidence, Event-first appeals, append-only governance history, Account-first public search and accurate statistics.

- [ ] **Step 1: Write failing link-evidence round-trip test**

```go
func TestPublishExportImportPreservesLinkEvidence(t *testing.T) {
	published := publishEvent(t, PublishRequest{Evidence: []EvidenceInput{{Type: "link", URL: "https://example.test/proof", Title: "chat record"}}})
	archive := exportSubject(t, published.SubjectPublicID)
	imported := importArchive(t, archive)
	if imported.Events[0].Evidence[0].URL != "https://example.test/proof" { t.Fatal("link URL was lost") }
}
```

- [ ] **Step 2: Run it to verify the current omission**

Run: `cd backend && go test ./internal/service ./internal/export -run TestPublishExportImportPreservesLinkEvidence -v`

Expected: FAIL because publish and import do not preserve link metadata.

- [ ] **Step 3: Implement link evidence without storing remote content**

Extend the publish request, evidence metadata and manifest v1-compatible fields with `type`, `url`, `title`, and `description`. Validate absolute `http`/`https` URLs. Store no remote binary and do not claim an unverifiable SHA-256 for a link; export/import metadata losslessly.

- [ ] **Step 4: Write Event appeal and malicious-submission tests**

```go
func TestMaliciousSubmissionOutcomeCreatesAuditedSanction(t *testing.T) {
	appeal := createEventAppeal(t)
	reviewAppeal(t, appeal.ID, "malicious_submission")
	if !hasActiveSanction(t, appeal.SubmittedBy) { t.Fatal("expected sanction") }
	if !hasAuditAction(t, "sanction.create") { t.Fatal("expected audit record") }
}
```

- [ ] **Step 5: Implement Event-first appeal and append-only behavior**

Add Event IDs to appeal creation/read APIs and make Case IDs a compatibility adapter only. Define the product-approved automatic sanction type/duration for `malicious_submission` before coding it; create the sanction and audit record in the same transaction as the adjudication. Replace physical deletion of Submission/Appeal records with status transitions or append-only history.

- [ ] **Step 6: Make public reads Account-first and statistics truthful**

Modify Subject lookup/search to query normalized Accounts first, then Identifier compatibility rows. Derive public counts from Event state rather than stale `case_count`; implement `/api/v1/statistics` with real published-subject/event values and defined zero/error semantics.

- [ ] **Step 7: Run transactional and archive verification**

Run: `cd backend && go test ./internal/service ./internal/export ./internal/repository -v && go test ./...`

Expected: PASS for duplicate Account rollback, storage compensation, link evidence round trip, sanctions, audit history and public statistics.

- [ ] **Step 8: Commit**

```bash
git add backend
```

### Task 4: 建立前端验证、API 和会话边界

**Covers:** S5, S6

**Files:**
- Modify: `frontend/package.json`
- Create: `frontend/vitest.config.ts`
- Create: `frontend/playwright.config.ts`
- Create: `frontend/test/setup.ts`
- Create: `frontend/lib/api.ts`
- Create: `frontend/lib/api-error.ts`
- Create: `frontend/lib/types.ts`
- Create: `frontend/lib/navigation.ts`
- Modify: `frontend/app/providers.tsx`
- Modify: `frontend/lib/settings.ts`
- Create: `frontend/lib/api.test.ts`
- Create: `frontend/lib/navigation.test.ts`
- Create: `frontend/app/providers.test.tsx`

**Interfaces:**
- Consumes: existing relative `/api` rewrite and localStorage JWT contract.
- Produces: `apiRequest<T>()`, `ApiError`, typed DTOs, one 401 logout boundary, `visibleNavigation()` and explicit provider loading/fallback state.

- [ ] **Step 1: Add reproducible frontend test commands**

Add scripts with these exact responsibilities:

```json
{
  "typecheck": "tsc --noEmit --incremental false",
  "test": "vitest",
  "test:run": "vitest run",
  "test:e2e": "playwright test"
}
```

Install only Vitest, jsdom, Testing Library and Playwright dependencies required by these scripts.

- [ ] **Step 2: Write failing API client tests**

```ts
it("adds session authorization and logs out once on 401", async () => {
  const logout = vi.fn();
  server.use(http.get("/api/private", () => new HttpResponse(null, { status: 401 })));
  await expect(apiRequest("/private", { auth: true })).rejects.toMatchObject({ status: 401 });
  expect(logout).toHaveBeenCalledTimes(1);
});
```

- [ ] **Step 3: Implement the smallest typed API boundary**

```ts
export class ApiError extends Error {
  constructor(readonly status: number, message: string, readonly body?: unknown) { super(message); }
}

export async function apiRequest<T>(path: string, options?: ApiRequestOptions): Promise<T>
```

Use only relative `/api` paths. Support JSON and `FormData`, caller-provided `AbortSignal`, structured non-JSON errors and one injected 401 callback. Token lookup remains inside the session/provider boundary, not in pages.

- [ ] **Step 4: Define DTOs and a single navigation registry**

Define `PublicSettings`, `AuthUser`, `Subject`, `Account`, `Event`, `Evidence`, `Statistics`, `Sanction`, `AdminUser`, `AccessList` and pagination DTOs based on verified backend responses. Define typed route items with `href`, `label`, `area`, `roles`, `requiresAuth`, `feature` and active-match behavior. Test anonymous, user, moderator and admin visibility plus registration feature gating.

- [ ] **Step 5: Refactor providers before pages**

Move settings parsing and token lifecycle behind providers. Preserve localStorage protocol but expose only `loading`, `anonymous`, `authenticated`, `login`, `logout`, `hasRole` and typed site settings. Inject validated brand theme variables in one location; retain safe fallback settings if the public settings request fails.

- [ ] **Step 6: Run verification and commit**

Run: `cd frontend && npm run test:run && npm run typecheck && npm run lint && npm run build`

Expected: PASS.

Commit:

```bash
git add frontend
```

### Task 5: 落地动态 Shell、token 与基础组件

**Covers:** S6, S7

**Files:**
- Modify: `frontend/app/globals.css`
- Delete: `frontend/styles/globals.css`
- Modify: `frontend/tailwind.config.js`
- Modify: `frontend/app/layout.tsx`
- Modify: `frontend/app/admin/layout.tsx`
- Modify: `frontend/components/shell/site-header.tsx`
- Create: `frontend/components/shell/admin-sidebar.tsx`
- Create: `frontend/components/shell/mobile-nav.tsx`
- Create: `frontend/components/shell/site-footer.tsx`
- Create: `frontend/components/ui/{button,field,panel,badge,alert,loading-state,empty-state,error-state,table}.tsx`
- Create: component tests under `frontend/components/ui/`

**Interfaces:**
- Consumes: typed providers, `visibleNavigation()`, CSS brand tokens.
- Produces: accessible Shells and a minimal reusable UI/state primitive set.

- [ ] **Step 1: Write failing button and state-component accessibility tests**

```tsx
it("keeps the primary button keyboard-focusable and at least 44px tall", () => {
  render(<Button>Save changes</Button>);
  expect(screen.getByRole("button", { name: "Save changes" })).toHaveClass("min-h-11");
});

it("announces an error state", () => {
  render(<ErrorState message="Unable to load subjects" />);
  expect(screen.getByRole("alert")).toHaveTextContent("Unable to load subjects");
});
```

- [ ] **Step 2: Add CSS runtime tokens and semantic Tailwind mappings**

Define `--background`, `--surface`, `--foreground`, `--muted`, `--border`, `--primary`, `--primary-foreground`, `--danger`, `--warning`, `--success`, `--focus` and radius tokens. Remove the incomplete dark-mode media override and page gradient. Add `prefers-reduced-motion` transition suppression. Map Tailwind semantic colors to those variables without a CDN.

- [ ] **Step 3: Implement the minimal UI primitives**

Each primitive must have one responsibility. `Button` supports `primary`, `secondary`, `ghost`, and `danger`; fields associate labels to controls; status components expose semantic roles; Table supplies only layout/overflow semantics, not business behavior.

- [ ] **Step 4: Replace hard-coded shell navigation**

Render public, account and admin navigation exclusively through `visibleNavigation()`. Make all internal routes use `Link`. Build a keyboard-accessible mobile navigation control with Escape handling and `aria-expanded`; retain server-side RBAC as the authority.

- [ ] **Step 5: Apply dynamic brand values across the root shell**

Use site name, description, logo (when configured) and validated theme color in header, footer and page metadata fallback behavior. `UniBlack` appears only when settings are unavailable. A configured primary color that fails contrast must retain a safe action foreground/focus combination.

- [ ] **Step 6: Run tests and commit**

Run: `cd frontend && npm run test:run && npm run typecheck && npm run lint && npm run build`

Expected: PASS.

Commit:

```bash
git add frontend
```

### Task 6: 逐页迁移到 Subject/Event 用户体验

**Covers:** S7, S8

**Files:**
- Modify: `frontend/app/page.tsx`
- Modify: `frontend/app/search/page.tsx`
- Modify: `frontend/app/subjects/page.tsx`
- Modify: `frontend/app/subjects/[id]/page.tsx`
- Create: `frontend/app/events/[id]/page.tsx`
- Modify: `frontend/app/cases/[id]/page.tsx`
- Modify: `frontend/app/submit/page.tsx`
- Modify: `frontend/app/login/page.tsx`
- Modify: `frontend/app/register/page.tsx`
- Modify: `frontend/app/setup/page.tsx`
- Modify: `frontend/app/sanctions/page.tsx`
- Modify: `frontend/app/admin/*.tsx`
- Create: page/component unit tests and `frontend/e2e/*.spec.ts`

**Interfaces:**
- Consumes: typed API/session/navigation/UI boundaries and Event-first backend reads.
- Produces: Event-first public browsing and governance paths with uniform data states and responsive behavior.

- [ ] **Step 1: Write public-page behavior tests before migration**

```tsx
it("keeps search actionable when statistics are unavailable", async () => {
  mockStatisticsFailure();
  render(<HomePage />);
  expect(await screen.findByRole("textbox", { name: /核验账号/i })).toBeEnabled();
  expect(screen.getByText(/统计暂不可用/i)).toBeVisible();
});

it("does not show an empty-result message before a search", () => {
  render(<SearchPage />);
  expect(screen.queryByText(/未找到相关结果/i)).not.toBeInTheDocument();
});
```

- [ ] **Step 2: Make public pages Event-first**

Use the real statistics endpoint on the home page, but never block the primary account-verification form on failure. Migrate Subject details from `CaseItem` to typed Event timeline entries and add `/events/[id]`. Make `/cases/[id]` a documented compatibility redirect or notice consistent with Task 1's Sunset policy, rather than a new feature target.

- [ ] **Step 3: Migrate authenticated pages to providers and API client**

Remove every page-level `localStorage.getItem`, manual `Authorization`, raw internal anchor and `window.location.href`. Submit multipart evidence with the shared client. Give login/register/setup consistent field labels, return navigation, errors and success feedback. Keep setup as a deliberately isolated first-run shell.

- [ ] **Step 4: Rebuild admin around Event governance, not legacy Case review**

During the compatibility window, label old Submission/Case screens as legacy and link to Event-first governance. Add Event appeal queue/decision entry points only after Task 3's API exists. Standardize admin data tables with error/empty/loading states, compact desktop density, mobile overflow or summary behavior, and accessible confirmation flows instead of `prompt()`/`confirm()`.

- [ ] **Step 5: Add E2E role and viewport coverage**

```ts
test("admin sees full management navigation", async ({ page }) => {
  await loginAs(page, "admin");
  await expect(page.getByRole("link", { name: "站点与配置" })).toBeVisible();
});

test("anonymous visitor cannot see management navigation", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("link", { name: /管理/i })).toHaveCount(0);
});
```

Run the key public search, login/logout, registration setting, submit, Subject/Event detail and admin-navigation scenarios at 375px, 768px and 1280px.

- [ ] **Step 6: Run full frontend verification and commit**

Run: `cd frontend && npm run test:run && npm run typecheck && npm run lint && npm run build && npm run test:e2e`

Expected: PASS.

Commit:

```bash
git add frontend
```

### Task 7: 完成可重复部署、CI 与文档发布门禁

**Covers:** S9

**Files:**
- Modify: `backend/Dockerfile`
- Modify: `frontend/Dockerfile`
- Create: `frontend/.dockerignore`
- Modify: `docker-compose.yml`
- Modify: `docker-compose.prod.yml`
- Modify or create: `nginx/nginx.conf`
- Modify: `.github/workflows/ci.yml`
- Modify: `README.md`
- Modify: `docs/configuration.md`
- Modify: `docs/implementation-gap-analysis.md`
- Modify: `docs/frontend-gap-analysis.md`

**Interfaces:**
- Consumes: verified backend/frontend commands, S3 selection, API rewrite and Nginx same-origin policy.
- Produces: repeatable CI/containers and evidence-backed project documentation.

- [ ] **Step 1: Write Docker smoke scenarios**

```text
docker compose up --build -d
GET / returns HTML and compiled CSS
GET /api/settings/public returns a public settings payload through the same origin proxy
POST /api/auth/login returns the expected authenticated response
GET /admin as anonymous returns the frontend guard/login flow
```

- [ ] **Step 2: Make images deterministic**

Use a Go builder version matching `go.mod`. Use `npm ci` in the frontend build stage and do not copy host `node_modules` into images. Ensure production browser requests use relative `/api` and Nginx proxies to backend; avoid build-time `localhost` fallback in production.

- [ ] **Step 3: Expand CI to mirror local gates**

Run Go formatting/lint/build/test/migrations; run frontend install with lockfile, lint, typecheck, unit and Playwright tests. Start required PostgreSQL/MinIO services for integration jobs. Fail CI if Docker build/smoke fails.

- [ ] **Step 4: Run local deployment verification**

Run: `docker compose up --build -d`

Expected: postgres and minio become healthy; backend performs migrations and initializes selected storage; frontend serves assets; same-origin API calls succeed.

Run: `docker compose down`

Expected: containers stop cleanly without deleting named data volumes.

- [ ] **Step 5: Reconcile documentation only with verified evidence**

Update README endpoint tables and local/production setup instructions, configuration rules, Case-to-Event migration guidance, and both gap ledgers. Preserve unmet items as partial/unimplemented, include commands and dates for every claim of verified completion, and correct `database-design.md` terminology only after the compatibility window is formally decided.

- [ ] **Step 6: Final release gate and commit**

Run: `cd backend && go test ./... && go build ./cmd/server`

Expected: PASS.

Run: `cd frontend && npm run lint && npm run typecheck && npm run test:run && npm run build && npm run test:e2e`

Expected: PASS.

Commit:

```bash
git add .github backend frontend docker-compose.yml docker-compose.prod.yml nginx README.md docs
```

## Dependency Order

```text
Task 1: Event API deprecation + S3 storage
  -> Task 2: production startup and access controls
  -> Task 3: Event governance data closure
  -> Task 4: frontend verification/API/session boundary
  -> Task 5: dynamic Shell/tokens/UI
  -> Task 6: Event-first page migration
  -> Task 7: Docker, CI, documentation and release gate
```

Task 4 can begin after Task 1's public Event DTO and deprecation contract are frozen, but Task 6 must wait until Task 3 exposes Event-first public reads and appeals. Do not declare Phase 12 or Phase 13 complete until Task 7 verification has passed.

## Deferred Decisions

- API Key is required by historical Phase 8 text but marked as an architecture decision conflict in `docs/frontend-gap-analysis.md`; decide its product/security model in a dedicated specification before implementation.
- OAuth, complete dark mode, multi-tenant organizations, plugin systems, bots, federation, GraphQL, full-text search and i18n remain outside this plan.
- Bulk import/export of access lists, password reset UX and expanded user role-management UI should be planned only after Task 2 has stabilized authorization and access-list middleware.

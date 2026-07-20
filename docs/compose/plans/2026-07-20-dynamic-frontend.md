# Dynamic Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use compose:subagent (recommended) or compose:execute to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 UniBlack 前端升级为配置驱动、身份感知、角色过滤、响应式且可复用的动态 Next.js 应用。

**Architecture:** 集中导航注册表、Auth/Settings Providers 和统一 API client 构成应用 Shell；公开、档案和后台布局共享 Tailwind token，但使用不同信息密度。保持现有后端 JWT/API 协议，不引入 CMS、全局状态库或请求库。

**Tech Stack:** Next.js 14 App Router、React 18、TypeScript strict、Tailwind CSS 3/PostCSS、Vitest/Testing Library、Playwright、Docker Compose。

## Global Constraints

- 执行前创建 `feature/dynamic-frontend`，不在 `main` 开发。
- Tailwind 走 npm/PostCSS，不使用 CDN。
- 代码仍命名 UniBlack；`site.name` 只控制实例显示名。
- 菜单过滤不替代后端 RBAC。
- 不迁移 localStorage JWT，不新增后端业务，不实现 CMS。
- 页面覆盖五类数据状态；375/768/1280 验收；支持 reduced motion。

## File Map

- `frontend/app/layout.tsx`、`app/providers.tsx`：Root Shell 与 Providers。
- `frontend/app/admin/layout.tsx`：管理守卫和 Shell。
- `frontend/components/shell/*`：顶栏、移动菜单、侧栏和认证布局。
- `frontend/components/ui/*`：最小基础组件和状态组件。
- `frontend/lib/{api,auth-context,settings-context,navigation,types}.ts(x)`：共享边界。
- `frontend/app/**/page.tsx`：逐类迁移页面。
- `frontend/app/globals.css`、`tailwind.config.js`：token。
- `frontend/Dockerfile`、`.dockerignore`、Compose：可复现构建。

---

### Task 1: 测试与设计 token 基线

**Covers:** S6, S8, S10

**Files:** Modify `frontend/package.json`, `tailwind.config.js`, `app/globals.css`; Delete `styles/globals.css`; Create `vitest.config.ts`, `test/setup.ts`, `components/ui/button.tsx`, `button.test.tsx`.

**Interfaces:** Produces semantic CSS-variable colors and native-compatible `ButtonProps`.

- [ ] 添加 Vitest、jsdom、Testing Library dev dependencies，以及 `test`、`test:run`、`typecheck` scripts。
- [ ] 先写 Button disabled、variant、focus-visible 失败测试并运行确认失败。
- [ ] 实现 `primary|secondary|ghost|danger` Button 和最小 44px 触控目标。
- [ ] 合并全局 CSS，移除不完整 dark media，加入 reduced-motion fallback。
- [ ] 运行 `npm run test:run && npm run typecheck`，预期通过。
- [ ] 提交 `test(frontend): establish UI token baseline`。

### Task 2: 统一 API 与领域类型

**Covers:** S4, S8

**Files:** Create `frontend/lib/types.ts`, `lib/api.ts`, `lib/api.test.ts`.

**Interfaces:** Produces `ApiError`, `apiRequest<T>()`, token getter and typed domain responses.

- [ ] 写 JSON success、backend error、非 JSON 500、Bearer、401 callback 失败测试。
- [ ] 实现 same-origin `/api` client、内容解析、ApiError 和一次性 401 边界。
- [ ] 定义 PublicSettings、AuthUser、Subject、Case、Statistics、Submission、AdminUser、AccessList、Pagination。
- [ ] 运行测试与 typecheck，预期通过。
- [ ] 提交 `refactor(frontend): centralize typed API access`。

### Task 3: Settings 与 Auth Providers

**Covers:** S2, S3, S4

**Files:** Create `lib/settings-context.tsx`, `settings-context.test.tsx`, `auth-context.tsx`, `auth-context.test.tsx`, `app/providers.tsx`; Modify `app/layout.tsx`.

**Interfaces:** Produces `useSiteSettings()` and `useAuth()`.

- [ ] 测试设置 loading/success/fallback，以及 JWT valid/expired/malformed。
- [ ] 测试 login 更新 context、logout 清 token、401 转 anonymous。
- [ ] 使用现有 localStorage keys 实现 Providers，不增加 refresh 复杂度。
- [ ] 校验主题色并注入 `--primary`；不安全色使用操作回退色。
- [ ] 运行 provider tests、typecheck、build。
- [ ] 提交 `feat(frontend): add settings and auth providers`。

### Task 4: 动态导航与三类 Shell

**Covers:** S2, S3, S5, S6, S8

**Files:** Create `lib/navigation.ts`, test, `components/shell/{site-header,mobile-nav,site-footer,admin-sidebar,auth-shell}.tsx`, `app/admin/layout.tsx`; Modify root layout.

**Interfaces:** Produces `NavItem`, `visibleNavigation()` and reusable Shells.

- [ ] 测试 anonymous/authenticated/moderator/admin 导航矩阵和注册开关。
- [ ] 实现单一类型化路由注册表，不接受后端任意 URL。
- [ ] 用 SiteHeader + `Link` + `aria-current` 替换硬编码 anchors。
- [ ] 实现登录/注册、用户/退出和按角色管理入口。
- [ ] 实现 Admin 守卫、角色侧栏和同源移动菜单。
- [ ] 验证键盘、Escape、375px 和角色测试。
- [ ] 提交 `feat(frontend): add role-aware application shells`。

### Task 5: 公开基础页与可信档案

**Covers:** S6, S7, S8

**Files:** Modify home, search, subjects, subject detail, case detail; Create minimal panel/badge/alert/skeleton/empty/page-header and subject summary/table components; Test search page.

**Interfaces:** Consumes typed api/UI; produces shared subject summary/table.

- [ ] 测试 search idle 不显示“未找到”、成功可进详情、错误可恢复。
- [ ] 首页接 statistics；失败时保留核验能力并标明统计不可用。
- [ ] 以账号核验输入为首页主要行动，不使用渐变 Hero/功能卡网格。
- [ ] 名单加入分页筛选，桌面表格与移动档案摘要共享数据。
- [ ] Subject/Case 按档案头、标识符、状态、时间线重组。
- [ ] 替换这些页面所有内部原生链接。
- [ ] 测试、typecheck、build、三视口验证后提交。

### Task 6: 认证、举报和初始化页

**Covers:** S2, S4, S6, S7, S8

**Files:** Modify login/register/submit/setup; Create `components/auth/captcha-widget.tsx`, UI input/select/form-field; Test login/register.

**Interfaces:** Produces isolated CaptchaWidget.

- [ ] 测试登录无刷新更新 Shell 和 return URL。
- [ ] 测试注册 loading、closed、email verification、missing captcha key。
- [ ] 抽离 captcha 脚本生命周期，保留现有 provider 行为。
- [ ] 替换直接 fetch/localStorage/window.location/内部 anchors。
- [ ] 应用 Auth Shell 和一致表单状态；setup 保持独立 Shell。
- [ ] 验证 captcha 关闭不加载第三方脚本。
- [ ] 测试、typecheck、build 后提交。

### Task 7: 管理控制台

**Covers:** S3, S5, S6, S7, S8

**Files:** Modify all `app/admin` pages; Create data-table/status-badge/confirm-dialog/toast/admin-page-header; Test admin layout/settings.

**Interfaces:** Consumes Admin Shell/API/schema; produces shared DataTable container.

- [ ] 测试 anonymous redirect、moderator 限制、admin 全菜单和 403。
- [ ] 将页面迁入统一 Admin Shell 和 header/toolbars。
- [ ] 由 settings schema 渲染已知控件，保留分组与 secret 语义。
- [ ] 将模糊 onBlur 保存改为 dirty/saving/saved/error 状态。
- [ ] 危险用户/名单操作增加确认，后端权限仍为真源。
- [ ] 验证桌面密度和移动列表/滚动行为。
- [ ] 测试、typecheck、build 后提交。

### Task 8: 生产构建、E2E 与文档收口

**Covers:** S8, S9, S10, S11

**Files:** Modify Next config, Dockerfile, Compose, README, roadmap, gap analysis; Create `.dockerignore`, Playwright config and E2E specs.

**Interfaces:** Produces reproducible dev/prod builds and release evidence.

- [ ] 添加匿名导航、登录退出、角色菜单、移动菜单、首页查询、管理导航 E2E。
- [ ] Docker 改 `npm ci`，增加 `.dockerignore`，选择不依赖预存 `public/` 的输出策略。
- [ ] 明确同源 `/api` 和 build-time rewrite 行为，生产不回退 localhost。
- [ ] 运行 lint、typecheck、unit、build、Playwright，全部通过。
- [ ] 构建生产镜像并 smoke test 首页、CSS、公开设置、登录、管理路由。
- [ ] 将实际证据写入差距文档，之后才标 Phase 12 完成。
- [ ] 提交 `docs: record dynamic frontend verification`。

## Final Review Gate

- [ ] `rg '<a\s+href="/' frontend/app frontend/components` 无内部 raw anchor。
- [ ] `rg 'localStorage|Authorization:.*Bearer' frontend/app` 无页面级认证访问。
- [ ] `npm run lint && npm run typecheck && npm run test:run && npm run build` exit 0。
- [ ] Playwright desktop/mobile 全通过。
- [ ] Production Compose frontend 通过 CSS/API proxy smoke test。
- [ ] S1–S11 全覆盖，差距台账附验证证据。

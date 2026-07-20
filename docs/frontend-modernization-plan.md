# Frontend Dynamic UI and Design System Implementation Plan

> **For agentic workers:** 实施前必须获得用户批准。本计划只定义后续工作，不代表已实现。

**Goal:** 把现有静态 MVP 前端升级为配置驱动、认证态驱动、RBAC 可见、响应式且具有统一设计语言的产品界面。

**Architecture:** 建立一层动态 App Shell，统一组合 public settings、session、导航 registry 与响应式布局。页面通过统一 API client、领域类型和 UI primitives 获取能力，不再直接依赖散落的 `localStorage`、硬编码品牌或重复 Tailwind 字符串。

**Tech Stack:** Next.js 14 App Router、React 18、TypeScript、Tailwind CSS 3（PostCSS 构建）、shadcn/ui/Radix、Vitest + Testing Library、Playwright（拟引入，版本以实施时官方兼容矩阵锁定）。

## Global Constraints

- 当前文档阶段不修改任何前端代码或依赖。
- 项目代码名保持 UniBlack；`site.name` 仅改变运行时显示品牌，不改变包名、镜像名和环境变量。
- Tailwind 必须通过构建流程，禁止 CDN。
- 后端 RBAC 是授权真源；前端角色过滤只改善可发现性，不构成安全边界。
- 开发和生产均保留同源 `/api/*` 请求；Next rewrite 只解决开发代理，生产由 Nginx 同源反代。
- 支持 375px 起的响应式布局、44px 触控目标、键盘操作和 `prefers-reduced-motion`。
- 每阶段必须可以独立测试、演示和回滚；功能代码使用 `feature/` 分支、PR、merge、delete、tag 流程。

## Target File Structure

```text
frontend/
  app/
    layout.tsx                    # 服务端根布局、metadata、provider 组装
    loading.tsx                   # 全局稳定加载骨架
    (public)/                     # 公共信息架构，可逐步迁移现有路由
    admin/layout.tsx              # 受保护管理壳层与侧栏
  components/
    app-shell.tsx                 # 顶栏、移动菜单、主内容和页脚
    site-header.tsx               # 品牌与公共导航
    account-menu.tsx              # 登录/用户/退出入口
    admin-nav.tsx                 # 管理子系统导航
    ui/                           # 经项目约束的 shadcn primitives
  lib/
    api/client.ts                 # 统一请求、错误映射、401/refresh
    api/types.ts                  # API DTO
    auth/context.tsx              # 会话状态和动作
    auth/session.ts               # token 生命周期与角色判定
    settings/context.tsx          # public settings 状态
    navigation/registry.ts        # 导航元数据
    navigation/filter.ts          # auth/role/feature 过滤纯函数
  styles/
    globals.css                   # 唯一全局入口与 CSS variables
  tests/
    unit/                         # registry、session、API client
    e2e/                          # 匿名/用户/admin 浏览器流程
```

## Navigation Contract

```ts
type AppRole = 'user' | 'moderator' | 'admin'

type NavigationItem = {
  id: string
  label: string
  href: string
  section: 'public' | 'account' | 'admin'
  auth: 'any' | 'anonymous' | 'authenticated'
  roles?: AppRole[]
  feature?: 'registration' | 'submissions' | 'appeals' | 'admin'
}
```

过滤规则固定为：先 auth，再 roles，再 feature；roles 缺省表示所有已登录角色；未知角色按最低权限处理。registry 只控制入口可见性，路由和 API 仍需后端鉴权。

## Phase A: Baseline and Test Harness

**Deliverable:** 为动态壳层建立可重复测试和构建基线，不改变现有视觉。

**Files:** `frontend/package.json`、lockfile、`vitest.config.ts`、`playwright.config.ts`、`tests/`、CI workflow、frontend Dockerfile。

1. 在 feature 分支锁定 Node/npm 与前端依赖，统一开发、CI、Docker 使用 `npm ci`。
2. 增加 Vitest/Testing Library，先为 navigation filter 和 settings adapter 写失败测试。
3. 增加 Playwright，记录匿名首页、登录、管理员设置入口三条当前基线流程。
4. 修正 production Docker 构建为 deterministic install；确认 Tailwind/PostCSS 只在 builder 中编译，runner 不需要开发依赖执行构建。
5. 验证：`npm run lint`、`npm run test`、`npm run build`、Playwright 三条 smoke 全部通过。

**Acceptance:** 现有页面行为无回归；本地与生产镜像从同一 lockfile 解析依赖；不存在 Tailwind CDN 请求。

## Phase B: Dynamic Foundation

**Deliverable:** 统一的 settings/session/API 边界，可在无闪烁的情况下驱动站点名、登录态和权限。

**Files:** `lib/api/*`、`lib/auth/*`、`lib/settings/*`、`app/layout.tsx`、`app/loading.tsx`。

1. 定义 `PublicSettings`、`SessionUser`、`ApiError` 类型；保留后端响应字段的单一 adapter。
2. 建立 API client：统一 `Content-Type`、非 2xx 错误、401 refresh/登出和 AbortSignal；业务页面不再重复解析错误。
3. 建立 session provider：状态为 `loading | anonymous | authenticated`，暴露 `user`、`hasRole()`、`login()`、`logout()`。
4. 建立 public settings provider：读取 `/api/settings/public`，失败时使用内置 UniBlack 安全默认值并展示非阻断状态。
5. 根布局使用 settings 生成显示名称、描述与 CSS brand token；配置加载期间保持相同尺寸 skeleton，避免“登录/注册关闭”闪烁。
6. 单元测试覆盖 token 缺失、过期、refresh 失败、settings 失败和非法主题色。

**Acceptance:** 登录后无需刷新即可变为用户菜单；退出后受保护入口立即消失；修改 `site.name` 后刷新显示新名称；配置失败仍可查询。

## Phase C: App Shell and Navigation

**Deliverable:** 响应式顶栏、账户菜单、数据化导航和独立管理壳层。

**Files:** `components/app-shell.tsx`、`site-header.tsx`、`account-menu.tsx`、`admin-nav.tsx`、`lib/navigation/*`、`app/admin/layout.tsx`。

1. 写 navigation filter 的失败测试，覆盖匿名、user、moderator、admin 与关闭功能开关。
2. 建立 registry：查询/黑名单对所有人；举报/申诉按登录和功能状态；管理仅 moderator/admin；settings/users/access-lists 仅 admin。
3. 所有站内入口改为 Next `Link`；当前路由使用 `aria-current="page"`。
4. 桌面顶栏显示主要公共任务；已登录显示用户名与退出；移动端使用可键盘操作的抽屉菜单。
5. `/admin` 使用统一侧栏/抽屉，包含审核、案件、用户、系统配置、名单管理；按角色过滤。
6. E2E 验证登录后无整页刷新、退出、角色菜单、直接 URL 访问的 401/403 表达。

**Acceptance:** 五个已知导航问题全部关闭；管理子系统最多两次点击可达；前端隐藏入口但后端仍拒绝越权请求。

## Phase D: Tailwind Design System

**Deliverable:** 与 `DESIGN.md` 对齐的 token、基础组件、完整暗色/响应式状态。

**Files:** `tailwind.config.js`、唯一 `styles/globals.css`、`components/ui/*`、各公共 layout/component。

1. 删除双份 globals 的歧义，确定一个导入入口；把 neutral/brand/semantic/radius/focus 映射为 CSS variables。
2. 引入最小 shadcn primitives：Button、Input、Label、Card、Badge、Alert、Dialog/Sheet、DropdownMenu、Skeleton、Table；不一次性安装未使用组件。
3. 为组件建立交互矩阵：default/hover/focus-visible/active/disabled/error/loading。
4. 把公共页面按“壳层 → 基础组件 → 页面”顺序迁移，避免一次性全站重写。
5. 暗色模式首期跟随系统偏好；所有 surface 和语义色经过对比度检查。
6. 视觉回归检查 375/768/1280px；低端设备禁用大 blur、背景滤镜和持续动画。

**Acceptance:** 页面不再散落品牌 hex；公共和管理页面共用同一 token；无横向溢出；键盘 focus 清晰；reduced-motion 生效。

## Phase E: Page Modernization

**Deliverable:** 动态首页、查询/详情/表单和管理页面统一状态与信息架构。

1. 首页接 `/api/v1/statistics`，以真实数据替代 `-`；统计失败不阻断核心查询。
2. 查询/列表/详情统一搜索、分页、空态、错误态和案件状态词汇；公开 pending 案件明确解释不可见原因。
3. 注册页继续由 settings 驱动 registration/email/captcha，使用共享表单和 field error；关闭注册时给出返回查询入口。
4. 举报/申诉明确步骤、材料要求、保存和提交状态；未登录时先解释再引导登录。
5. 管理列表统一筛选工具栏、分页、批量操作确认和危险动作二次确认。
6. 将纯展示和初始数据尽量留在 Server Components；仅交互岛使用 `'use client'`。

**Acceptance:** 所有数据页都有 loading/empty/error；首页展示真实统计；关键流程在移动端和键盘下完成；页面不直接读取 token。

## Phase F: Deployment, Documentation and Release Gate

**Deliverable:** 开发与生产构建一致、文档准确、可观察且可回滚。

1. 对齐 `.env.example`、compose、Nginx 与 Next：浏览器始终请求相对 `/api`；构建时公开变量与运行时变量边界写入部署文档。
2. CI 执行 lint、typecheck、unit、build；合并前执行 Playwright 关键路径。
3. 记录 bundle 分析：初始 JS 预算以当前基线为上限，新增依赖必须说明成本；按路由动态加载 admin-only UI。
4. 更新 README、roadmap、`docs/frontend-gap-analysis.md` 状态和可验证步骤。
5. 在 375px 移动端与桌面验证 anonymous/user/moderator/admin；验证 Slow 4G、reduced motion 和暗色系统偏好。
6. 通过 feature branch PR 合并，删除分支并创建下一版本 tag；保留前一镜像以便回滚。

**Acceptance:** 本地、CI、production Docker 三条构建路径一致；所有 P0/P1 gap 关闭；用户可按文档查看和验证每个可见能力。

## Explicit Non-Goals

- 不实现由管理员任意输入 URL/组件名的“远程动态菜单”；功能变化通过受类型约束的 registry 发版，避免安全和维护风险。
- 不在此阶段引入 Redux、Zustand 或完整 React Query；出现跨页复杂缓存需求后再以数据证明。
- 不重做后端权限模型；前端只消费现有 session/role，并保留服务端授权。
- 不在基础优化阶段设计复杂页面转场、3D、parallax 或实时协作。
- 不把运行时显示名称改动扩散到代码包名、镜像名、数据库名或 UniBlack 文档身份。

## Decision Trace

| Decision | Reason | Alternatives considered | Tradeoff |
| --- | --- | --- | --- |
| 动态 App Shell | 集中消费站点配置和认证状态，直接解决写死品牌、入口漂移和 hydration 闪烁 | 每页独立读取；所有页面由 middleware 强制服务端化 | 必须认真划分 server/client 边界 |
| CSS variables + Tailwind 映射 | 运行时主题色可热更新，同时保留静态构建、语义 token 和 Tailwind 统一体验 | 动态 class safelist；大量 inline style | 第一阶段需要整理现有色值和 utilities |
| 公共顶栏 + 管理侧栏 | 公共任务保持轻量，管理子系统获得稳定且可扩展的入口 | 所有入口塞入顶栏；全站统一侧栏 | 存在两种导航呈现，但共享同一 registry |
| React Provider/hooks 的轻依赖状态层 | 当前需求是 session/settings 等低频全局状态，无需重型 store | Redux Toolkit；Zustand；React Query 全量接管 | 若未来出现复杂缓存，需以数据重新评估 |
| 系统字体优先 | 无 CDN 依赖、首屏稳定、低端设备成本低，中文回退可控 | Google Fonts；仓库内打包品牌字体 | 字体本身的品牌辨识度更弱 |
| 克制动效 | “丝滑”由无整页刷新、即时反馈和稳定布局实现，符合老设备与 reduced-motion 要求 | 全局页面转场；滚动 reveal；背景动画 | 展示性较低，但信息密度和可靠性更高 |
| 类型化 registry 而非后台任意动态菜单 | 页面随功能发版可扩展，同时保持路由、权限和文案可审查 | DB 驱动任意菜单；插件式远程组件 | 新入口仍需代码发布，不是无代码页面搭建器 |

## Official References Gate

实施每一阶段前，必须重新核对以下官方资料，并在 PR 中记录采用版本；本设计不提前虚构或锁定尚未安装的版本号：

- Next.js App Router navigation：<https://nextjs.org/docs/app/building-your-application/routing/linking-and-navigating>
- Next.js Server and Client Components：<https://nextjs.org/docs/app/building-your-application/rendering/composition-patterns>
- Tailwind CSS Next.js installation：<https://tailwindcss.com/docs/installation/framework-guides/nextjs>
- shadcn/ui Next.js installation：<https://ui.shadcn.com/docs/installation/next>
- WAI-ARIA Authoring Practices：<https://www.w3.org/WAI/ARIA/apg/>
- WCAG 2.2：<https://www.w3.org/TR/WCAG22/>

若官方最新版要求升级 Next/Tailwind，应单独设计升级任务，不与视觉迁移混在同一 PR；优先使用当前 Next.js 14 与 Tailwind 3 的兼容安装方式。

## Verification Matrix

| 场景 | 预期 |
| --- | --- |
| 匿名访问 | 显示登录/注册（注册开启时），不显示管理 |
| 普通用户登录 | 显示用户名、退出、举报/申诉，不显示管理 |
| moderator 登录 | 显示审核管理入口，不显示系统设置/用户/名单管理 |
| admin 登录 | 显示完整管理侧栏，包括 settings/users/access-lists |
| 修改 `site.name` | 公共 Header 和 metadata 使用配置值，代码身份仍为 UniBlack |
| API/settings 失败 | 使用安全默认品牌，显示可恢复错误，不阻断公开查询 |
| token 过期 | 自动 refresh；失败则清理会话并回登录，不显示旧权限入口 |
| 客户端导航 | 使用 Link/router，不发生 document reload |
| 375px | 无横向溢出，菜单可达，关键目标至少 44px |
| reduced motion | 无非必要 transition/动画 |
| production build | Tailwind CSS 已编译进静态产物，无 CDN 请求 |

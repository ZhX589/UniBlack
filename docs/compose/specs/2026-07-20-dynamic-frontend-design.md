# 动态前端与基础页面优化设计规格

## [S1] 目标与范围

把页面级静态实现升级为配置驱动、身份感知、可扩展的 Next.js 应用，并统一公开端、名单档案和管理端设计语言。本阶段不新增后端业务、不迁移 JWT 存储、不建立 CMS、不实现 OAuth/申诉/完整暗色模式。

## [S2] 动态 Shell

- Root Shell 读取公开设置中的站点名称、描述、Logo、主题色和功能开关。
- 设置加载失败稳定回退 `UniBlack`，不能闪现错误业务状态。
- 类型化导航注册表集中声明 `href`、`label`、`audience`、`roles` 和可选 `feature`。
- 内部跳转统一使用 Next.js `Link` 或 Router。

## [S3] 登录态与权限

- AuthProvider 暴露 `loading|anonymous|authenticated`、用户、角色、login 和 logout。
- 从现有 JWT claims 恢复身份；非法/过期 token 清理为 anonymous。
- 登录后返回原受保护路径；退出清理 access/refresh token。
- moderator 可见审核入口；admin 可见审核、用户、名单和配置。
- `/admin` 未登录跳登录，角色不足显示 403；菜单显隐不能替代后端 RBAC。

## [S4] API 与类型边界

- `lib/api.ts` 统一请求、JSON、Authorization、401 和 `ApiError`。
- `lib/types.ts` 定义 PublicSettings、AuthUser、Subject、Case、Submission、AdminUser、AccessList 与分页类型。
- 页面不得自行读 localStorage、拼 token header 或使用业务 `any`。
- 不引入状态库或请求库。

## [S5] 信息架构

公开顶栏包含站点名、查询、名单、举报；访客显示登录/可选注册，登录用户显示用户菜单和退出，有管理角色才显示管理。管理侧栏包含概览/审核、案件、用户、名单、系统设置，并按角色过滤。移动端使用同一注册表生成折叠菜单。

## [S6] 视觉系统

根目录 `DESIGN.md` 是视觉真源：公开端轻量 SaaS，后台数据控制台，名单与详情可信档案。主题色运行时写入 CSS variables，语义风险色固定。Tailwind 只通过本地 PostCSS 构建。

最小组件集：Button、Input、Select、Panel、Badge、Alert、Skeleton、EmptyState、PageHeader、DataTable 容器。不一次性搬入完整 shadcn 组件库。

## [S7] 页面结构

- 首页：品牌说明、主核验输入、真实 statistics、举报次级入口。
- 查询：普通搜索与精确 lookup 分段；查询前是指导，不是“未找到”。
- 名单：筛选分页；桌面表格、移动摘要。
- Subject/Case：档案头、标识符、状态、案件/证据时间线和申诉上下文。
- 认证：独立 Auth Shell；管理：统一 Admin Shell、工具栏和反馈。

## [S8] 状态、响应式与性能

- 数据页覆盖 idle、loading、empty、error、unauthorized、normal。
- 验收 375px、768px、1280px，不得横向溢出。
- 表格在小屏转摘要列表；确需完整列时受控横向滚动。
- 动效限 120–200ms，并支持 `prefers-reduced-motion`。
- 无字体 CDN、大图和大型图表/图标包；captcha 仅注册页按需加载。
- 首页 First Load JS 增量不超过 25KB gzip。

## [S9] Tailwind 与部署

- 合并 globals，只保留 `app/globals.css`；Tailwind 使用语义 token。
- Docker 使用 lockfile + `npm ci`、`.dockerignore` 和一致的 Next 输出策略。
- API rewrite/build 参数必须明确，生产不得意外回退 localhost。
- 开发与生产使用相同 Tailwind/PostCSS 源码。

## [S10] 测试与验收

- Vitest/Testing Library：导航矩阵、Auth、设置回退、主题色安全回退、API 401。
- Playwright：访客导航、登录/退出、admin/moderator 菜单、移动菜单、首页查询和管理入口。
- 运行 lint、typecheck、unit、Next build 和生产 Docker smoke test。
- 验证键盘、focus-visible、aria-current、label 和 reduced motion。

## [S11] 文档与完成定义

- roadmap 新增 Phase 12，当前标记“已设计，待实施”。
- `docs/frontend-gap-analysis.md` 是滚动差距台账。
- 只有测试、build、视口和生产镜像验证全部通过，Phase 12 才能完成。

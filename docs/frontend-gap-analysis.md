# 前端现状与规划差距分析

> 进度台账（非规划规格）。**2026-07-21** 以 `main`（含 next-development merge 与生产 smoke）为准。

全栈对象/事件/验证/治理差距见 `docs/implementation-gap-analysis.md`；本文件保留 Phase 12 前端专项进度，不改写 compose 规划正文。

## 状态定义

- `未实现`：没有对应能力或仅有占位。
- `部分实现`：主路径存在，但状态、权限、响应式或复用性不完整。
- `已实现`：满足 roadmap 验收标准，并有实际构建或交互验证。

## 当前基线

| 领域 | 当前证据 | 状态 | 目标（规划不变） |
| --- | --- | --- | --- |
| 全局导航 | `SiteHeader` + Auth/Settings + `navigationRegistry` | 已实现 | 完整 navigation registry 与功能开关矩阵 |
| 登录态 | Providers 解析 JWT、401 统一 logout、return URL | 已实现 | 统一 401、return URL、token 仅在 providers/api |
| 站点品牌 | 顶栏 `site.name`、主题色 token | 部分实现 | 描述/Logo 更完整驱动 Shell |
| 前端 RBAC | 管理入口按角色；`admin/layout` 守卫 | 已实现 | 声明式注册表（已落地） |
| 管理信息架构 | 侧栏：审核/用户/名单/处罚/归档/设置/（申诉队列） | 部分实现 | 完整数据控制台密度与工具栏 |
| 客户端导航 | `Link`/router 为主 | 已实现 | 业务页统一 Link/router |
| API 访问 | `lib/api.ts`；页面无直连 token/`fetch` | 已实现 | typed API client 统一 auth、401、错误 |
| 类型安全 | `lib/types.ts` 集中领域类型 | 部分实现 | 页面无业务 `any` |
| UI 组件 | Button/Panel/Badge/Alert/State 最小集 | 部分实现 | 最小 Button/Input/Panel/Badge/Table/State 集 |
| Tailwind | CSS variables + semantic colors | 已实现 | CSS token + Tailwind semantic colors |
| 响应式 | e2e 375/768/1280 通过 | 部分实现 | 三档验收、菜单/表格降级 |
| 暗色模式 | 明确 out of scope | 未实现 | 独立阶段 |
| 页面状态 | 多数页用共享 State 组件 | 部分实现 | idle/loading/empty/error/unauthorized 一致 |
| 首页动态数据 | `/api/v1/statistics` | 已实现 | statistics API、降级与主查询入口 |
| 名单档案 | 列表/详情 + Event 时间线 | 部分实现 | 分页筛选、小屏摘要 |
| 管理控制台 | 侧栏 + 处罚/归档/设置等 | 部分实现 | 数据控制台 Shell、工具栏 |
| 可访问性 | focus-visible、min-h-touch、reduced-motion | 部分实现 | WCAG AA 系统验收 |
| 老旧设备 | reduced-motion；无完整浏览器矩阵文档 | 部分实现 | 主流浏览器最近两版 |
| 前端测试 | Vitest + Playwright 21/21 | 已实现 | Vitest + Playwright |
| Docker 构建 | `npm ci`；生产 Compose smoke 通过 | 已实现 | lockfile 构建与镜像 smoke |
| 文档 | 台账随 main 更新 | 部分实现 | 与部署说明持续同步 |

## 页面差距

| 页面 | 当前 | 主要差距 | 目标视觉 |
| --- | --- | --- | --- |
| `/` | 统计 + 核验入口 | 品牌细节 | 轻量 SaaS |
| `/search` | 查询可用 | 空态体验打磨 | 结果可信清晰 |
| `/subjects` | 列表可用 | 分页筛选 | 可信档案列表 |
| `/subjects/[id]` | 详情 + Event 时间线 | 证据层级密度 | 对象档案 |
| `/events/[id]` | 详情 + 申诉入口 | 证据展示更完整 | 事件档案 |
| `/cases/[id]` | 兼容详情 | 明确兼容跳转 | Sunset 后下线 |
| `/submit` | multipart 发布 | 状态组件统一 | 清晰分区表单 |
| `/login`、`/register` | Auth + demo captcha | 错误态统一 | 简洁认证 Shell |
| `/sanctions` | 我的处罚 + 一次申诉 | — | 治理自助页 |
| `/setup` | 初始化 | Setup Shell 抛光 | 独立 Setup Shell |
| `/admin/*` | 守卫 + 侧栏 + 治理页 | 表格密度 | 数据控制台 |
| `/admin/appeals` | 事件申诉队列 | 批量操作 | 审核工作台 |

## Roadmap 功能差距

| Roadmap 项 | 判断 | 说明 |
| --- | --- | --- |
| Phase 7 申诉页面 | 部分实现 | 处罚申诉 `/sanctions`；事件申诉 `/events/[id]` + 管理队列 |
| Phase 8 OpenAPI | 延后 | 决策：不阻塞发布；契约见 README + routes + `docs/api/*` |
| Phase 8 API Key | **不做** | 产品决策锁定，见 `AGENTS.md` |
| Phase 9 统一设计语言 | 部分实现 | token + 最小组件已有；非完整 design system |
| Phase 9 深色模式 | 未实现 | Phase 12 out of scope |
| Phase 9 响应式 | 部分实现 | e2e 三档有证据；表格降级可再加强 |
| Phase 10 OAuth | 仅配置预留 | 无 provider 登录流程 |
| Phase 11 用户角色/重置密码 | 未完整实现 | 列表与启停为主 |
| Phase 11 名单批量导入导出 | 对象归档 ZIP 已有 | 用户名单批量 CSV 等未做 |

## 本轮范围

纳入 Phase 12：动态 Shell、Auth/Settings/API 架构、RBAC 导航、管理信息架构、Tailwind 设计系统、基础页面重塑、响应式、测试和部署构建。

不纳入：后端认证协议迁移、通用 CMS、任意后端菜单、完整深色模式、OAuth、HttpOnly Cookie 迁移。

## 更新规则

每项改为“已实现”时必须附自动化测试、`npm run build`、Playwright 场景或生产镜像 smoke test 证据。不得仅因文件存在而标记完成。

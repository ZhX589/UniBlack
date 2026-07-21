# 前端现状与规划差距分析

> 进度台账（非规划规格）。基线曾为 `1719cab`；**2026-07-21** 起进度以 `main@b10329a` 为准，并叠加 `feature/next-development`（tip `81d3615`）：API client、navigation、tokens、页面级 fetch 清零、Event 详情链路、e2e 脚手架。

全栈对象/事件/验证/治理差距见 `docs/implementation-gap-analysis.md`；本文件保留 Phase 12 前端专项进度，不改写 compose 规划正文。

## 状态定义

- `未实现`：没有对应能力或仅有占位。
- `部分实现`：主路径存在，但状态、权限、响应式或复用性不完整。
- `已实现`：满足 roadmap 验收标准，并有实际构建或交互验证。

## 当前基线

| 领域 | 当前证据（main@b10329a） | 状态 | 目标（规划不变） |
| --- | --- | --- | --- |
| 全局导航 | `SiteHeader` + Auth/Settings；登录/角色过滤 | 部分实现 | 完整 navigation registry 与功能开关矩阵 |
| 登录态 | `Providers` 解析 JWT、退出、清理坏 token；登录后顶栏变化 | 部分实现 | 统一 401 处理、return URL 全覆盖、减少页面级 localStorage |
| 站点品牌 | 顶栏读 `site.name`；metadata 仍偏静态 | 部分实现 | 描述/Logo/主题色完整驱动 Shell |
| 前端 RBAC | 管理入口按角色隐藏；`admin/layout` 403/跳转登录 | 部分实现 | 更细权限与菜单声明式注册表 |
| 管理信息架构 | 侧栏：审核/用户/名单/处罚/归档/设置 | 部分实现 | 完整数据控制台密度与工具栏规范 |
| 客户端导航 | 壳层与登录/注册多用 `Link`；部分业务页仍可能有原生跳转 | 部分实现 | 业务页内部统一 `Link`/router |
| API 访问 | `lib/api.ts` + 401 统一 logout；公开/管理/提交页已迁移，页面无直连 token | 已实现（unit+build 证据） | typed API client 统一 auth、401、错误 |
| 类型安全 | `lib/types.ts` 已集中领域类型；管理页仍有 `any` | 部分实现 | 集中领域类型，页面无业务 `any` |
| UI 组件 | Button/Panel/Badge/Alert/State 最小集已落地 | 部分实现 | 最小 Button/Input/Panel/Badge/Table/State 集 |
| Tailwind | CSS variables + semantic colors 已映射 | 部分实现 | CSS token + Tailwind semantic colors |
| 响应式 | 管理布局有 grid 降级；未做 375/768/1280 正式验收 | 部分实现 | 三档验收、菜单/表格降级 |
| 暗色模式 | 仍不完整 | 未实现 | 独立阶段 |
| 页面状态 | 部分页仍不统一 | 部分实现 | idle/loading/empty/error/unauthorized 一致 |
| 首页动态数据 | 接入 `/api/v1/statistics`；失败不阻断核验 | 部分实现 | statistics API、降级与主查询入口 |
| 名单档案 | 列表/详情可用；公开 ID 链路增强中 | 部分实现 | 档案层级、分页筛选、小屏摘要 |
| 管理控制台 | 侧栏+处罚/归档页已加；表格密度仍粗 | 部分实现 | 数据控制台 Shell、工具栏和操作反馈 |
| 可访问性 | 导航有 `aria-current`；未系统验收 | 部分实现 | WCAG AA、键盘流、44px、reduced motion |
| 老旧设备 | 无正式兼容矩阵 | 未实现 | 主流浏览器最近两版；禁用非必要动效 |
| 前端测试 | Vitest 通过；Playwright 21/21 通过（系统 Chrome，375/768/1280 项目） | 已实现（e2e 证据） | Vitest/Testing Library + Playwright |
| Docker 构建 | Dockerfile 已改 `npm ci`；生产 smoke 待补 | 部分实现 | `npm ci`、明确 build args、镜像 smoke |
| 文档 | 进度台账已随合并更新 | 部分实现 | 与部署说明持续同步 |

## 页面差距

| 页面 | 当前（main） | 主要差距 | 目标视觉 |
| --- | --- | --- | --- |
| `/` | 静态介绍；统计仍可能占位 | 真实 statistics、品牌配置、核验主交互 | 轻量 SaaS，核验框为焦点 |
| `/search` | 查询可用 | 空态/loading 统一、类型与详情链接体验 | 轻量 SaaS，结果可信清晰 |
| `/subjects` | 列表可用 | 分页筛选、小屏摘要、档案密度 | 可信档案列表 |
| `/subjects/[id]` | 详情可用；public ID 链路增强中 | Event 时间线文案、证据层级 | 对象档案与事件时间线 |
| `/cases/[id]` | 旧案件详情仍在 | 事件化或兼容跳转；与 Event 模型对齐 | 可追溯事件档案 |
| `/submit` | 分段对象/账号/事件、文本+多文件 multipart、验证码/demo、未登录灰态 | 统一 API client、状态组件、更少 localStorage | 清晰分区表单 |
| `/login`、`/register` | Auth 接入、Link、demo captcha | 品牌/错误态与 Auth Shell 完全统一 | 简洁认证 Shell |
| `/sanctions` | 我的处罚 + 一次申诉 | 管理端申诉队列、更完整反馈 | 治理自助页 |
| `/setup` | 初始化可用 | 独立首次启动 Shell | 独立 Setup Shell |
| `/admin/*` | layout 守卫+侧栏；处罚/归档/设置/用户/名单 | 表格密度、工具栏、品牌分组、响应式验收 | 数据控制台 |

## Roadmap 功能差距

| Roadmap 项 | 判断 | 说明 |
| --- | --- | --- |
| Phase 7 申诉页面 | 部分实现 | 处罚申诉页 `/sanctions` 已有；事件申诉前端入口仍弱 |
| Phase 8 OpenAPI | 未核验 | README 端点表不等于自动生成 OpenAPI |
| Phase 8 API Key | 决策冲突 | roadmap 要求 API Key，现有架构决定不实现，需另行决策 |
| Phase 9 统一设计语言 | 未实现 | 无 token 和组件体系 |
| Phase 9 深色模式 | 未实现 | 当前媒体查询与白色面板冲突 |
| Phase 9 响应式 | 部分实现 | 无移动导航和后台表格策略 |
| Phase 10 OAuth | 仅配置预留 | 无 provider 登录流程 |
| Phase 11 用户角色/重置密码 | 未完整实现 | 当前用户页主要是列表与启停 |
| Phase 11 名单导入导出 | 未实现 | 当前只有增删与筛选 |

## 本轮范围

纳入 Phase 12：动态 Shell、Auth/Settings/API 架构、RBAC 导航、管理信息架构、Tailwind 设计系统、基础页面重塑、响应式、测试和部署构建。

不纳入：后端认证协议迁移、通用 CMS、任意后端菜单、完整深色模式、申诉新业务、OAuth、名单批量导入导出。

## 更新规则

每项改为“已实现”时必须附自动化测试、`npm run build`、Playwright 场景或生产镜像 smoke test 证据。不得仅因文件存在而标记完成。

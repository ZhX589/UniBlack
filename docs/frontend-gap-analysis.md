# 前端现状与规划差距分析

> 基线：2026-07-20，commit `1719cab`。本文件是滚动差距台账；每次前端阶段完成后更新状态与验证证据。

全栈对象/事件/验证/治理差距见 `docs/implementation-gap-analysis.md`；本文件保留 Phase 12 前端专项差距，不替代 Phase 13 领域审计。

## 状态定义

- `未实现`：没有对应能力或仅有占位。
- `部分实现`：主路径存在，但状态、权限、响应式或复用性不完整。
- `已实现`：满足 roadmap 验收标准，并有实际构建或交互验证。

## 当前基线

| 领域 | 当前证据 | 状态 | 目标 |
| --- | --- | --- | --- |
| 全局导航 | `app/layout.tsx` 固定 5 个 `<a>` | 未实现 | 登录态、角色和功能开关过滤的导航注册表 |
| 登录态 | 约 14 处 `localStorage` 读写；登录后顶栏不变 | 部分实现 | AuthProvider 统一用户、退出、401 和 return URL |
| 站点品牌 | metadata、顶栏、首页写死 `UniBlack` | 未实现 | `site.name`、描述、Logo、主题色驱动 Shell |
| 前端 RBAC | 管理对访客可见；管理页只检查 token | 未实现 | 角色菜单、管理守卫和 403 页面 |
| 管理信息架构 | settings/users/access-lists 无统一入口 | 未实现 | 管理侧栏覆盖审核、案件、用户、名单、设置 |
| 客户端导航 | 约 16 个内部 `<a>`，多处硬跳转 | 未实现 | 内部统一 `Link` / Router |
| API 访问 | 约 23 处页面内 fetch | 未实现 | typed API client 统一 auth、401、错误 |
| 类型安全 | 列表和设置多处 `any` | 部分实现 | 集中领域类型，页面无业务 `any` |
| UI 组件 | `components/ui/` 为空 | 未实现 | 最小 Button/Input/Panel/Badge/Table/State 集 |
| Tailwind | npm + PostCSS 已接入，theme 近乎空白 | 部分实现 | CSS token + Tailwind semantic colors，无 CDN |
| 响应式 | 少量断点；导航和后台表格窄屏不可用 | 部分实现 | 375/768/1280 验收，菜单和表格降级 |
| 暗色模式 | body 随系统变暗，面板仍固定白色 | 未实现 | 本阶段移除不完整 dark，后续独立实现 |
| 页面状态 | loading/empty/error 风格不一 | 部分实现 | idle/loading/empty/error/unauthorized 一致 |
| 首页动态数据 | 统计均为 `-` | 未实现 | statistics API、降级与主查询入口 |
| 名单档案 | 基础 table + 详情可访问 | 部分实现 | 档案层级、分页筛选、小屏摘要 |
| 管理控制台 | 功能页存在，表格表单重复 | 部分实现 | 数据控制台 Shell、工具栏和操作反馈 |
| 可访问性 | 缺统一 focus、aria-current、错误关联 | 未实现 | WCAG AA、键盘流、44px、reduced motion |
| 老旧设备 | 无兼容基线和性能预算 | 未实现 | 主流浏览器最近两版；禁用非必要动效 |
| 前端测试 | 无组件或端到端测试 | 未实现 | Vitest/Testing Library + Playwright |
| Docker 构建 | `npm install`；无 `.dockerignore`；rewrite env 时机不清 | 部分实现 | `npm ci`、明确 build args、镜像 smoke test |
| 文档 | roadmap 的响应式/统一语言无验证证据 | 部分实现 | 规范、计划、部署说明和差距台账同步 |

## 页面差距

| 页面 | 当前 | 主要差距 | 目标视觉 |
| --- | --- | --- | --- |
| `/` | 静态介绍、占位统计 | 无真实统计、品牌配置、核验交互 | 轻量 SaaS，核验框为焦点 |
| `/search` | 两表单共用 query | 初始即空态、无详情链接、`any` | 轻量 SaaS，结果可信清晰 |
| `/subjects` | 固定第一页表格 | 无分页筛选、小屏溢出 | 可信档案列表 |
| `/subjects/[id]` | 基本资料与案件 | 层级和来源弱 | 对象档案与案件时间线 |
| `/cases/[id]` | 基本案件详情 | 证据/申诉上下文弱 | 可追溯案件档案 |
| `/submit` | 基础举报表单 | 页面内鉴权、反馈弱 | 清晰分区表单 |
| `/login`、`/register` | 功能可用 | 品牌硬编码、链接和错误态不统一 | 简洁认证 Shell |
| `/setup` | 初始化可用 | 与普通 Shell 混用 | 独立首次启动 Shell |
| `/admin/*` | 四个孤立页面 | 无统一入口、守卫和响应式 | 数据控制台 |

## Roadmap 功能差距

| Roadmap 项 | 判断 | 说明 |
| --- | --- | --- |
| Phase 7 申诉页面 | 未实现 | 后端流程存在，前端无独立申诉入口与记录页 |
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

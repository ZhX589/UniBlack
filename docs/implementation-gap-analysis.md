# UniBlack 实现与设计目标差距分析

> 进度台账（非规划规格）。记录当前真实实现与已确认设计目标之间的差距；不修改 Phase 12/13 规划目标本身。

> **合并进度（2026-07-21）**：`feature/subject-event-governance` 已合入 `main`（merge `49e4ddc`，功能 tip `c6f6d42`）。合并后在 main 上验证：`go test ./...`、`go build ./cmd/server`、`npx tsc --noEmit`、`npm run build` 通过。
>
> **已落地并验证**：迁移 `000006`–`000010`、`UBS_<ULID>`、Account/Event 兼容层、真实 LocalStorage、multipart 多文件+文本同事务发布、Event 独立文件上传、处罚列表/撤销/一次申诉/管理裁决、demo captcha、归档导出/预览/确认导入加固、动态 Auth/Settings Shell 与管理侧栏。
>
> **仍未完成（以 `feature/next-development` 为准，见文末 2026-07-21 进度）**：Playwright 浏览器实测、生产 Compose/Nginx smoke、旧 Submission 审核 UI 兼容窗口收束。

## 阅读规则

- `已实现`：代码存在，并有本地/CI/接口/构建证据。
- `部分实现`：主路径存在，但行为、数据完整性、权限或运行可靠性不满足设计。
- `未实现`：只有旧接口、占位代码、文档设想或页面入口。
- `过时文档`：文档描述与代码实际行为不一致，需要修正描述，但不改变未来目标。

## 目标文档真源

- Phase 12 动态前端设计：`docs/compose/specs/2026-07-20-dynamic-frontend-design.md`
- Phase 13 对象/事件/验证/治理设计：`docs/compose/specs/2026-07-20-subject-event-governance-design.md`
- Phase 13 原始实施任务计划：`docs/compose/plans/2026-07-20-subject-event-governance.md`
- 本轮开发排序计划：`docs/compose/plans/2026-07-20-subject-event-governance-execution.md`
- 前端视觉规范：`DESIGN.md`

本文件与现有路线的关系：只记录差距、阻塞点、证据和执行顺序；不删除、不重写 Phase 12/13 的目标。

## 总体判断

| 领域 | 当前状态 | 主要证据 | 结论 |
| --- | --- | --- | --- |
| 对象模型 | 部分实现 | `public_id`、`accounts`、兼容读取已实现 | 旧 Identifier 仍在兼容窗口 |
| 事件模型 | 部分实现 | `events` 表、默认发布与兼容 Case 回填已实现 | 旧 Case API 仍是历史主路径 |
| 对象通用名规则 | 部分实现 | 新发布路径 `ResolveDisplayName` 已用第一条账号用户名 | 旧 Submission 审核路径仍可能写“待补充” |
| 公开对象 ID | 已实现 | `UBS_<ULID>`、唯一索引、归档命名和 public 查询已验证 | 历史对象保留 UUID 兼容回填值 |
| 账号字段 | 部分实现 | `accounts` 表含 username/account_id/custom_attributes，规范化唯一索引已落地 | 旧 Identifier 兼容读取仍保留 |
| 默认发布 | 部分实现 | multipart 发布可同事务写入文本+多文件证据，导出 ZIP 含 T/F 键，已 API smoke | 旧 Submission 审核兼容保留；链接证据随发布待补 |
| 事件证据 | 部分实现 | Event 文本/文件归档键、SHA-256、可见性检查已 smoke | 旧 Case 证据路径仍兼容 |
| 真实文件存储 | 部分实现 | LocalStorage 真实写盘；发布/导入补偿删除已 smoke | MinIO/S3 adapter 待补 |
| 导出/导入包 | 部分实现 | ZIP、manifest v1、README、强制哈希、命名空间、冲突预览与确认导入已 smoke | 导入来源记录、异步大包任务待补 |
| 申诉结论 | 部分实现 | Event appeal outcome + 处罚一次申诉/裁决已 smoke | malicious 自动建处罚、事件版本历史待补 |
| 分级处罚 | 部分实现 | 列表/创建/撤销/发布拦截/审计/用户申诉/管理裁决已 smoke | 申诉管理列表 UI 仍可加强 |
| 人机验证 | 部分实现 | 无第三方 URL；demo token 按 purpose/IP 或 JWT 用户绑定且单次使用已 smoke | appeal UI/API 与发送限流待补 |
| 邮箱开发模式 | 部分实现 | development 固定 `123456`；register/submission 发码入口已实现 | appeal 发码 UI 与频率限制待补 |
| 邮箱生产模式 | 部分实现 | SMTPMailer 支持 SSL/认证；缺 host 失败；purpose 已区分 | 发送频率限制待补 |
| 配置 OptionMap | 已实现基础 | `setting/options.go`、Bootstrap、schema/settings/values | 可补 demo mode 文案与证据上限配置 |
| 统一控制台 | 部分实现 | admin 侧栏含审核/用户/名单/处罚/归档/设置 | 品牌分组与更完整治理 IA 待补 |
| 前端动态化 | 部分实现 | Auth/Settings Provider、角色导航、管理守卫、Link 顶栏已实现并构建 | 完整 design token、Playwright 矩阵、页面级 API client 统一待补 |
| 测试 | 部分实现 | 后端 domain/export/service/storage/captcha 单测与 API smoke | 前端 unit/e2e 仍缺 |
| 部署可靠性 | 部分实现 | LocalStorage 已写盘；前端生产 build 通过 | Docker `npm ci`/rewrite 构建参数仍待核对 |

## 后端逐项差距（相对 main@b10329a）

### 1. Subject 与 Account

**已有**：`subjects.public_id`（`UBS_<ULID>`）、`accounts` 表（platform/username/account_id/custom_attributes、规范化唯一索引）、新发布路径的 `ResolveDisplayName`。

**仍缺**：旧 `identifiers` 与公开查询/部分页面仍兼容并存；需统一读写 adapter 或弃用窗口后收束 Identifier API。

### 2. Case 到 Event

**已有**：`events` 表、兼容 Case 回填、`/api/subjects/publish`、Event 读写与证据关联、`legacy_case_id`。

**仍缺**：旧 `cases` 表与 `/api/cases`、`/api/v1/cases`、Submission/Evidence/Appeal 的 Case 外键路径仍在；前端部分文案仍为「案件」。差距是**双轨收束与弃用**，不是「Event 尚未建模」。

### 3. Submission / 发布行为

**已有**：`POST /api/subjects/publish`（JSON 或 multipart），事务内创建 subject/accounts/events/text+file 证据索引，存储失败可补偿删除；发布前检查有效处罚。

**仍缺**：旧 `CreateSubmission`（pending 审核、「待补充」）仍保留兼容；链接证据随发布、旧审核 UI 退役说明。

### 4. Evidence 与 Storage

**已有**：Event 证据 `event_id` + `storage_key` + SHA-256；文本 UTF-8/200KiB；文件归档键 `subjects/<publicID>/evidence/...`；LocalStorage 真实写盘；归档 ZIP/manifest 哈希校验与确认导入。

**仍缺**：旧 Case 证据路径仍可用；MinIO/S3 adapter；生产对象存储切换与健康检查。

### 5. Appeal 与 Sanction

**已有**：Event 申诉 outcome（含 corrected/withdrawn 等）；`sanctions` + `sanction_appeals`；列表/创建/撤销/用户一次申诉/管理裁决；发布拦截。

**仍缺**：`malicious_submission` 自动建处罚；事件版本历史；处罚申诉管理列表 UI 可再加强；事件申诉前端入口仍弱。

### 6. Captcha 与邮箱

**已有**：运行时仅 demo captcha（无第三方脚本/siteverify）；register/submission token 绑定；development 固定 `123456`；生产缺 SMTP 失败；purpose 区分；submission 发码需登录。

**仍缺**：发送频率限制；appeal 发码 UI；配置台 demo 模式文案与证据上限等 OptionMap 补全。

## 前端逐项差距（Phase 13 相关）

页面级细节以 `docs/frontend-gap-analysis.md` 为准。与 Phase 13 相关的当前状态：

| 页面/能力 | 当前实现（main） | 剩余差距 |
| --- | --- | --- |
| `/submit` | 分段对象/账号/事件、文本+多文件 multipart、验证码/demo captcha、未登录灰态引导 | 仍读 localStorage token；状态组件未统一 |
| 对象列表/详情 | 列表与 `/subjects/[id]` 可用；public ID 链路增强中 | 档案层级、Event 时间线文案、分页筛选 |
| `/cases/[id]` | 旧案件详情仍在 | 事件化或兼容跳转说明 |
| 注册 | demo captcha 卡；无第三方脚本 | 品牌/错误态与 Auth Shell 统一 |
| 管理 | 侧栏含处罚、归档导入导出、设置等 | 品牌分组、表格密度、事件申诉管理页 |
| `/sanctions` | 我的处罚 + 一次申诉 | 管理端申诉队列 UI |

## 文档一致性说明

1. 本文件总表与上文「后端逐项」已按合入后代码对齐；`docs/compose/specs/*`、`plans/*` 规划正文不因进度改写。
2. `docs/configuration.md` 若仍混有「第三方 captcha 运行时」表述，应以「当前为 demo；Catalog 仅保留接入契约」为准逐步校正配置说明（非改规划 Goal）。
3. Phase 8–11 的「完成」只代表当时基线能力，不覆盖 Phase 12/13 全部验收。

## 不变的未来路线

以下目标保持原样，本文件不做削弱或替换：

- Phase 12：动态前端、Auth/Settings/API 架构、RBAC 导航、Tailwind 设计系统、响应式和构建验收。
- Phase 13：对象/账号/事件、`UBS_<ULID>`、混合存储、默认发布、JSON 包、申诉更正、分级处罚、演示 captcha、开发固定验证码、生产 SMTP、统一控制台。

## 差距结论

## 2026-07-21 后续进度（feature/next-development）

**分支 tip**：`main` @ merge `90ef0c2` + 生产 smoke 修复提交。开发产物已合入 `main` 并推送 `origin/main`。

### 已验证并交付

| 区域 | 证据 |
| --- | --- |
| Case API 弃用窗口 | 弃用头 + `docs/api/case-event-migration.md`；Sunset 2026-12-31；管理端旧 UI 明确标为兼容窗口 |
| MinIO/S3 + 生产 fail-closed 存储 | `storage/s3.go`、`selectStorage` 测试；生产 Compose 使用 MinIO 启动成功 |
| 启动/初始化/访问控制 | 原子 setup、黑白名单、动态限速、迁移 CI |
| Event 治理闭环 | 链接证据随发布、Event-first 申诉、malicious→warning、Account-first 查询、真实 statistics、迁移 `000011`；`go test ./...` 通过 |
| 前端共享边界 | `lib/api.ts` / navigation registry / design tokens / 最小 UI；unit/typecheck/lint/build 通过 |
| 页面级 token/fetch 清零 | 全站页面不直连 token/`fetch`（仅 `providers` 持 token） |
| Playwright 角色/视口 | `E2E_ALLOW_DEFAULT_USERS=1 npm run test:e2e` → **21 passed**（系统 Chrome，desktop/tablet/mobile） |
| 生产 Compose/Nginx smoke | `docker compose -f docker-compose.prod.yml --env-file .env.smoke up --build -d` 后：`/` 200 HTML、`/api/settings/public` 200、`/api/v1/statistics` 200、`/api/setup/initialize` 200、`/api/auth/login` 200+tokens、`/api/admin/settings` 200、`/_next/static/css/*` 200 |

### 有意保留（非未完成）

```text
旧 Submission/Case 管理入口：兼容窗口至 2026-12-31 Sunset（见 docs/api/case-event-migration.md）
  管理页文案标明“兼容”，新内容走 /submit Event 发布
```

### 计划文档

- 总路线：`docs/compose/plans/2026-07-21-next-development.md`
- 收尾包：`docs/compose/plans/2026-07-21-remaining-completion.md`
- 产品决策：`docs/product-decisions.md`、`AGENTS.md`

## 2026-07-21 文档对齐 + 体验收口（`docs/gap-and-completion-2026-07-21`）

| 区域 | 状态 | 证据 |
| --- | --- | --- |
| 文档真源（AGENTS / roadmap Status / frontend-gap / database-design 注记） | 已对齐 | 本分支 diff |
| API Key / OpenAPI / OAuth 决策 | 已锁定（不做 / 延后 / 仅预留） | `docs/product-decisions.md` |
| 邮件验证码发送频率限制 | 已实现 | 同 email+purpose 60s 冷却；429；`auth_verification_rate_test.go` |
| Event 申诉用户入口 | 已实现 | `frontend/app/events/[id]/page.tsx` |
| Event 申诉管理队列 | 已实现 | `frontend/app/admin/appeals` + nav registry |
| Case 兼容收束检查清单 | 文档完成（非立即删表） | `docs/product-decisions.md` Sunset checklist |
| 后端 `go test ./...` | 通过 | 本分支验证 |
| 前端 unit / lint / typecheck | 通过 | Vitest 10/10、eslint clean |

## 2026-07-21 可选后续（`feature/post-merge-followups`）

| 项 | 状态 |
| --- | --- |
| 清理 `.worktrees/next-development` + 本地 `feature/next-development` | 已删除 |
| README Event-first API 表 + 遗留 Case 标注 | 已更新 |
| Sunset 检查清单细化（已完成/待办分栏） | `docs/product-decisions.md` |
| Playwright：事件申诉队列 + Case 软跳转 | `e2e/appeals-and-compat.spec.ts` + navigation 扩展 |
| OAuth / 暗色 / OpenAPI / API Key | **明确不做**（决策锁定） |
| Case 表物理删除 | **禁止直至 Sunset 检查清单完成** |

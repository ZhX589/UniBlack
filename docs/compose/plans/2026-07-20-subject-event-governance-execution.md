# Subject Event Governance Execution Plan

> **For agentic workers:** This is a development-order plan only. It does not change the approved Phase 13 design or replace `docs/compose/plans/2026-07-20-subject-event-governance.md`.

**Goal:** 根据当前实现差距，以最少返工顺序逐步实现已确认的对象/事件/验证/治理路线。

**Architecture:** 先建立兼容数据层和真实文件存储，再接入验证契约和默认发布事务，之后实现申诉处罚、导出和控制台，最后迁移前端文案与动态交互。每个阶段保留旧 Case/Submission API 的兼容窗口，直到新读取路径和导出验证通过。

**Tech Stack:** Go、Echo、GORM、PostgreSQL、golang-migrate、本地/S3-compatible Storage、Next.js 14、React 18、Tailwind CSS。

## Global Constraints

- 不修改已确认的 Phase 12/13 目标、术语、状态和非目标。
- 不直接删除 `cases`、`identifiers` 或旧 API；使用向前迁移和兼容 adapter。
- 不在完成 Storage 真实写入前实现“成功导出”。
- 不在完成处罚查询前把默认发布入口暴露给用户。
- 不接入真实第三方 captcha；配置契约可以保留，运行时只使用 demo provider。
- `APP_ENV=development` 固定邮箱验证码为 `123456`；非开发环境缺 SMTP 必须失败。
- 每阶段使用 `feature/subject-event-governance` 或其子分支，通过测试、构建、接口或 smoke 证据才能进入下一阶段。

## 当前阻塞与依赖

| 阻塞 | 影响 | 解除阶段 |
| --- | --- | --- |
| Subject 无 public ID | URL、文件名、导出包无稳定身份 | A |
| Case/Identifier 旧结构 | Event/Account 无法表达 | A/B |
| LocalStorage 只返回 placeholder URL | 文件和 txt 证据不可恢复 | B |
| Submit 仍 pending 审核 | 默认发布无法接入 | C |
| 真实 captcha provider | 与 demo 目标冲突 | C |
| LogMailer 降级 | 生产邮箱验证可绕过 SMTP 配置 | C |
| 无 sanction 查询 | 默认发布无治理闭环 | D |
| 无 archive package | 导出/导入无法验收 | E |
| 前端静态壳层 | 新资源无法统一入口 | F |

## Phase A: 迁移前基线与兼容读取

**目标：** 在不改变当前可用接口行为的前提下，让数据库可以承载新目标。

**范围：**

- 添加 `subjects.public_id`，为历史对象生成稳定迁移值；新对象生成 `UBS_<ULID>`。
- 新增 `accounts`、`events`、版本/修订所需最小索引。
- 定义旧 `Identifier` 到新 `Account` 的只读 adapter，不立即删除旧表。
- 定义旧 `Case` 到新 `Event` 的只读兼容映射。
- 为迁移写唯一约束、时间字段、状态约束和回滚说明。

**完成证据：**

- migration up/down 在空库和已有开发库各执行一次。
- ULID 唯一性、历史对象 public ID、账号去重和事件时间约束测试通过。
- 旧 `/api/v1/cases` 和新只读 Event 查询都能读取同一条历史数据。

## Phase B: 真实 Storage 与证据索引

**目标：** 先让文件和文本证据真实可恢复，再设计导出。

**范围：**

- 修复 LocalStorage 真实写盘，保证目录创建、原子临时文件、删除和读取。
- 为未来 S3/MinIO 保持 `Storage` 接口，Storage key 不使用用户原始文件名。
- Evidence 增加 storage key、original filename、event relation、UTF-8/text 元数据。
- 文件命名采用对象 public ID + 事件序号 + 证据序号。
- 文本证据限制 200 KiB、验证 UTF-8、写入 `.txt`、计算 SHA-256。
- 下载按 Event 可见性检查，不能只按证据 ID 返回。

**完成证据：**

- 上传后进程重启仍能读取文件。
- 文本内容能从 `.txt` 恢复，超限和非法 UTF-8 被拒绝。
- 同内容哈希、删除、路径穿越和错误权限有测试。

## Phase C: 验证契约与默认发布入口

**目标：** 在默认发布之前完成验证与环境隔离。

**范围：**

- 新增 Demo captcha token store，按 purpose/session/过期时间/单次使用验证。
- 删除注册/提交路径对第三方脚本和 siteverify 的运行依赖；保留配置字段与控制台说明。
- development 邮箱验证只接受 `123456`，不使用 LogMailer。
- production/staging 缺 SMTP 明确失败；配置 SMTP 后使用随机一次性验证码。
- 区分 `register`、`submission`、`appeal` purpose，并加入发送频率限制。
- 将验证接口接入注册和新提交入口，错误返回明确且不泄露密钥。

**完成证据：**

- 测试中 HTTP client 不会访问三家 captcha endpoint。
- development 错误验证码失败，`123456` 成功。
- 非开发无 SMTP 发送失败；SMTP fake server 可收到邮件并完成一次性校验。
- 验证 token 不能跨 purpose、跨 session 或重复使用。

## Phase D: 默认发布、申诉结论与处罚

**目标：** 形成可追责的公开发布闭环。

**范围：**

- 新建 verified subject/event publish service：对象、账号、事件、证据索引和审计在一个事务内创建。
- 通用名为空时使用第一条账号 username；无 username 时拒绝。
- 新提交默认 `published`；保留旧 Submission review 作为兼容入口。
- 申诉资源从 Case 兼容读取迁移到 Event 资源；结论支持 upheld/corrected/withdrawn/malicious_submission。
- 新增 warning/submission_suspension/submission_ban 和有效期/撤销审计。
- 发布前检查有效处罚；只有 admin 能创建/撤销处罚，自动化不判断恶意。

**完成证据：**

- 重复账号时整个事务回滚，不留下半成品对象。
- 正常用户可以发布公开事件；被处罚用户被拒绝并得到明确原因。
- 申诉修正/撤销保留历史和审计；处罚过期/撤销后行为恢复。
- 旧 Case 页面和新 Event 页面在兼容窗口都能说明状态。

## Phase E: JSON 归档、导入预览与后台治理

**目标：** 让对象数据能够可校验导出，同时提供治理入口。

**范围：**

- `manifest.json` schema version 1、中文 `README.txt`、证据文件和 SHA-256。
- 导出只包含允许公开/管理员可见的数据，不包含 secret、JWT、验证码和内部敏感字段。
- 导入先 schema/哈希/public ID 冲突预览，默认不覆盖；确认接口单独执行。
- Admin 统一入口增加对象、事件、申诉、处罚、审计、归档、站点与验证设置。
- Settings Catalog 补齐 demo captcha mode、邮箱环境策略、文件上限和导出配置。

**完成证据：**

- 导出的 zip 可解压，manifest schema 为 1，所有哈希匹配。
- 已存在 public ID 的导入只能显示冲突预览，不会覆盖。
- admin/moderator 权限矩阵和危险操作确认测试通过。

## Phase F: 前端提交与动态信息架构

**目标：** 最后把已稳定的后端能力接入用户可见流程，避免 UI 先于数据模型反复返工。

**范围：**

- 使用 Phase 12 Shell、AuthProvider、SettingsProvider 和导航 registry。
- `/submit` 未登录时显示灰态说明；已登录时显示对象/账号/事件/证据/验证/发布分区。
- 对象详情使用 `UBS_<ULID>`、账号档案和事件时间线；旧 case 链接提供兼容跳转。
- 注册页替换第三方 captcha UI 为演示卡；邮箱文案区分 development/production。
- Admin 侧栏补齐治理和站点配置分组。
- 全部内部链接迁移 `Link`，页面不直接读 token；补 loading/empty/error/unauthorized。

**完成证据：**

- anonymous/user/moderator/admin 导航矩阵通过。
- 375px、768px、1280px 无横向溢出；提交错误能定位到对应分区。
- 登录后提交表单不整页刷新，发布结果能跳转对象公开 ID。

## Phase G: 文档、兼容窗口和发布门禁

**目标：** 在不改变路线的情况下，把实际完成情况同步回文档。

**范围：**

- 每阶段更新 `docs/implementation-gap-analysis.md` 的证据和状态。
- 仅修正旧文档中“已实现”但实际只是基础占位的描述，不删 Phase 12/13 目标。
- 为旧 Case API 标记兼容窗口、弃用时间和迁移说明。
- README 增加对象/事件、验证模式、导出包和开发/生产差异的验证方式。

**完成证据：**

- `go test ./...`、`go build ./cmd/server`、前端 lint/typecheck/build 通过。
- 核心接口 smoke、导入导出、captcha 网络隔离、SMTP fake server、角色流程都有可复现命令。
- 差距文档中未验证项仍保持 `部分实现` 或 `未实现`，不提前标完成。

## 依赖关系

```text
A → B → C → D → E → F → G
        ↘ D 的处罚检查必须先于默认发布
```

UI 可以在 A/B 阶段做只读原型，但不能在 D 完成前宣称“默认发布提交可用”。导出不能在 B 完成前开发为最终功能。

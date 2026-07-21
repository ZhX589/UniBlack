# UniBlack 实现与设计目标差距分析

> 基线：2026-07-20，代码基线 `1719cab`；工作区后续未包含功能实现提交。本文件记录当前真实实现与已确认设计目标之间的差距，不修改未来路线。

> 实施更新（分支 `feature/subject-event-governance`）：已实现并验证 `000006`–`000009` 兼容迁移、`UBS_<ULID>`、Account/Event、真实 LocalStorage、发布内嵌文本证据事务、Event 文件上传、受限 UTF-8 校验、默认发布、分级处罚列表/撤销/审计、demo captcha、开发固定验证码、归档导出、哈希命名空间校验、冲突预览与确认导入、动态 Auth/Settings Shell 与管理侧栏。仍未完成：发布请求内多文件二进制事务、处罚申诉用户自助流、旧 Case API 弃用窗口、完整 Phase 12 设计 token/Playwright 矩阵。

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
| 对象通用名规则 | 未实现 | `SubmissionService` 审核时硬编码 `DisplayName: "待补充"` | 不能满足空名自动取第一条账号用户名 |
| 公开对象 ID | 已实现 | `UBS_<ULID>`、唯一索引、归档命名和 public 查询已验证 | 历史对象保留 UUID 兼容回填值 |
| 账号字段 | 部分实现 | `Identifier` 有 platform/account_type/value/label | 没有 username/account_id/custom_attributes 的独立模型，也没有新规则的去重策略 |
| 默认发布 | 部分实现 | `/api/subjects/publish` 创建 active Subject、published Event 与内嵌文本证据索引，已 API smoke | 发布请求内多文件二进制事务仍待补；旧 Submission 审核兼容保留 |
| 事件证据 | 部分实现 | Event 文本/文件归档键、SHA-256、可见性检查已 smoke | 旧 Case 证据路径仍兼容；链接证据导入细节待加强 |
| 真实文件存储 | 部分实现 | LocalStorage 真实写盘；Event 文本/文件与归档 inclusion 已 smoke | MinIO/S3 adapter 待补 |
| 导出/导入包 | 部分实现 | ZIP、manifest v1、README、强制哈希、命名空间、冲突预览与确认导入已 smoke | 导入来源记录、异步大包任务和回滚 UI 待补 |
| 申诉结论 | 部分实现 | outcome/resolution 持久化，Event corrected/withdrawn 更新已实现 | malicious 提交到处罚创建的自动工作流、版本历史待补 |
| 分级处罚 | 部分实现 | sanctions 列表/创建/撤销/发布拦截/审计与管理页已 smoke | 用户自助处罚申诉待补 |
| 人机验证 | 部分实现 | 无第三方 URL；demo token 按 purpose/IP 或 JWT 用户绑定且单次使用已 smoke | appeal UI/API 与发送限流待补 |
| 邮箱开发模式 | 部分实现 | development 固定 `123456`、生产无 SMTP 失败已实现 | submission/appeal 发码入口与频率限制待补 |
| 邮箱生产模式 | 部分实现 | SMTPMailer 支持 SSL/认证；缺 host 失败 | purpose/频率边界待补 |
| 配置 OptionMap | 已实现基础 | `setting/options.go`、Bootstrap、schema/settings/values | 可补 demo mode 文案与证据上限配置 |
| 统一控制台 | 部分实现 | admin 侧栏含审核/用户/名单/处罚/归档/设置 | 品牌分组与更完整治理 IA 待补 |
| 前端动态化 | 部分实现 | Auth/Settings Provider、角色导航、管理守卫、Link 顶栏已实现并构建 | 完整 design token、Playwright 矩阵、页面级 API client 统一待补 |
| 测试 | 部分实现 | 后端 domain/export/service/storage/captcha 单测与 API smoke | 前端 unit/e2e 仍缺 |
| 部署可靠性 | 部分实现 | LocalStorage 已写盘；前端生产 build 通过 | Docker `npm ci`/rewrite 构建参数仍待核对 |

## 后端逐项差距

### 1. Subject 与 Account

当前 `Subject`：

- UUID 作为唯一 ID，没有 `public_id`。
- 关联的是 `[]Identifier`，字段只有 `platform`、`account_type`、`value`、`label`。
- 没有独立 `accounts` 表，不能同时表达 username、account_id 和有限 custom attributes。
- 现有唯一约束仍是旧 `identifiers` 设计，不能直接代表 Phase 13 的账号冲突策略。

目标需要新增可回滚 migration 和兼容 adapter。旧 `identifiers` 不能直接删除，原因是现有公开查询、subject 详情、提交和历史数据仍依赖它。

### 2. Case 到 Event

当前代码仍在以下位置使用 Case：

- `models.Case` 和 `cases` 表。
- `CaseRepository`、`CaseService`、`CaseHandler`。
- `Submission.CaseID`、`Evidence.CaseID`、`Appeal.CaseID`。
- `/api/cases`、`/api/v1/cases`、subject cases 路由。

这不是简单的改名任务。需要同时处理外键、公开 API、证据关联、申诉关联、历史数据和前端文案。Phase 13 已明确要求先新增结构、迁移读取路径、保留旧 API 兼容窗口，因此当前差距是“整体迁移未开始”，不是“漏改几个字符串”。

### 3. Submission 行为

当前 `CreateSubmission`：

- 接收 `subject_identifiers` 和 `reason`，没有 display name、账号 ID、事件时间、证据清单。
- 默认状态为 `pending`。
- 审核通过后才创建 Subject 和 Case。
- 审核逻辑硬编码 Subject 名称为“待补充”。

Phase 13 需要新增独立的 verified publish 入口，事务内创建对象、账号、事件、审计；旧 submission review 作为迁移期兼容路径保留。当前不能通过把 `pending` 改成 `published` 来完成，因为数据结构和验证前置条件也没有实现。

### 4. Evidence 与 Storage

当前 Evidence 已有 `file/link/text` 类型，但：

- 外键仍为 `case_id`。
- 文件 key 为 `evidence/<caseID>/<timestamp>.<ext>`，不是对象公开 ID 文件名。
- `LocalStorage.Upload` 没有写入 `basePath`，只返回 placeholder URL。
- 没有持久化明确 `storage_key` 和原始文件名字段。
- 文本证据仍通过普通 JSON 请求创建，没有转 UTF-8 `.txt` 文件。
- 没有 200 KiB 文本限制、manifest、哈希包校验和导入预览。

因此导出功能必须排在实际 Storage 和证据索引之后，不能直接从现有 URL 生成“看似完整”的 JSON 包。

### 5. Appeal 与 Sanction

当前 Appeal：

- 只能关联 Case。
- 只允许 `approved`/`closed` Case。
- 结论只有 `approved`/`rejected`。
- approved 处理只调用一次 Case 更新，没有明确修正、撤销、恶意提交和版本记录。

当前没有任何处罚模型。默认发布策略若先上线而不先实现处罚查询，会形成“发布容易、治理无法落地”的不完整闭环，因此处罚检查必须在发布入口之前完成。

### 6. Captcha 与邮箱

当前 captcha 明确存在第三方网络调用：

- 后端 `Turnstile`、`Recaptcha`、`HCaptcha` 实现 `postSiteverify`。
- 前端注册页动态加载三家第三方脚本。
- Provider 配置选择会改变运行时真实验证行为。

这与 Phase 13 的“仅演示 captcha、不接入真实服务”冲突，需要标记为旧实现，而不是标为已完成。未来路线不变，执行时应：保留配置 Catalog/API/UI 契约，替换运行 provider 为 demo provider，并添加“演示模式”状态说明。

当前邮件：

- `mailer.New` 在 SMTP host 为空时返回 `LogMailer`。
- AuthService 会生成并存储随机验证码，再通过 LogMailer 输出。
- SMTPMailer 已有普通 SMTP 和隐式 SSL 支持。

这意味着 SMTP 能力是部分实现，但环境隔离、固定开发码、生产缺失 SMTP 失败、purpose 区分和发送限流都未完成。

## 前端逐项差距

现有 `docs/frontend-gap-analysis.md` 仍然有效，但范围主要覆盖 Phase 12。与 Phase 13 的新增差距如下：

| 页面/能力 | 当前实现 | Phase 13 目标 |
| --- | --- | --- |
| `/submit` | 仅有 subject_identifiers/reason 基础表单，页面直接读 token | 对象、账号、事件、证据、验证、默认发布的单页分段表单 |
| 未登录提交 | 当前页面逻辑不具备设计中的灰态说明和分区禁用 | 说明页 + 灰态上传/发布区 + 登录引导 |
| 对象页面 | 使用 Subject UUID 和 cases 文案 | 使用公开 `UBS_<ULID>`、账号档案、事件时间线、证据可见性 |
| 案件详情 | `/cases/[id]` 和“案件”文案 | 事件详情，兼容旧链接但新文案和状态模型使用 Event |
| 注册 captcha | 动态加载第三方 provider | 内置“我不是自动程序”演示卡，不加载第三方脚本 |
| 注册邮箱 | 依赖后端现有开关和 LogMailer | development 固定 `123456`，production SMTP 必须配置 |
| 管理设置 | 基础/安全/登录三 tab | 站点与品牌、注册验证、SMTP、演示 captcha、治理和导入导出统一分组 |
| 管理治理 | 无处罚、导出、申诉结论页面 | 对象、事件、申诉、处罚、审计、归档统一入口 |

## 过时或需要修正文档

以下不是未来路线变更，而是现状描述需要校正：

1. `docs/configuration.md` 的原始注册链路仍描述真实第三方 captcha 可运行；Phase 13 目标章节已说明这是待实施变更，实施前不能把旧段落当作最终行为。
2. `docs/roadmap.md` Phase 8/9/11 的“完成”只代表历史功能页面或基础接口存在，不代表 Phase 12/13 的动态壳层、对象事件模型、演示验证和导出治理已经完成。
3. `README.md` 中的“支持证据管理”“申诉流程”“响应式前端”是能力方向描述，不应理解为 Phase 13 目标已验收。
4. `docs/frontend-modernization-plan.md` 和 `docs/compose/plans/2026-07-20-dynamic-frontend.md` 是 Phase 12 的设计/计划；它们不覆盖后端对象事件迁移，不可替代 Phase 13 执行计划。

## 不变的未来路线

以下目标保持原样，本文件不做削弱或替换：

- Phase 12：动态前端、Auth/Settings/API 架构、RBAC 导航、Tailwind 设计系统、响应式和构建验收。
- Phase 13：对象/账号/事件、`UBS_<ULID>`、混合存储、默认发布、JSON 包、申诉更正、分级处罚、演示 captcha、开发固定验证码、生产 SMTP、统一控制台。

## 差距结论

当前剩余的阻塞链为：

```text
发布请求仍不携带二进制文件证据
  → 文本证据可随发布事务创建
  → 文件证据需发布后独立上传
  → 完整“多文件一次提交即归档”体验尚不能验收
```

验证与体验链：

```text
Phase 12 动态 Shell 仅完成 Auth/Settings/角色导航基础
  → 设计 token、统一 API client、Playwright 矩阵未完成
  → 处罚申诉用户自助流与旧 Case API 弃用窗口未完成
```

已完成的领域迁移、demo 验证、处罚、文本/文件证据、归档导入与基础动态壳层可继续支撑后续工作。下一步优先：发布请求多文件事务、处罚申诉、Phase 12 剩余验收项。

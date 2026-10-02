# UniBlack 审计修复与验证进度

> 2026-10-02。本记录是实施进度，不改写设计提案或既有 Goal。工作基线 `f6632d9`；分支 `fix/audit-ledgers-and-domain-logic`；工作树已有未提交修改，本轮保留，不把它们冒认为本轮修复。未提交、未部署、未访问真实 .env/生产数据。

## 本轮实施

| 审计项 | 实施状态 | 实际范围与证据 |
| --- | --- | --- |
| A04 刷新丢身份 | 代码与单元回归通过；DB/入口集成待补 | [AuthService](<../../../backend/internal/service/auth.go>) 使用独立 refresh 验证端口，先验证凭证，再重载用户与角色，最后签发 token；[会话回归](<../../../backend/internal/service/auth_session_test.go>) 覆盖刷新身份与角色更新 |
| A03 停用/撤权后旧会话 | 部分修复，未关闭 | access/refresh 都读取当前账户，停用/删除/DB 不可用拒绝，撤权后即时使用新角色；密码重置撤销在第四批增加 auth_version；**refresh 一次性消费/token family 尚未实施** |
| A07 受限 Event 读取 | 目标通道代码与单元回归通过；DB/入口集成待补 | [统一策略](<../../../backend/internal/service/event_visibility.go>)；[Event handler](<../../../backend/internal/handler/event.go>) 详情与版本先授权；[Subject handler](<../../../backend/internal/handler/subject.go>) copy-filter 预载事件；[矩阵测试](<../../../backend/internal/handler/event_visibility_test.go>)。当前真正预载 Events 的是 Get/Update，其余筛选是防回归；未宣称全部读取通道安全认证 |
| A09 not-found 标识 | 标识已修；第四批分页回归通过 | [Subject service](<../../../backend/internal/service/subject.go>) 复用仓储 sentinel，避免同文案不同 error identity；未处理全部分页响应或全项目错误映射 |
| A12 注册标签未执行 | 代码与单元回归通过；DB/入口集成待补 | service 在验证码消费/持久化前验证非空用户名、纯邮箱地址和 8–72 byte 密码；[Auth handler](<../../../backend/internal/handler/auth.go>) 输入错误返回 400；[输入回归](<../../../backend/internal/service/auth_env_test.go>) |
| A13 附件覆盖/误删 | 代码与单元回归通过；DB/入口集成待补 | [Evidence service](<../../../backend/internal/service/evidence.go>) 给每次附件操作随机物理身份；[回归](<../../../backend/internal/service/evidence_attachment_test.go>) 含旧对象保全、失败只删本次对象、并发键和真实归档 preview 格式兼容 |
| A14 后续验证失败残留 | 代码与单元回归通过；DB/入口集成待补 | [Publish](<../../../backend/internal/service/event.go>) 在任何上传前校验全部 link/text/file；[7 组前置校验场景](<../../../backend/internal/service/event_publish_test.go>) |
| A18 迁移测试旧版本假定 | 测试代码已修，集成未执行 | [迁移集成测试](<../../../backend/internal/db/migrate_integration_test.go>) 从实际最新版本真实 down/up 新迁移，再准备 version 11 guard fixture，不以 Force 假装回滚 newer schema |
| JWT 契约补强 | 代码与单元回归通过；DB/入口集成待补 | [JWT provider](<../../../backend/internal/auth/jwt.go>) 固定 HS256、issuer 和必须存在 expiry；[负向测试](<../../../backend/internal/auth/jwt_test.go>) |

架构优化限于既有单体：用例端口 `AuthUserStore` / `RefreshIdentityValidator`、单一读取政策、证据展示编号与物理身份分离、上传前整批校验。没有新增 ORM 自动建表、微服务、OAuth、API Key、Case 功能或破坏性迁移。

## 第二批实施（同日自动继续轮）

| 审计项 | 状态 | 改动与边界 |
| --- | --- | --- |
| A01 Subject 越权修改 | service/handler 单测通过；DB/真实 JWT 路由待补 | [Subject service](<../../../backend/internal/service/subject.go>) 显式 actor/roles，非空 admin/moderator 才可改风险/状态、删除、增删 Identifier。Subject 是共享治理记录，投稿归属不授予其他贡献者资料的控制权；[授权矩阵](<../../../backend/internal/service/subject_authorization_test.go>) 覆盖拒绝先于 DB 和治理正向，handler 测试拒绝 JSON 伪造角色。新增 SubjectStore 注入接口，不扩大 legacy 功能 |
| A02 申诉退休越权 | service/handler 单测通过；PG 测试 SKIP | [申诉服务](<../../../backend/internal/service/appeal.go>) 只允许 owner 退休 pending 且未裁决的申诉，治理角色可软退休；[仓储](<../../../backend/internal/repository/appeal.go>) 行锁内重复归属/终局检查，退休与审计同事务。终局 owner 返回 409，其他用户 403；历史未删 |
| A05 恢复邮件 fail-closed | service 单测通过；PG 投递回滚 SKIP | [ForgotPassword](<../../../backend/internal/service/auth.go>) 非开发模式先检查 SMTP 并禁止实际 LogMailer 回退；发送/lookup 失败公共结果泛化，固定安全日志不含凭证/邮箱/错误原文；[恢复服务](<../../../backend/internal/service/password_reset.go>) 投递失败传播到事务回滚 |
| A06 重置一次性消费 | 代码已修；PG 并发与故障测试 SKIP | [重置仓储](<../../../backend/internal/repository/password_reset.go>) 事务内 FOR UPDATE + CAS，token 消费/密码更新均要求一行，否则回滚；[PG 回归](<../../../backend/internal/repository/password_reset_integration_test.go>) 覆盖并发单赢家、过期、投递失败及密码更新失败 |

新增测试均已在全量 Go 编译中通过编译；未配置 DATABASE_URL 的 PG 测试明确跳过，不据包 PASS 宣称并发与 SQL 事务已实测。环境没有可用 psql/postgres/pg_isready/docker 命令，未安装或连接生产设施。

邮件与数据库不是原子事务：当前最小实现跨投递持 DB 事务，发送成功后 commit 失败会出现已送达 token 无效；SMTP 旧实现缺完整 context/deadline，需下轮有界 I/O 修复。统一响应仅防内容枚举，不保证等时。

## 第三批实施（同日继续轮）

| 项 | 状态 | 证据/边界 |
| --- | --- | --- |
| SMTP 有界 I/O | 本机真实 SMTP 夹具 + race 通过 | [mailer](<../../../backend/internal/mailer/mailer.go>) 全链路默认 15 秒预算（可缩短），DialContext、连接 deadline、取消主动关闭套接字；SSL/STARTTLS 共享发送路径、最低 TLS1.2，拒绝邮件头 CRLF 注入。[测试](<../../../backend/internal/mailer/mailer_test.go>) 实测 greeting/TLS/DATA 阶段取消、deadline、正常完整投递及头注入；不连接真实 SMTP |
| A11 处罚申诉原子性 | service 回归通过；3 组 PG 测试 SKIP | [处罚仓储](<../../../backend/internal/repository/sanction.go>) 锁申诉、必要时锁处罚；终局更新、撤销及审计同事务；已撤销处罚不覆盖原原因。Create/Revoke 审计也同事务；[service](<../../../backend/internal/service/sanction.go>) 单命令委托不吞错；[PG 故障/并发测试](<../../../backend/internal/repository/sanction_integration_test.go>) 已编译待执行 |
| A17 发布/裁决快照与版本竞争 | 代码编译通过；2 组 PG 测试 SKIP | [发布仓储](<../../../backend/internal/repository/event.go>) 初始版本和 Event/审计同事务，去掉 [service](<../../../backend/internal/service/event.go>) post-commit 吞版本错；[版本仓储](<../../../backend/internal/repository/event_version.go>) stable Event 行锁串行化 MAX+1。正式 corrected/withdrawn 裁决在同事务记录 reviewer snapshot；无 actor 的内部 UpdateStatus 不冒用 submitter。[PG 测试](<../../../backend/internal/repository/event_version_integration_test.go>) 包含版本写失败整笔发布回滚、并发顺序、裁决 actor；历史快照缺口不自动回填 |
| A22/A23 异步可信状态 | RTL 回归 + 类型/lint/build 通过 | search/lookup 共用请求 generation+真正 AbortController；管理队列列表与计数独立状态；事件详情与历史独立请求、失败明确报错/可重试而非“暂无”。[6 个 deferred promise 回归](<../../../frontend/test/request-lifecycle.test.tsx>) 验证乱序、取消、旧 finally、局部失败和恢复 |

最新门禁：`go test ./...`、`go vet ./...`、后端 build、`go test -race ./internal/mailer ./internal/service ./internal/handler ./internal/storage` 全部 PASS；前端 11 文件 **100/100**、typecheck、lint、build PASS。`go test -v` 确认新增 3 个处罚和 2 个版本 PG 用例因 DATABASE_URL 缺失 SKIP，不当成 SQL 事务/并发运行证据。没有启动替代服务器或 Compose。

SMTP 原无界读写风险已补；仍需真实 TLS/SMTP 环境验收。邮件与 SQL 无法 exactly-once，发送跨 DB 事务仍持连接；成功投递后的 commit 故障可能使已送达 token 无效。A11 处罚申诉提交的独立审计已在事务内落实；handler 状态错误映射已在第六批补齐（见下）。A17 历史快照/导入和 legacy review 旁路需继续盘点。

## 第四批实施（同日继续轮）

- A03：新增 [000014 up](<../../../backend/internal/migrations/000014_auth_version.up.sql>) / [down](<../../../backend/internal/migrations/000014_auth_version.down.sql>) 与 users.auth_version。JWT access/refresh 都携带版本，登录和刷新从当前账号签发，认证重载并比较。自助及管理员重置密码在密码 SQL 更新中同时递增版本，旧 access/refresh 被拒；[service 回归](<../../../backend/internal/service/auth_revocation_test.go>) 验证旧凭证失效、新登录及刷新正常。兼容无版本旧凭证为 0，只能用于未重置账号；新 token_use 区分 access/refresh，即使签名密钥相同也不能混用。AuthProvider 扩展接口仍保留，不硬绑外部 OAuth。
- 配套防竞态：[用户仓储](<../../../backend/internal/repository/user.go>) profile 更新仅写 username/email，不再 Save 整行，避免陈旧 profile 覆盖新密码/auth_version。[PG 回归](<../../../backend/internal/repository/auth_version_integration_test.go>) 已编译待隔离 DB 执行；重置 PG 用例增加 version 一次递增/失败回滚断言。**迁移尚未实际 up/down，混合版本部署须先迁移并协调切换；回滚旧认证代码会失去撤销保证。**refresh family/消费仍未实施，不标 A03 全关闭。
- A09：共享 [NormalizePagination](<../../../backend/internal/domain/pagination.go>)，8 类 handler 与相关 service 用相同有效页码/大小，保留非法 size→20 而非 clamp100。[64 组合 handler 回归](<../../../backend/internal/handler/pagination_test.go>) 使用 GORM DryRun 校验 response 与实际 LIMIT/OFFSET，纯函数测试含幂等，全部通过。极大页码乘法溢出未扩改。
- A24/A25： [settings 页面](<../../../frontend/app/admin/settings/page.tsx>) 分离 secret draft 与展示 map，成功清空，不在通用 JSON 存原值或mask；schema/row/legacy secret 过滤，空值/mask保持“不修改”，不新增清除或 OAuth。安全 checkbox 用中文 label/description 关联；[8 个 RTL 回归](<../../../frontend/test/admin-settings.test.tsx>) 验证DOM无原值、失败重试及可访问名称。A25 发布标签部分仍待补，不标整项完成。

最新验证：后端全量 test/vet/build PASS；auth/service/handler/mailer race PASS；profile 修复后再次全量 test/vet/build PASS。前端 **10 文件89/89**、typecheck/lint/build PASS。所有 DATABASE_URL 依赖测试仍 SKIP，迁移与 SQL 并发尚无实际证据。

## 第五批实施（同日继续轮）

- A21：[API client](<../../../frontend/lib/api.ts>) 按 token+session generation 绑定每次请求，匿名/公开/旧登录401不能退出新会话；按 owner single-flight refresh，GET/HEAD 最多重试一次，写请求不自动重放。[Provider](<../../../frontend/app/providers.tsx>) 过期access/仍有refresh的bootstrap可续期；晚到刷新成功/失败都验证owner后才写storage或logout。[11项延迟回归](<../../../frontend/test/auth-session.test.tsx>) 已通过。没有迁移Cookie或页面读取token。
- A10：[legacy review](<../../../backend/internal/repository/appeal.go>) 仅Case-only approved/rejected；行锁事务验证pending/未退休/无既有裁决，Event-target 409要求canonical resolve。移除ID-only Case Save，完整锁读Case仅验证存在，不擅自映射approved为corrected/withdrawn或改Case状态；终局与audit同事务。service/handler单测通过，[SQL并发/审计故障用例](<../../../backend/internal/repository/appeal_review_integration_test.go>) 已编译仍SKIP。旧approved含义欠明确按兼容审批保留，不宣称会执行Case纠正后果。
- A15：[Archive Import](<../../../backend/internal/export/archive.go>) 每次操作随机分配独立物理对象namespace；输赢实例即使共享DB/存储也不写同一key，补偿仅删自身对象；Subject查找非not-found错误fail closed，不把DB失败当不存在。cleanup脱离已取消ctx并限制10秒；不是存储级create-if-absent/持久清理任务，随机碰撞/崩溃残留仍为限制。跨实例SQL+对象完整性尚未实测，不能标整项验收。
- A16：v1 manifest新增可选severity/occurred range/correction note，保留旧包severity默认1；文件/文本标题说明回导恢复。entry上限与发布20MiB对齐，compressed64MiB/expanded128MiB/256 entries预算不解除，导出也检查相同预算（超预算明确失败）。[往返/边界回归](<../../../backend/internal/export/archive_roundtrip_test.go>) 包含17MiB/20MiB合法、20MiB+1拒绝、旧包兼容、业务字段恢复；单测/race通过，真实DB完整Build→Import尚待补。对象IDs/投稿人映射/全部审计历史不构成完整备份承诺。

最新后端全量test/vet/build PASS，export/service/handler race PASS；前端 **11文件100/100**、typecheck/lint/build PASS。仍未配置隔离数据库，SQL集成/000014迁移up/down未实测；浏览器完整E2E和Compose未运行。所有原有dirty修改保留，未提交或部署。

## 第六批实施（同日继续轮）

- A11：[处罚 handler](<../../../backend/internal/handler/sanction.go>) 此前新增的 `sanctionHTTPError` 助手是死代码——Create/Revoke/Appeal/ResolveAppeal 仍用 `err.Error()` 直接回显，泄漏内部错误。本轮将其接线到全部四个变更端点，typed sentinel（NotFound/Forbidden/Revoked/AlreadyAppealed/AppealNotFound/AppealAlreadyResolved/InvalidRequest）映射到 400/403/404/409；未匹配错误统一脱敏为 500 `sanction operation failed`，不再反射 `%w` 包裹的 SQL/校验内部。List/ListMine 的 500 也脱敏。[9 组 handler 回归](<../../../backend/internal/handler/sanction_http_test.go>) 覆盖每个端点的 typed 映射 + 内部泄漏断言（响应体不含 secret SQL/deadlock 等内部串）。service 层接口化 + `%w` 传播既有 4 个单测仍通过；repo 集成测试仍 SKIP。
- A25：[发布表单](<../../../frontend/app/submit/page.tsx>) 多账号/多事件/证据动态行补齐稳定唯一中文 label：账号（平台/用户名/账号 ID）、事件（标题/详情/严重级别）、文本/链接标题/链接 URL/文件证据、邮箱验证码均带序号 `htmlFor`/`id` 配对，保留原有 `aria-label="平台 N"` 兼容 e2e。静态校验确认 12 组 htmlFor/id 全部一致匹配、跨 map 作用域前缀不冲突。前端 100/100、typecheck/lint/build PASS。

## 第七批实施（同日继续轮）

- 证据删除授权绕过（新增发现，高危）：[DeleteEvidence](<../../../backend/internal/service/evidence.go>) 原先只验证 JWT 登录、**不检查归属**——认证用户获知有效证据 ID 后可删除他人证据；没有证据证明 UUID 可预测枚举。本轮签名改为 `DeleteEvidence(ctx, id, actorID, roles)`，删除前强制授权：空 actor 永远拒绝，仅原始投稿人（`UploadedBy == actorID`）或治理角色（admin/moderator）可删；拒绝时不触碰存储或元数据。[handler](<../../../backend/internal/handler/evidence.go>) 传入 `user_id`/`roles` 并映射 `ErrEvidenceAccessDenied→403`，500 也脱敏。新增 [HasGovernanceRole](<../../../backend/internal/service/event_visibility.go>)，目前仅 Evidence 写授权使用，未宣称 Subject/Event 已统一。[8 组授权回归](<../../../backend/internal/service/evidence_authorization_test.go>) 覆盖 owner/governance 正向、空 actor/陌生人/空 uploader 拒绝且断言无副作用。既有删除测试更新签名后仍通过。
- 审计范围记录：handler 层另有 60+ 处 `err.Error()` 直回（主要 500 InternalServerError 反射 DB 内部）。全量重构超本轮可验证范围且有破坏既有断言风险，列为后续轮次专项。本轮聚焦授权绕过（高危、可单测验证）而非批量脱敏。

## 文件与状态标记校正

- [全栈台账](<../../implementation-gap-analysis.md>) 顶部明确早期总表是历史快照，Playwright/Compose 已有历史交付，legacy UI 是有意兼容保留。
- [前端台账](<../../frontend-gap-analysis.md>) 不再把有失败的 E2E 标为完整通过；校正已存在的角色/密码管理和名单 CSV。
- [体验历史报告](<product-experience-deepening.md>) 保留已交付事实，撤销“失败与实现无关”的无证据豁免；串行通过不证明跨 worker mock 污染。
- [只读审计](<2026-10-02-readonly-audit-and-validation.md>) 是先前阶段原始记录，保留其“当时未实施”边界；本文件记录后续实施，不篡改原始发现。

## 本次真实验证

环境：Go `go1.27.1-X:nodwarf5 linux/amd64`，Node `v26.10.0`，npm `12.2.0`。

| 命令/检查 | 结果 | 说明 |
| --- | --- | --- |
| `gofmt` / `git diff --check` | PASS | 语法格式/空白检查，不等于 Go 编译通过 |
| `npm run test:run` | PASS | 最新 11 文件、100/100；早期 75/75 为历史结果 |
| `npm run typecheck` | PASS | 无类型错误 |
| `npm run lint` | PASS | 无 warnings/errors |
| `npm run build` | PASS | 18 静态页面生成；login/search 有既有 client-rendering deopt 警告 |
| `go test ./internal/service ./internal/auth ./internal/db ./cmd/server` | FAIL (setup) | 首轮 sandbox 拒绝工作区外默认 Go 缓存；授权原命令重试后依赖 `proxy.golang.org` 下载 TCP 超时，未进入编译/测试阶段，不能将此标成修复回归失败或测试通过 |
| `GOPROXY=https://goproxy.cn GOSUMDB=sum.golang.google.cn go test ./...` | PASS | 默认代理 TCP 超时，国内镜像与校验服务可达；授权默认缓存后全包通过，未修改依赖版本。最新 SubjectStore 变更后再次全量通过 |
| `go vet ./...` / `go build -o /tmp/uniblack-audit ./cmd/server` | PASS | 后端静态检查和服务可编译 |
| `go test -race ./internal/service ./internal/handler ./internal/storage` | PASS | 三包竞态检查通过，不等于 PostgreSQL 并发证明 |
| PostgreSQL 迁移/治理集成 | SKIP（明确记录） | `go test -v ./internal/db ./internal/repository ./internal/integration -run 'Test(RunMigrations\|PasswordReset\|AppealRetirement)' -count=1`：迁移、1 组申诉退休、4 组重置均因 DATABASE_URL 未配置跳过；没有运行生产库 |
| 完整 Playwright / Compose / 真实入口 | NOT RUN | 不用历史 smoke 替代本轮验证；没有启动/清理生产容器 |

本轮未修改依赖版本。下列普通门禁已通过；配置隔离数据库后仍需重跑获得 SQL 集成证据：

```bash
# backend/，集成测试 DATABASE_URL 只能指向隔离测试库
 go test ./...
 go vet ./...
 go build -o /tmp/uniblack-audit ./cmd/server
 go test -race ./internal/service ./internal/handler ./internal/storage
```

## 快速收尾交接

按用户要求停止扩展修复范围，交付当前工作树而非宣称所有审计项完成。最终补证据空对象/空 actor 携带治理角色的拒绝测试，正向用例断言对象与元数据实际均被删除；修正“UUID 可枚举”“所有治理判定已收敛”“repo 事务单测已实测”等过度表述。验证命令使用 `&&` 保留失败退出码，不再通过 tail/head 或末尾 echo 掩盖失败。

已知剩余风险：证据删除仍是先对象后元数据，非跨存储原子操作；repository/service 的 EvidenceNotFound sentinel 尚不统一；裸 `/uploads` 与 S3 对象直链需独立授权审计。发布请求预算虽有代码修改，缺专门边界回归；发布标签仅静态检查，没有新增 RTL/浏览器验收。不得将本次收尾当作可直接生产发布的安全认证。

## 未关闭问题与继续开发顺序

1. 补隔离 PostgreSQL/真实 JWT 路由/真实 SMTP 证据；A06/A11/A17 并发和事务仍未实测。A03 密码重置撤销已实现/单测通过，迁移与 SQL 集成待补，refresh family/一次性消费仍待开发。
2. A10兼容Case审批语义仍欠明确，Event旁路已封；A11 handler 错误映射已补齐（第六批），service 委托与 handler 单测通过，repo 审计事务 PG 测试尚未实测；A17 历史快照覆盖盘点。
3. A15/A16需要真实DB+存储多实例/完整往返验收；持久清理队列和备份语义仍待设计。
4. A25 发布标签已补齐（第六批静态校验 + 门禁通过），仍需浏览器/E2E 动态验收。A21–A24 和安全开关名称已有本轮单元回归，仍需浏览器验收。

限制：随机键不是存储级 create-if-absent；历史已损坏对象不自动恢复；cleanup 取消/失败仍可留下孤立对象。当前状态公开的 Event 历史仍沿用原语义，逐 snapshot 披露、TOCTOU、legacy/导出/文件直链尚待复核。密码重置版本撤销已加入，但迁移尚未实际验证/部署，refresh 重放仍未解决，不能把 A03 标成完成。上述限制必须在运行验证与后续修复中继续追踪。

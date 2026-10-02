# UniBlack 只读审计：错误标记、验证规格与执行规划

> 2026-10-02，Asia/Shanghai。状态：静态审计完成；运行验证未执行；没有修复实施。
> 配套：[可信治理演进设计](<../specs/2026-10-02-trust-and-governance-evolution-design.md>)。

## 1. 范围与证据边界

本轮读取项目约束、最新体验报告、进度台账，并并行审查认证/治理、前端、文件归档/迁移/部署；仅新建 Markdown 文档，未修改任何已有代码、配置、测试或 Goal。没有提交、切换分支、启动服务或访问生产数据；未读取真实 .env。

**本会话没有 shell/命令执行工具，未执行 Go、npm、Playwright、Compose 测试。**现有台账的历史通过记录只是历史信息，不是本轮验证结果。也没有获取 git SHA/工作树状态，证据是本次读取的工作区文件，而非声称 main 的固定 commit。维护者运行时必须附 SHA、dirty 状态、依赖版本和环境。

标记规则：
- **C（静态确认）**：代码路径或配置事实已定位；不代表实际生产遭遇或浏览器复现。
- **R（条件风险）**：代码窗口存在，但最终影响取决于并发、环境、产品语义或外部设施，需要测试。
- **D（设计建议）**：未来增强，不算现有错误。
- 优先级：P0 为建议立即安全复核/修复；P1 为近期发布门槛；P2 为后续质量与一致性。不是 CVSS 或漏洞披露结论。

尊重 [产品决策](<../../product-decisions.md#L5-L17>)：不新增 Case 功能，不在 Sunset checklist 完成前删表，不擅自引入 OAuth/API Key 等。已交付三段发布、可信档案与治理队列，见 [体验报告](<product-experience-deepening.md#L14-L68>)，不将其重新列为缺失。

## 2. 已有保障，避免误判

- [发布仓储](<../../../backend/internal/repository/event.go#L58-L87>) 已把 Subject/Accounts/Events/Evidence 元数据与发布审计置于事务。
- [正式 Event 申诉裁决](<../../../backend/internal/repository/appeal.go#L119-L172>) 已有事务、Appeal 行锁、pending 检查、状态/处罚后果和审计；缺口是旁路与其他流程不一致，不是“全项目没有原子裁决”。
- [申诉退休](<../../../backend/internal/repository/appeal.go#L175-L193>) 是软退休并原子写审计；问题是授权，不是宣称永久删除历史。
- [LocalStorage](<../../../backend/internal/storage/storage.go#L37-L68>) 已拒绝路径逃逸；不据此猜测任意文件读写漏洞。
- [现有治理集成测试](<../../../backend/internal/integration/governance_test.go#L91-L148>) 已覆盖正式裁决后果与回滚。测试存在不等于本轮执行通过。

## 3. 问题登记与复现规格

所有测试均要求隔离测试账号、数据库 schema、对象目录/bucket；不得在生产尝试越权、故障注入或破坏性删除。下列“预期”是建议安全/业务契约，不是已执行结果。

### A01｜P0 / C：普通用户可修改、删除任意 Subject

**证据**：[路由](<../../../backend/cmd/server/main.go#L210-L236>) 对 Subject PUT/DELETE、Identifier 写入仅要求登录；[修改 handler](<../../../backend/internal/handler/subject.go#L93-L121>) 不传 actor；[服务](<../../../backend/internal/service/subject.go#L123-L151>) 无归属/角色检查，可直接改 status/risk_level。

**触发/影响**：用户 B 知道用户 A 的对象 ID，可以改名称、风险等级或状态；删除是否成功取决于外键，但修改越权路径明确存在。Identifier 操作同样缺少对象级政策。前端不显示按钮不能弥补服务器授权。

**验证 V01**：A 创建对象，普通 B PUT 该 ID；预期 403/策略规定的 404，DB 不变；覆盖 DELETE、添加/删除 Identifier，及合法管理者正向操作。建议 service 层显式 actor + ManageSubjectPolicy，并审计变更。

### A02｜P0 / C：其他用户可软退休他人申诉

**证据**：[路由](<../../../backend/cmd/server/main.go#L265-L270>)；[handler](<../../../backend/internal/handler/appeal.go#L133-L143>)；[service](<../../../backend/internal/service/appeal.go#L168-L170>)；[仓储](<../../../backend/internal/repository/appeal.go#L177-L193>) 只按 ID 与 deleted_at 匹配。

**影响**：B 可隐藏 A 的 pending 申诉，阻断正式裁决的活跃读取；已有原子审计不解决越权。

**V02**：A 创建申诉，B 删除，预期拒绝且仍在治理队列；所有者是否允许撤回必须先决定，不能默认把所有删除改成 admin-only。

### A03｜P0 / C：停用、撤权、重置密码不撤销旧会话

**证据**：[AuthService](<../../../backend/internal/service/auth.go#L273-L279>) 直接委托 provider；[JWT](<../../../backend/internal/auth/jwt.go#L111-L140>) 只恢复 claims；[角色中间件](<../../../backend/internal/middleware/auth.go#L30-L65>) 依据 token；[用户仓储](<../../../backend/internal/repository/user.go#L149-L160>) 与 [密码更新](<../../../backend/internal/repository/password_reset.go#L82-L87>) 不改变会话版本。

**影响**：禁用后旧 access 仍有效，撤 admin 后旧 token 仍有管理权限；refresh 没有服务端撤销/消费。登录检查 active 不等于已有会话被撤销。

**V03**：登录后分别停用、撤角色、重置密码，重用旧 access/refresh；预期失效或按当前授权处理。候选：session/token family + auth_version + 当前 active/身份重载，定义绝对期限及允许的撤销延迟。

### A04｜P1 / C：刷新丢失用户名、邮箱、角色

**证据**：[refresh claims](<../../../backend/internal/auth/jwt.go#L87-L96>) 只含 user ID；[刷新](<../../../backend/internal/auth/jwt.go#L117-L125>) 从缺失字段生成新 access；[名单](<../../../backend/internal/middleware/access.go#L24-L30>) 对空邮箱/用户名跳过检查。

**影响**：管理员续期后变 403；仅被邮箱/用户名封禁的用户可能绕过相关检查。用户 ID 与 IP 检查仍存在，不是全部认证失效。

**V04**：管理员刷新前后 profile/管理访问一致；邮箱封禁但 IP 未封的用户刷新后仍被拒绝。必须重载最新用户，不要简单把陈旧角色放进 refresh。

### A05｜P0 / C（环境条件）：无 SMTP 的生产忘记密码泄露恢复令牌到日志

**证据**：[ForgotPassword](<../../../backend/internal/service/auth.go#L457-L478>) 先建有效 token，best-effort 发信并吞错；[mailer](<../../../backend/internal/mailer/mailer.go#L35-L55>) host 空则 LogMailer 打印完整正文。这与 [生产 fail-closed 约束](<../../../AGENTS.md#L50>) 不一致。

**影响**：满足 production 且 host 空时，日志读者可取得恢复凭证；用户却收不到邮件。开发日志便利不构成生产豁免。

**V05**：隔离 production/空 SMTP，触发测试账号忘记密码，捕获并脱敏输出；预期不产生可用凭证、不记录秘密，并保持防账号枚举策略。实际 SMTP 故障也要覆盖，不能忽略邮件失败。

### A06｜P1 / R：重置令牌并发消费不保证一次性

**证据**：[重置仓储](<../../../backend/internal/repository/password_reset.go#L65-L87>) 事务外读 used_at，事务内无条件更新，没有锁或 `used_at IS NULL` CAS。

**影响**：两个请求均读到未使用后可能双成功，最后提交者决定密码。单调用事务与顺序拒绝重复都不是并发保证。

**V06**：读取屏障让两请求同时使用同 token/不同密码；预期仅一次成功，失败者不改密码，令牌消费与密码变更同事务。

### A07｜P0 / C：受限事件的历史与父级对象读策略不一致

**证据**：[Event Get](<../../../backend/internal/handler/event.go#L135-L154>) 检查状态/归属；[GetVersions](<../../../backend/internal/handler/event.go#L157-L165>) 无可见性检查，直接 [ListByEvent](<../../../backend/internal/service/event.go#L370-L374>)；[登录 Subject Get](<../../../backend/internal/handler/subject.go#L39-L51>) 返回预载 [全部 Events](<../../../backend/internal/repository/subject.go#L37-L55>)。

**影响**：非投稿者普通用户可通过 versions/Subject 响应读到 withdrawn 等事件正文，而直接 Event Get 被拒绝；历史中还可能保留更正前敏感材料。

**V07**：A 的 withdrawn Event，普通 B 读直接详情、versions、父 Subject，预期披露政策一致；公开更正摘要与完整旧正文需分别定义。

### A08｜P1 / C + 产品语义 R：公开对象状态过滤不一致

**证据**：[lookup/search](<../../../backend/internal/repository/subject.go#L82-L99>)、[Search](<../../../backend/internal/repository/subject.go#L201-L214>) 要求 active；[公开列表](<../../../backend/internal/handler/public.go#L203-L223>) 接受任意 status，底层 [List](<../../../backend/internal/repository/subject.go#L110-L132>) 无默认限制；[公开详情](<../../../backend/internal/handler/public.go#L92-L129>) 不检查 Subject status。

**影响**：如非 active 意味着隐藏/撤回，公开 list/detail 是旁路；如只是公开档案状态，则需明确解释为什么 lookup 不返回。Accounts/custom_attributes 被整体序列化，还需字段披露白名单。

**V08**：同一个非 active 对象覆盖 search/lookup/list/detail/statistics，先确认状态语义再断言。不要未定义语义就声称所有非 active 都是秘密。

### A09｜P1 / C：未找到错误跨层不统一，404 可变 500

**证据**：[repository sentinel](<../../../backend/internal/repository/subject.go#L14-L18>) 与 [service sentinel](<../../../backend/internal/service/subject.go#L12-L15>) 分别 `errors.New`；[service Get](<../../../backend/internal/service/subject.go#L100-L108>) 不映射；[public handler](<../../../backend/internal/handler/public.go#L96-L101>) 用相等判断。

**影响**：合法格式但不存在的 Subject/lookup 可返回 500；不是 SQL 注入。分页另外在 service 归一化，handler 仍回显未归一化 page/page_size，见 [分页 service](<../../../backend/internal/service/subject.go#L110-L119>) 与 [响应](<../../../backend/internal/handler/public.go#L226-L230>)。

**V09**：不存在 UUID 返回 404；缺省/非法分页回显实际采用值。统一领域错误与 `errors.Is`/typed error，不把 DB 内部错误直接暴露给用户。

### A10｜P1 / C：旧 review 可绕过正式裁决状态机

**证据**：[并存路由](<../../../backend/cmd/server/main.go#L291-L294>)；[旧 service](<../../../backend/internal/service/appeal.go#L128-L149>) 不要求 pending，批准 Case 时保存仅 ID 的 Case；[旧仓储](<../../../backend/internal/repository/appeal.go#L91-L103>) 无条件覆盖；[Case Save](<../../../backend/internal/repository/case.go#L83-L85>)。

**影响**：有审核权限者可把已裁决/退休申诉再改状态，与 outcome/处罚脱节；Case 批准可能零值覆盖或违反约束，实际数据库结果待测，前面的申诉写入已提交。

**V10**：正式 resolve → 旧 review 改回 pending → 再 resolve，不允许重复后果；完整 Case 旧批准检查所有字段/回滚。修复方向是兼容入口委托唯一命令，不加新 legacy 功能。

### A11｜P1 / C：处罚申诉批准与撤销、审计分离

**证据**：[SanctionService](<../../../backend/internal/service/sanction.go#L112-L135>) 先批准，再撤销，审计错误吞掉；[仓储](<../../../backend/internal/repository/sanction.go#L89-L102>) 已裁决后不能正常重试。处罚创建/撤销也见 [service](<../../../backend/internal/service/sanction.go#L41-L59>)。

**影响**：撤销失败后显示 approved 但处罚仍有效；审计可缺失。不要把已有正式 Event 裁决事务误认为覆盖处罚申诉。

**V11**：注入 Revoke 与 Audit 写失败，预期整笔裁决回滚或可显式恢复；并发/重复请求仅一后果。

### A12｜P1 / C：注册校验标签没有执行

**证据**：[请求标签](<../../../backend/internal/service/auth.go#L106-L112>)；[handler](<../../../backend/internal/handler/auth.go#L23-L32>) 只有 Bind；[注册 service](<../../../backend/internal/service/auth.go#L199-L233>) 无最短密码/邮箱格式检查。

**V12**：注册开放且验证条件通过时，空/1/7 字节密码、空用户名、非法邮箱应 400 且零新增；开启邮箱验证也必须拒绝短密码。密码重置已有长度检查，不能扩大为所有密码流程缺失。

### A13｜P0 / C：客户端证据编号可覆盖原件，失败清理可删旧证据

**证据**：[handler](<../../../backend/internal/handler/evidence.go#L104-L129>) 接收/default 编号；[service](<../../../backend/internal/service/evidence.go#L133-L162>) 按编号建键，上传后写 DB，失败删键；[Local rename](<../../../backend/internal/storage/storage.go#L81-L98>) 覆盖；[S3 Put](<../../../backend/internal/storage/s3.go#L59-L61>)。

**影响**：同 Subject 相同编号得到同键，原 Evidence 行 hash 与字节分离；第二次 DB 失败补偿还可能删第一条原件。新 Event 所有者可能覆盖同 Subject 其他 Event 材料，事件归属检查不保护共享物理键。

**V13**：连续两次默认编号文本，比较两个 DB 行键/哈希/内容；第二次 metadata 故障后第一份仍可读。建议物理键由服务端 Evidence UUID/operation ID 分配，展示编号与键解耦，采用不可变写。

### A14｜P1 / C：后续文本验证失败遗漏补偿

**证据**：[cleanup](<../../../backend/internal/service/event.go#L236-L239>)；[文本循环](<../../../backend/internal/service/event.go#L255-L266>) 第一份上传后第二份 Validate 失败直接返回；文件分支 [已有 cleanup](<../../../backend/internal/service/event.go#L281-L284>)。

**V14**：合法第一文本 + 非法第二文本，DB 零新增且对象零残留。先全量验证再 I/O；请求取消/清理失败采用独立限时补偿与重试，不默默吞错。

### A15｜P1 / R：导入互斥只在单进程，跨实例可能破坏成功导入

**证据**：[mutex](<../../../backend/internal/export/archive.go#L55-L61>)、[检查与上传](<../../../backend/internal/export/archive.go#L334-L379>)、[失败清理](<../../../backend/internal/export/archive.go#L435-L437>)。

**影响**：共享 DB/存储的两实例同时通过检查、写同键；失败方可覆盖/删除成功方对象。DB 唯一约束不保护对象；Open 的非 NotFound 错误也不能当不存在。

**V15**：两个 service/同步屏障，同 public ID 但不同字节并发导入；唯一成功结果的每个 hash/对象必须完整；Open 网络/权限错误应 fail closed。单实例不是声称当前已部署多实例，属于扩展前门槛。

### A16｜P1 / C：归档往返不闭合，合法文件可能无法回导

**证据**：[发布上限 20 MiB](<../../../backend/internal/service/event.go#L90>) 与 [导入 entry 16 MiB](<../../../backend/internal/export/archive.go#L23-L27>)；[Manifest](<../../../backend/internal/export/archive.go#L39-L52>) 不含 Severity/发生时间/纠正说明；[回导构造](<../../../backend/internal/export/archive.go#L393-L426>) 固定 Severity=1，文件/文本标题说明不完整恢复。

**影响**：自身合法数据的导出包可能被拒绝；能导入也丢业务语义。先明确迁移包/备份包/展示快照，不能不加区分宣称完整备份。

**V16**：空隔离库 round-trip，17 MiB 文件、Severity=5、发生时间、corrected note、证据标题/说明，逐字段比较；多条目/总大小边界也测。不能简单无限抬高解压预算，需统一契约与资源上限。

### A17｜P1 / C：发布成功可缺版本，MAX+1 有竞争窗口

**证据**：[发布后吞错](<../../../backend/internal/service/event.go#L311-L319>)；[版本仓储](<../../../backend/internal/repository/event_version.go#L21-L41>) 无锁 MAX+1，ChangedBy=SubmittedBy。

**V17**：注入版本写失败，发布不能返回“完整历史成功”却缺 snapshot；并发 RecordVersion 唯一顺序；裁决操作人必须准确。状态、版本、审计同事务；现有 append-only 命名不等于不可篡改或完整事件溯源。

### A18｜P1 / C（未执行）：迁移测试假定最新版本仍为 11

**证据**：[测试](<../../../backend/internal/db/migrate_integration_test.go#L43-L64>) 全部 Up 后一次 Down 期望 11 的拒绝信息；目录已有 [13 down](<../../../backend/internal/migrations/000013_event_versions.down.sql#L1>)，首先回滚的是 13。

**推断**：在完整当前目录和可用测试 DB 下，该断言应失败；本轮未运行，不能标成实际 CI 失败。

**V18**：临时 schema 运行迁移集成测试，记录当前版本；专门 fixture 明确迁至 10/11 测保护，12/13 单独 up/down；不要用 Force 伪装实际 schema 回滚。

### A19｜P1 / C + 环境 R：仓库生产入口默认无 TLS，上传预算未配置

**证据**：[Compose](<../../../docker-compose.prod.yml#L74-L84>) 映射 80/443；[Nginx](<../../../nginx/nginx.conf#L31-L46>) 仅启用 HTTP，[HTTPS 块](<../../../nginx/nginx.conf#L73-L82>) 注释，配置无 client_max_body_size。

**影响**：若没有外部 TLS 终止，443 映射不构成 HTTPS；Nginx 默认体积可能在 handler 前拒绝合法大文件。README 已要求配置 HTTPS，因此是部署默认/验收门槛，不是断言所有生产实例明文上线。

**V19**：维护者核对真实入口、effective config、可信代理；隔离入口测 2/20 MiB 文件和归档边界（含 multipart 开销）。HTTP/TLS、限流、应用/代理限制一起验收。

### A20｜P2 / C + 外部策略 R：Release 标签提示与实际标签不一致

**证据**：[metadata](<../../../.github/workflows/release.yml#L33-L42>) 后端 semver 去 v；[发布提示](<../../../.github/workflows/release.yml#L85-L86>) 用 github.ref_name；[触发/依赖](<../../../.github/workflows/release.yml#L3-L9>) 未在同一流程绑定测试门禁。

**V20**：v1.2.3 的 metadata 输出/GHCR tags 对比发布提示；核对外部 ruleset 和对应 SHA 的 CI，不能仅凭 workflow 缺测试就宣称仓库没有保护。建议输出实际 tag+digest，发布与测试绑定同 SHA。

### A21｜P1 / C：前端存 refresh 但不续期，401 状态跨会话

**证据**：[Provider](<../../../frontend/app/providers.tsx#L75-L85>) 过期直接清 tokens；[登录](<../../../frontend/app/providers.tsx#L112-L117>) 只保存；[API](<../../../frontend/lib/api.ts#L20-L30>) 全局 unauthorizedNotified，仅 configure 重置，[401](<../../../frontend/lib/api.ts#L89-L103>) 不区分 auth/session。

**影响**：有效 refresh 无法保住长表单；首次 401 → 重新登录 → 再 401 不再通知退出；旧会话晚到 401 可能退出新会话；匿名请求 401 也消耗全局标记。竞态具体 UI 待浏览器测。

**V21**：expired access/valid refresh；A 401→登录 B→B 401 应按会话处理；A 慢请求在 B 登录后失败不得影响 B。后端 A04 契约先修/验证，再设计 session generation + single-flight，写请求不得无条件重放。

### A22｜P2 / C（乱序影响待测）：查询与治理资源状态可陈旧/误导

**证据**：[search](<../../../frontend/app/search/page.tsx#L28-L67>) 无取消/request ID；[申诉队列](<../../../frontend/app/admin/appeals/page.tsx#L89-L120>) 列表失败与计数成功共用 error；[effects](<../../../frontend/app/admin/appeals/page.tsx#L128-L162>) 创建的 abort controller 不是 refresh 请求的 controller。

**V22**：A 查询慢、B 快，A 最后完成，仍只显示 B；列表 500/计数 200 不应抹错误变成无申诉；卸载/切筛选取消实际请求。保留旧数据时必须显式“刷新失败”，不是装作新数据。

### A23｜P2 / C：申诉历史异常被描述成“暂无记录”

**证据**：[事件页](<../../../frontend/app/events/[id]/page.tsx#L42-L63>) catch 清空 appeals；[渲染](<../../../frontend/app/events/[id]/page.tsx#L239-L244>) 空数组显示无记录；加载开始未清旧 detail error，[错误优先](<../../../frontend/app/events/[id]/page.tsx#L103-L104>)。

**V23**：历史 403/500/取消分别显示相应状态而非无记录；旧取消不能清新数据；详情先失败后成功能恢复。治理数据“未知”和“没有”必须区分。

### A24｜P2 / C：保存密钥后通用配置 DOM 展示明文

**证据**：[settings save](<../../../frontend/app/admin/settings/page.tsx#L83-L93>) 把提交原值写 map；[raw 展示](<../../../frontend/app/admin/settings/page.tsx#L269-L271>) JSON.stringify；密钥调用见 [SMTP](<../../../frontend/app/admin/settings/page.tsx#L181-L187>)、[Captcha](<../../../frontend/app/admin/settings/page.tsx#L214-L220>)、[OAuth 预留](<../../../frontend/app/admin/settings/page.tsx#L256-L262>)。

**影响**：屏幕共享、截图、页面导出/DOM 暴露面扩大；这是 admin 本页披露，不是已证明的远程越权读取。

**V24**：保存测试密钥，通用 pre/展示 map 不含原值；专用输入与展示模型分离，用已配置/未配置状态，明确不修改/替换/清除语义。

### A25｜P2 / C：安全开关缺可访问名称，发布标签不稳定

**证据**：[checkbox](<../../../frontend/app/admin/settings/page.tsx#L326-L334>) label 内无名称，与外部文字无关联；[发布账号](<../../../frontend/app/submit/page.tsx#L292-L307>)、[事实输入](<../../../frontend/app/submit/page.tsx#L346-L360>) 主要用 placeholder。

**V25**：按 role+中文 name 定位安全 checkbox；多账号/事件输入具有唯一、可理解名称；axe + 键盘/屏幕阅读器验收。placeholder 有些环境会回退为名称，不武断认定所有输入绝对无名称；现有响应式测试不等于 WCAG 系统验收。

## 4. 尚待核验，不升级为确定漏洞

1. **ULID 编码互操作**：[自写编码](<../../../backend/internal/domain/id.go#L39-L59>) 从首个实际 bit 分组、末尾补零；标准 ULID 的 130-bit 文本需要前置填充；[validator](<../../../backend/internal/domain/id.go#L96-L111>) 也只校验形状，[测试](<../../../backend/internal/domain/id_test.go#L8-L19>) 不解码时间。建议用已知向量与成熟实现做 parse/time round-trip。本轮官方规范正文抓取不完整，不把互操作影响列为已运行确认；现有 ID 仍可作为 opaque key，不能直接重写历史 public ID。
2. **静态原件路径**：[静态上传路由](<../../../backend/cmd/server/main.go#L333-L334>) 不经过 Event 可见性；若 LocalStorage 原件落该目录且 URL 可达，撤回后可能直接读。生产 MinIO 私有性、Nginx 路由、签名有效期均需实测，不宣称所有生产文件公开。
3. **聚合请求预算**：[multipart](<../../../backend/internal/handler/event.go#L66-L90>) 每个 field 的第一文件 ReadAll，只有单文件限制。需检查总 body/数量/并发预算，压力测试资源峰值，不能只抬 Nginx body limit。
4. **依赖安全**：[frontend package](<../../../frontend/package.json#L15-L36>) 锁定较旧框架版本；需使用当前官方 advisory/锁文件与实际功能验证可利用性。本轮未跑 audit/govulncheck，不给具体 CVE 或“安全”结论。
5. **E2E 并发归因**：[旧报告](<product-experience-deepening.md#L68-L72>) 将部分失败归因 page.route 跨 worker；串行通过不能单独证明原因。需要 trace、共享 backend/mock fixture 盘点、随机化和重复执行，不接受“与实现无关”作为永久豁免。
6. **旧 Subject 创建吞 Identifier 错误**：[service](<../../../backend/internal/service/subject.go#L74-L97>) 保存 Subject 后继续吞添加错误，可能返回部分数据；兼容窗口内需验证和告知，不新扩展 legacy 模型。

## 5. 可执行验证计划（本轮全部 NOT RUN）

### 5.1 先建立独立环境

维护者记录 git SHA、分支、dirty 状态、Go/Node/npm/PostgreSQL 版本、实际迁移版本；测试数据库必须独立于生产。确认集成测试要求的变量与清理行为，见 [迁移测试开头](<../../../backend/internal/db/migrate_integration_test.go#L1-L38>)。禁止把生产 DATABASE_URL 直接传测试。

运行目录与命令建议（只是计划，不是本轮执行记录）：

```bash
# 在 backend/，指向隔离测试数据库；先读测试环境约定
 go test ./...
 go vet ./...
 go build -o /tmp/uniblack-audit ./cmd/server

# 在 frontend/；依赖已准备，使用隔离后端/浏览器
 npm run test:run
 npm run typecheck
 npm run lint
 npm run build
 npm run test:e2e -- --workers=1
 npm run test:e2e
```

命令会产生构建/缓存/报告，不属于本轮只读操作。依赖扫描如需网络应独立批准，避免修复式安装。现有 npm scripts 已由 [配置](<../../../frontend/package.json#L5-L13>) 核对；具体测试 DB 环境按文件读取确认。

### 5.2 建议验收矩阵

| 门禁 | 用例 | 必须保留证据 |
| --- | --- | --- |
| G1 权限 | V01/V02/V07/V08 | 角色×归属×状态×读取通道；HTTP 与 DB 前后快照 |
| G2 会话与恢复 | V03–V06/V12/V21 | access/refresh 生命周期；日志仅脱敏；并发唯一消费 |
| G3 治理原子性 | V10/V11/V17 | DB 失败注入、重复/并发裁决、状态/版本/审计一致 |
| G4 原件保全 | V13–V16 | 对象 hash、取消/失败清理、两实例竞争、归档逐字段比较 |
| G5 迁移部署 | V18–V20 | 迁移版本、入口 TLS/413、镜像 digest 与同 SHA 测试 |
| G6 前端可信呈现 | V22–V25/V09 | 乱序 trace、错误/空态、秘密值 DOM 断言、可访问名称 |

每条结果记录 PASS/FAIL/SKIP、命令、时间、环境、SHA、artifact、负责人。Skip 需说明依赖与补测日期；并发测试用同步屏障控制 interleaving，不用 sleep 充当确定性证明。

若后续获准 Compose smoke，必须使用隔离项目/数据，记录实际 URL；结束执行既有清理要求并核查容器/网络。不要启动替代服务就声称真实生产 URL 已验证，也不要删除长期生产数据卷。

## 6. 修复与规划顺序（未实施）

1. **先复核高风险边界**：A01/A02/A03/A05/A07/A13；暂不扩功能。必要时由维护者限制暴露入口，但本轮不修改配置。
2. **恢复发布门禁**：A18（迁移 fixture）、A19（真实入口）、A16（归档契约）、A04（刷新身份）。历史 smoke 不覆盖这些场景。
3. **收束治理命令**：A10/A11/A17，保留正式 Event 事务优势，消除 legacy 旁路，定义幂等/版本/操作人。
4. **证据任务可靠性**：A14/A15，服务器不可变键、预验证、任务命名空间、失败重试与对账，参考配套设计。
5. **可信 UI**：A21–A25，session 状态机、独立资源错误、未知/空区分、秘密编辑模型、Field label。
6. **兼容退役**：从现在盘点流量与双轨差异；遵守 [原有 checklist](<../../product-decisions.md#L15-L42>) 的公告、双读、备份和证据，不因 2026-12-31 到期自动删表。

建议拆为测试规格 docs/ PR → 单问题 fix/ PR → 架构 feature/ PR，每个有回归场景和退出条件；不直接提交 main。配套设计覆盖 20 条候选架构路径及分阶段路线，本报告的错误不依赖先完成大重构。

## 7. 本轮交付清单

- 本报告：25 个问题登记、6 项待核验风险、25 组验证规格与 6 类门禁。
- [演进设计](<../specs/2026-10-02-trust-and-governance-evolution-design.md>)：目标/非目标、锁定约束、20 个候选方向、推荐架构、Mermaid 草图、风险与阶段路线、已抓取参考来源。
- 所有发现状态保持“未修复/未运行验证”；这不是安全认证或完整覆盖证明。后续人员应逐项复核，允许依据复现结果降低严重程度或关闭误报，并保留理由。

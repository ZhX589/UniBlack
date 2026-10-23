# 配置与数据库连接审查跟进（2026-10-23）

分支：`fix/audit-ledgers-and-domain-logic`。前批审计修改已通过提交 `2cbd067` 推送到同名远程修复分支；未合并 main、未部署。

## 本批改动

- 生产配置拒绝开发默认 MinIO access/secret（任一为 minioadmin 即拒绝），但允许在明确配置非默认凭证时保留默认 endpoint/bucket 名称。补单独默认密钥、空白凭证与 Load 缺少凭证的回归。
- 修复环境测试污染：测试恢复原环境是否存在及原值，不再无条件 Unsetenv 丢失调用方环境。
- 放弃对完整数据库连接串的自动密码修复：直接交给 GORM/pgx 解析，不使用 LastIndex 猜 userinfo，也不使用 QueryEscape 改写密码。原始密码必须在组装 URL 时正确编码；不能猜测整个 DSN 中哪些保留字符属于密码。
- 直接使用 pgx.ParseConfig 的离线测试覆盖空格、加号、百分号、保留字符、编码用户名、IPv6、带 @ 的 query、冒号密码及含 URL 的 keyword 密码。该测试验证实际驱动解析语义，不等价于真实数据库连接成功。

## 验证

`go test ./internal/config ./internal/db -count=1`、`go test ./...`、`go vet ./...`、`go build -o /tmp/uniblack-audit ./cmd/server` 全部退出码 0；`git diff --check` 通过。依赖 DATABASE_URL 的集成测试未配置时仍跳过，本批没有前端修改。

## 证据读取边界后续

配置修复已在 `c8016d4` 推送。本轮统一 Evidence service/repository 的 NotFound sentinel，handler 使用 errors.Is 支持包裹错误，使缺失证据详情/删除正确返回404；内部读取错误脱敏500。Event证据缺少授权服务时返回503，不再跳过授权直接披露；空结果安全返回404。新增6场景读取矩阵及1个删除404回归，legacy非Event读取兼容行为保留。裸uploads/S3直链和Case范围读取政策未据此声称解决。全量Go test/vet/build及handler/service race退出码均为0；数据库依赖测试仍未配置时跳过。

## Multipart 发布映射边界

前批证据边界已在 `b3262fc` 推送。本批拒绝多个payload值、重复文件field、同一文件被多条元数据重复引用、显式映射遗漏文件，以及自动映射中无效/负数索引，不再静默忽略错误字段。新增9场景映射回归（含2种合法协议）及65文件超量拒绝用例。首次数量夹具使用控制字符导致MIME解析失败，已改为合法file_0_N字段后重跑；定向测试、全量test/vet/build与handler race全部退出码0。本批不宣称已测试72MiB请求体/64MiB总附件/20MiB单文件所有临界点。

## 服务层发布预算

映射修复已在 `b5e391d` 推送。本轮将64文件与64MiB总文件预算下沉到service，Publish在访问验证/仓储等依赖前拒绝超量请求；handler共用同一常量，防止未来入口偏离。新增64/65文件、64MiB/64MiB+1总量、20MiB/20MiB+1单文件及nil依赖前拒绝的回归。Content标记为json:"-"，现有JSON不能直接携带二进制；服务层防线主要覆盖直接调用及未来入口，不把它描述为已存在的JSON二进制攻击。全量Go test/vet/build、service/handler race均退出码0。72MiB HTTP原始请求体临界点尚待单独验收。

## 原始HTTP请求预算

服务预算已在 `0dc6dfa` 推送。本轮提前拒绝Content-Length超过72MiB的请求，并在普通JSON绑定后通过MaxBytesReader有界读取剩余请求体，防止首对象后的尾随数据绕过总预算。已知长度和未知长度（模拟chunked）均测试72MiB恰好合法与72MiB+1拒绝；超量发布返回413，且不调用service。使用流式空白生成器，不分配整个72MiB缓冲。定向临界测试、全量Go test/vet/build与handler race均退出码0。此证据不涵盖代理超时/慢速请求策略，也不声称完整multipart与真实网络组合验收。

## 边界

生产 Compose 直接将原始 POSTGRES_PASSWORD 插入 URL 的历史问题仍待单独修复；本批没有修改 Compose。更早的健康检查、部署清理文档等未提交修改保留在工作树，不随本批配置修复混入。

隔离 PostgreSQL 迁移/并发集成与完整浏览器验收仍未获得证据，不能将本批单测当作 SQL 事务验收。

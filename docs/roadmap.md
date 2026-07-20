# Development Roadmap

> 本文档描述 UniBlack 的开发路线、每个阶段的目标以及验收标准。
>
> 开发遵循"先稳定基础，再开发业务，最后完善体验"的原则，避免频繁推翻已有设计。

---

# Conventions & Decisions (locked)

以下是已敲定的约定，后续阶段必须遵循，勿擅自更改。

## 命名统一

- 项目名称固定为 **UniBlack**（"云黑系统"为产品描述，非代码命名）。
- 代码、仓库、文档、Docker 镜像、环境变量前缀一律使用 `UniBlack` / `uniblack`，不得出现 `CloudBan` 等旧称。
- Go module 路径建议：`github.com/ZhX589/UniBlack`（后端子模块 `github.com/ZhX589/UniBlack/backend`）。

## 技术工具选型（已定，勿随意替换）

| 用途 | 选定方案 | 说明 |
| --- | --- | --- |
| 后端框架 | Go + Echo | README 已定 |
| ORM | GORM | 仅用于读写，**不用于建表** |
| 数据库迁移 | **golang-migrate** | 使用 `migrate` CLI，SQL 迁移文件，支持 up/down、可重复执行 |
| 数据库 | PostgreSQL (>=14) | |
| 对象存储 | MinIO（S3 兼容） | 本地开发用 MinIO，生产可换 S3，代码走 S3 SDK |
| 前端 | React + Next.js (App Router) | |
| 样式 | Tailwind CSS + shadcn/ui | |
| 认证 | JWT (access) + Refresh Token | OAuth2/OIDC 仅预留接口，见下 |
| 编排 | Docker Compose | `docker compose up` 一键起 backend + frontend + postgres + minio |
| CI | GitHub Actions | lint / build / test / migrate 校验 |

## 认证扩展（AuthProvider 抽象，向前兼容）

Phase 2 实现 JWT 登录，但必须预留 OAuth2/OIDC，避免后期返工：

- 定义 `AuthProvider` 接口（如 `Verify(ctx, credentials) -> (SubjectIdentity, error)`），JWT 与未来 OAuth 均为其实现。
- 用户身份与登录方式解耦：内部统一为 `User`，外部来源（`local` / `oauth:github` / `oauth:discord` 等）记录在 `User.auth_provider` + `User.external_id`。
- 新增登录方式 = 新增一个 `AuthProvider` 实现 + 配置，不改核心登录流程。

## 代码分层与目录结构（已定）

### 后端 `backend/`

```
backend/
  cmd/server/            # 程序入口 main.go，组装依赖、启动 Echo
  internal/
    config/              # 环境变量加载（.env / .env.example）
    db/                  # 数据库连接、migrate 执行入口
    migrations/          # golang-migrate SQL 文件 (*.up.sql / *.down.sql)
    models/              # GORM 模型（Subject, Identifier, Case, ...）
    repository/          # 数据访问层，封装 GORM，不暴露 ORM 给上层
    service/             # 业务逻辑层
    handler/             # Echo HTTP 层（路由、请求校验、响应）
    middleware/          # JWT 鉴权、RBAC、限流
    auth/                # AuthProvider 接口及各实现（jwt, oauth 预留）
    storage/             # 对象存储抽象（S3/MinIO）
    api/                 # OpenAPI 文档生成配置
  pkg/                   # 可对外复用的公共库（如 uniblack 公共类型）
  go.mod
  Dockerfile
```

分层规则：
- `handler -> service -> repository -> db/models`，单向依赖，禁止跨层反向调用。
- `models` 仅定义结构体与表映射；建表由 `migrations/` 负责。
- 外部依赖（DB、存储、AuthProvider）通过接口注入，便于测试替换。

### 前端 `frontend/`（Next.js App Router）

```
frontend/
  app/                   # 路由与页面（首页、查询、Case、举报、登录、管理后台）
  components/ui/         # shadcn/ui 生成组件
  components/            # 业务组件
  lib/                   # API client、hooks、工具
  styles/                # Tailwind 入口
  .env.example           # NEXT_PUBLIC_API_BASE 等
```

## 环境配置约定

- `.env` 被 gitignore；提交 `.env.example` 作为模板。
- 后端必需的 env（初版）：`DATABASE_URL`、`JWT_SECRET`、`REFRESH_SECRET`、`MINIO_*`、`PORT`。
- 前端必需的 env：`NEXT_PUBLIC_API_BASE`（指向 backend）。
- 所有 env 必须有默认值或通过 `.env.example` 明示，CI 与 `docker compose up` 可直接跑通。

---

# Phase 0 - Project Bootstrap

## Why

在开发任何业务之前，需要确保所有开发者拥有一致的开发环境，并且项目能够被快速部署和运行。

稳定的项目结构可以避免后期频繁调整目录、构建流程和部署方式。

## Goal

按上文 **Conventions & Decisions** 落地整个项目初始化，包括：

- 创建 Git 仓库（已完成）
- 按 `backend/` 与 `frontend/` 目录结构初始化 Go (Echo) 后端与 Next.js 前端
- 配置 PostgreSQL（>=14）
- 引入 **golang-migrate**，建好 `backend/internal/migrations/` 骨架
- 配置 MinIO（S3 兼容）用于证据存储
- 配置 Docker Compose 编排 backend + frontend + postgres + minio（用于最终部署）
- 提交 `.env.example`（后端 + 前端），确保所有 env 有默认值
- 配置 GitHub Actions：lint / build / test / migrate 校验
- 配置代码格式化（gofmt/golangci-lint）与前端（prettier/eslint）
- 建立统一目录结构（见 Conventions）

## 开发策略：本地优先

**日常开发使用本地环境验证，Docker 仅用于最终部署打包。**

### 本地开发流程

1. **后端开发**
   ```bash
   cd backend
   go run cmd/server/main.go
   ```

2. **前端开发**
   ```bash
   cd frontend
   npm install
   npm run dev
   ```

3. **数据库**
   - 本地安装 PostgreSQL，或使用 `docker compose up postgres -d` 只启动数据库
   - 运行迁移：`cd backend && go run cmd/server/main.go`（自动执行迁移）

### Docker 用途

Docker Compose 仅用于：
- 生产环境部署
- CI/CD 流水线
- 一键启动完整环境（需要时）

## Verification

应满足以下条件：

- 后端 `go run cmd/server/main.go` 可以启动并监听端口
- 前端 `npm run dev` 可以启动并访问 http://localhost:3000
- 数据库迁移可以正常执行
- GitHub Actions 可以成功运行
- Docker Compose 可以构建镜像（验证镜像可用性）

---

# Phase 1 - Database Design

## Why

数据库决定了整个项目的数据结构。

API、权限、前端几乎都会依赖数据库设计，因此数据库应尽可能在开发初期稳定下来，而不是边开发边修改。

## Goal

完成所有核心实体设计，包括：

- User
- Role
- Permission
- Organization（预留）
- Subject
- Identifier
- Case
- Evidence
- Submission
- Appeal
- Audit Log

建立完整的实体关系图（ER Diagram）。

编写数据库 Migration（使用 **golang-migrate**，SQL 文件置于 `backend/internal/migrations/`），而不是依赖 GORM 自动建表。

## Verification

应满足以下条件：

- 所有表均能够通过 `migrate up` 创建
- `migrate up` 可重复执行（幂等），`migrate down` 可正常回滚
- 所有实体关系经过 Review
- 不存在明显的数据冗余

---

# Phase 2 - Authentication & Authorization

## Why

几乎所有业务都会涉及身份验证。

权限系统应尽早建立，避免后续接口重新设计。

## Goal

实现：

- 定义 `AuthProvider` 接口（见 Conventions），先落地 `jwt` 实现
- JWT 登录（access + refresh），通过 `auth` 包注入
- 用户注册 / 登录 / Token 刷新
- RBAC 权限控制（基于 `Role` / `Permission`）
- 管理员账户初始化
- `User` 表预留 `auth_provider` + `external_id`，为 OAuth/OIDC 留好扩展位（接口与数据层就位，具体 provider 实现留到 Phase 10）

## Verification

应满足以下条件：

- 用户能够成功登录
- Token 过期后能够刷新
- 不同角色访问接口时权限正确
- 未登录用户无法访问受保护接口
- 权限变更立即生效
- 新增登录方式只需新增 `AuthProvider` 实现，不动核心流程

---

# Phase 3 - Core Domain Model

## Why

本项目真正的核心不是"案件"，而是"对象（Subject）"。

所有案件、举报、申诉都会围绕 Subject 展开。

## Goal

实现：

- Subject
- Identifier
- Subject 查询
- Identifier 管理

支持多个 Identifier（社交账号）：

**国内平台**：
- QQ
- 微信 (wechat)
- B站 (bilibili)
- 抖音 (douyin)
- 快手 (kuaishou)
- 微博 (weibo)

**国际平台**：
- X (Twitter)
- Telegram
- Discord
- Steam
- Minecraft

**通用**：
- 手机号 (phone)
- 邮箱 (email)
- 自定义 (custom)

每个 Identifier 包含：
- platform: 平台类型
- account_type: 账号类型（username/nickname/id/phone/other）
- value: 账号值
- label: 自定义标签（platform=custom 时使用）

未来允许扩展新的平台和账号类型。

## Verification

应满足以下条件：

- 一个 Subject 可以拥有多个 Identifier
- Identifier 不允许重复
- 查询能够正确定位 Subject
- API 返回的数据结构稳定

---

# Phase 4 - Case Management

## Why

Case 是整个系统最重要的业务对象。

只有完成 Case 管理，系统才真正具备"云黑"能力。

## Goal

实现：

- 创建案件
- 编辑案件
- 删除案件
- 状态管理
- 审核流程
- 案件历史
- 操作记录

Case 应与 Subject 建立关联。

## Verification

应满足以下条件：

- 管理员能够完整管理案件
- 修改历史可追踪
- 删除操作受到权限控制
- 所有重要操作写入 Audit Log

---

# Phase 5 - Evidence System

## Why

案件必须具备可验证的依据。

证据应独立管理，而不是直接存储在案件中。

## Goal

实现：

- 图片上传
- 文件上传
- 外部链接
- SHA256 校验
- S3 / MinIO 存储

数据库仅保存元数据。

## Verification

应满足以下条件：

- 文件能够成功上传
- 文件能够正确删除
- 数据库与对象存储保持一致
- 重复上传能够正确处理

---

# Phase 6 - Submission Workflow

## Why

社区成员需要能够主动提交举报。

所有举报应经过审核，而不是直接公开。

## Goal

实现：

- 举报提交
- 草稿保存
- 审核
- 驳回
- 补充材料
- 提交记录

## Verification

应满足以下条件：

- 普通用户可以提交举报
- 管理员能够审核举报
- 举报状态能够正确流转
- 所有审核操作均可追踪

---

# Phase 7 - Appeal Workflow

## Why

公开的社区治理系统必须允许申诉。

申诉能够提高系统可信度，并减少误判造成的影响。

## Goal

实现：

- 发起申诉
- 上传补充材料
- 管理员处理申诉
- 保留历史记录

## Verification

应满足以下条件：

- 每个案件均可关联多个申诉
- 审核过程完整保留
- Case 状态能够同步更新

---

# Phase 8 - Public API

## Why

开放 API 可以方便机器人、网站和第三方工具接入。

API 应建立在稳定的数据模型之上。

## Goal

开放：

- 查询 API
- Case API
- Subject API
- Statistics API

提供完整 OpenAPI 文档。

## Verification

应满足以下条件：

- API 文档自动生成
- API 返回格式统一
- 分页正常
- Rate Limit 生效
- API Key 能正确控制权限

---

# Phase 9 - Frontend

## Why

当前端开始开发时，后端接口已经基本稳定，可以减少重复修改。

## Goal

完成：

- 首页
- 查询页面
- Case 页面
- 举报页面
- 登录页面
- 管理后台
- 设置页面

统一设计语言。

支持深色模式。

## Verification

应满足以下条件：

- 所有页面均可正常访问
- API 调用正常
- 响应式布局正常
- 错误提示清晰

---

# Phase 10 - Production Ready

## Why

完成核心功能后，应关注稳定性、安全性和部署体验。

## Goal

完成：

- Docker Image
- Release Workflow
- 自动备份
- Webhook
- 邮件通知
- OAuth 登录
- 性能优化
- 安全加固
- 文档完善

## Verification

应满足以下条件：

- 可一键部署
- CI/CD 正常
- 文档完整
- API 稳定
- 可以发布 v1.0.0

---

# Phase 11 - Admin Console & Enhanced Registration

## Why

云黑名单系统需要支持自定义配置，而不是硬编码。同时注册流程需要增加安全验证（人机验证、邮箱验证），防止恶意注册。

管理控制台让管理员可以配置系统行为，而无需修改代码或环境变量。

## Goal

### 1. 注册页面增强
- 邮箱验证码验证（可配置开关）
- 人机验证（reCAPTCHA / hCaptcha / Cloudflare Turnstile，可配置）
- 注册协议展示

### 2. 管理控制台（System Settings）

**基础配置**：
- 项目名称（自定义显示名称）
- 项目描述
- 主题色（主色调、Logo）
- 联系邮箱

**安全配置**：
- 邮件服务（SMTP 配置）
- 邮箱验证开关
- 人机验证（provider + site key + secret key）
- API 限速配置（公开接口、认证接口分别配置）

**登录配置**：
- OAuth 第三方登录配置（GitHub、Discord 等）
- 注册开关（允许/禁止新用户注册）

### 3. 用户管理
- 用户列表（搜索、分页、筛选）
- 用户详情查看
- 禁用/启用用户
- 角色分配（admin / moderator / user）
- 重置密码

### 4. 名单管理
- 白名单（IP / 用户名，跳过限速）
- 黑名单（IP / 邮箱，禁止注册/访问）
- 批量导入/导出

### 5. 初始配置
- **生产环境**：首次启动时要求配置 admin 账户密码
- **开发环境**：默认 admin 密码 `admin123`

## 数据库设计

### system_settings 表
| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| key | VARCHAR(100) | 配置键（唯一） |
| value | JSONB | 配置值（支持复杂对象） |
| description | TEXT | 配置说明 |
| updated_by | UUID | 最后修改者 |
| updated_at | TIMESTAMP | 最后修改时间 |

**默认配置键（扁平 key，值为 JSON 标量）**：
- `site.name` / `site.description` / `site.theme_color` / `site.logo_url` / `site.contact_email`
- `security.email_verification` - 邮箱验证开关
- `security.smtp_host` / `smtp_port` / `smtp_username` / `smtp_password` / `smtp_from` - SMTP
- `security.captcha_enabled` / `captcha_provider` / `captcha_site_key` / `captcha_secret_key` - 人机验证（turnstile|recaptcha|hcaptcha|none）
- `security.rate_limit_public` / `security.rate_limit_auth` - 限速
- `auth.registration_enabled` - 注册开关
- `auth.oauth_github_*` - OAuth 预留
- `system.initialized` - 是否完成首次初始化

**验证码存储**：表 `verification_codes`（email + purpose + code + expires_at）

### access_lists 表（白名单/黑名单）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| type | VARCHAR(20) | 类型（whitelist/blacklist） |
| target | VARCHAR(50) | 目标类型（ip/email/username） |
| value | VARCHAR(255) | 值 |
| reason | TEXT | 原因 |
| created_by | UUID | 创建者 |
| created_at | TIMESTAMP | 创建时间 |

## Verification

应满足以下条件：

- 管理控制台可正常访问和配置
- 配置变更实时生效（无需重启）
- 注册页面可配置人机验证和邮箱验证
- 生产环境首次启动要求设置 admin 密码
- 开发环境使用默认密码 admin123
- 用户管理功能正常
- 白名单/黑名单功能正常
- 前端 `/subjects/:id`、`/cases/:id` 可打开详情（公开案件仅 approved/closed）
- 注册页不在 settings 加载完成前误显示「注册已关闭」

## Status

- **已实现（2026-07）**：控制台 + 注册增强 + NewAPI 风格 OptionMap + 可插拔 captcha/mailer + 详情页

详见 `docs/configuration.md`。

---

# Phase 11.1 - UX / 可配置安全补丁

## Why

Phase 11 落地后发现：详情路由缺失导致 404；注册页 settings 未加载时闪现「已关闭」；邮箱/人机验证需「配置文件 + 接口 + 控制台」完整闭环（参考 NewAPI Option）。

## Goal

- 补齐 Subject / Case 前端详情页
- 注册页 loading 态与配置驱动的 captcha 组件
- `internal/setting`：Catalog + env 默认值 + 内存 OptionMap + DB 覆盖
- `internal/captcha`、`internal/mailer` 可插拔实现
- `verification_codes` 迁移与发送/校验接口
- 控制台 / admin API：`schema` + 脱敏 secrets + SMTP/captcha 全项
- `.env.example` 与 `docs/configuration.md` 文档

## Verification

- 列表链到 `/subjects/:id` 可打开
- 公开案件链到 `/cases/:id`；pending 案件返回明确未公开提示
- 注册页先显示「加载注册配置」
- `GET /api/admin/settings` 返回 schema + settings
- 开启邮箱验证后发送验证码可写入 DB（SMTP 未配时走 LogMailer）
- 开启 captcha 且配置 secret 后校验走对应 Provider
- 环境变量可覆盖默认项，控制台保存后无需重启

---

# Long-term Goals

未来版本计划包括：

- 多组织（Multi-Tenant）
- 插件系统
- Bot（Discord / Telegram / QQ）
- 联邦同步
- GraphQL API（可选）
- 全文搜索
- 国际化（i18n）

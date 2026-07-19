# Development Roadmap

> 本文档描述 Community CloudBan 的开发路线、每个阶段的目标以及验收标准。
>
> 开发遵循"先稳定基础，再开发业务，最后完善体验"的原则，避免频繁推翻已有设计。

---

# Phase 0 - Project Bootstrap

## Why

在开发任何业务之前，需要确保所有开发者拥有一致的开发环境，并且项目能够被快速部署和运行。

稳定的项目结构可以避免后期频繁调整目录、构建流程和部署方式。

## Goal

完成整个项目的初始化，包括：

- 创建 Git 仓库
- 初始化 Go (Echo) 后端
- 初始化 Next.js 前端
- 配置 PostgreSQL
- 配置 Docker Compose
- 配置环境变量
- 配置 GitHub Actions
- 配置代码格式化和 Lint
- 建立统一目录结构

最终应能够使用一条命令启动整个开发环境。

## Verification

应满足以下条件：

- `docker compose up` 可以成功启动所有服务
- 前端能够访问
- 后端能够启动
- PostgreSQL 可以正常连接
- GitHub Actions 可以成功运行
- 项目 README 中的 Quick Start 可以被其他开发者成功复现

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

编写数据库 Migration，而不是依赖 ORM 自动建表。

## Verification

应满足以下条件：

- 所有表均能够通过 Migration 创建
- Migration 可重复执行
- 数据库能够正常回滚
- 所有实体关系经过 Review
- 不存在明显的数据冗余

---

# Phase 2 - Authentication & Authorization

## Why

几乎所有业务都会涉及身份验证。

权限系统应尽早建立，避免后续接口重新设计。

## Goal

实现：

- JWT 登录
- Refresh Token
- 用户注册
- 用户登录
- Token 刷新
- RBAC 权限控制
- 管理员账户

预留 OAuth / OIDC 扩展能力。

## Verification

应满足以下条件：

- 用户能够成功登录
- Token 过期后能够刷新
- 不同角色访问接口时权限正确
- 未登录用户无法访问受保护接口
- 权限变更立即生效

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

支持多个 Identifier：

- QQ
- Discord
- Telegram
- Minecraft UUID
- Steam
- Email

未来允许扩展新的 Identifier 类型。

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

# Long-term Goals

未来版本计划包括：

- 多组织（Multi-Tenant）
- 插件系统
- Bot（Discord / Telegram / QQ）
- 联邦同步
- GraphQL API（可选）
- 全文搜索
- 国际化（i18n）

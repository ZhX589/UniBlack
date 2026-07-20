# UniBlack: 一个可复用的通用云黑系统

[![License](https://shields.io/badge/license-MIT-green)](https://github.com/ZhX589/UniBlack)

UniBlack 是一个可复用的通用云黑系统，支持在线提交&查询、通用查询API、申诉、管理员审核和追责。

> 其中“云黑名单”指的是一个统计曾骗人或做了其他在这个圈子内不被接受的事情的人及其相关社交账号的名单。
> 这仅仅是一个某社区在某时间，对某账号作出了某处理，并公开处理依据、证据、申诉的渠道。一切信息都来源于用户提交，并通过申诉和追责保证可信度、避免诽谤。

---

## 特性

UniBlack 支持：
- ✅ 在线提交云黑信息
- ✅ 支持QQ、微信、B站、抖音、X、Telegram、Discord、Steam、手机号等多平台
- ✅ 通用查询API（公开接口）
- ✅ 申诉&审核流程
- ✅ 完善的用户系统（RBAC权限控制）
- ✅ 证据管理（图片、文件、链接）
- ✅ 审计日志
- ✅ 响应式前端

## 快速上手

### 前置要求

- [Go](https://go.dev/dl/) (>= 1.22)
- [Node.js](https://nodejs.org/) (>= 18)
- [PostgreSQL](https://www.postgresql.org/download/) (>= 14)
- [Git](https://git-scm.com/)

### 克隆项目

```bash
git clone https://github.com/ZhX589/UniBlack.git
cd UniBlack
```

### 启动后端

```bash
# 安装依赖并构建
cd backend
go mod tidy
go build -o server ./cmd/server

# 启动服务器（需要先启动 PostgreSQL）
./server
```

后端将在 http://localhost:8080 启动

### 启动前端

```bash
# 新开终端
cd frontend
npm install
npm run dev
```

前端将在 http://localhost:3000 启动

### 数据库

确保 PostgreSQL 已启动，然后创建数据库：

```bash
sudo -iu postgres
psql -c "CREATE USER uniblack WITH PASSWORD 'uniblack';"
psql -c "CREATE DATABASE uniblack OWNER uniblack;"
exit
```

后端启动时会自动执行数据库迁移。

### Docker 部署（可选）

Docker 仅用于生产部署，详见 `docker-compose.prod.yml`：

```bash
# 复制环境配置
cp .env.production .env

# 编辑配置（修改密码等）
nano .env

# 启动生产环境
docker compose -f docker-compose.prod.yml up -d
```

生产环境包含：
- PostgreSQL 数据库
- MinIO 对象存储
- 后端 API 服务
- 前端 Web 应用
- Nginx 反向代理

## API 文档

### 公开 API（无需认证）

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/search?q=` | GET | 搜索黑名单 |
| `/api/v1/lookup?platform=&value=` | GET | 按平台查询 |
| `/api/v1/subjects` | GET | 列出黑名单 |
| `/api/v1/subjects/:id` | GET | 获取详情 |
| `/api/v1/cases/:id` | GET | 获取案件 |
| `/api/v1/statistics` | GET | 获取统计 |

### 认证 API（需登录）

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/auth/register` | POST | 注册 |
| `/api/auth/login` | POST | 登录 |
| `/api/auth/refresh` | POST | 刷新Token |

### 业务 API（需登录）

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/subjects` | GET/POST | 管理黑名单 |
| `/api/cases` | GET/POST | 管理案件 |
| `/api/evidence` | POST | 提交证据 |
| `/api/submissions` | GET/POST | 管理举报 |
| `/api/appeals` | GET/POST | 管理申诉 |

## 文档

详细文档可以参考 `./docs`

## 开发

### 技术栈 

本项目采用以下技术栈：

**后端**：
- Go&Echo
- GORM
- PostgreSQL

**前端**:
- React + Next.js
- Tailwind CSS
- shadcn/ui

**认证**:
- JWT + Refresh Token
- OAuth2

### 开发步骤

参考 `docs/roadmap.md`。

### Git 提交指南

#### Git Flow

本项目采用简化修改的Git Flow：
- 功能提交都在 `feature` 下创建子分支，测试后再合并进主分支
- bug修正都在 `fix` 下创建子分支，测试之后再合并进主分支
- 文档更新修正都在 `docs` 下创建子分支

遵循以下步骤：

开发完成  
↓  
PR  
↓  
merge main  
↓  
删除原分支
↓  
Tag

> [!IMPORTANT] 
> 一般情况下不在 `main` 分支下提交代码

## LICENSE

本项目采用**MIT LICENSE**授权

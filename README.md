# UniBlack: 一个可复用的通用云黑系统

[![License](https://shields.io/badge/license-MIT-green)](https://github.com/ZhX589/UniBlack)

UniBlack 是一个可复用的通用云黑系统，支持在线提交&查询、通用查询API、申诉、管理员审核和追责。

> 其中“云黑名单”指的是一个统计曾骗人或做了其他在这个圈子内不被接受的事情的人及其相关社交账号的名单。
> 这仅仅是一个某社区在某时间，对某账号作出了某处理，并公开处理依据、证据、申诉的渠道。一切信息都来源于用户提交，并通过申诉和追责保证可信度、避免诽谤。

---

## 特性

UniBlack 支持：
- ✅ 在线提交云黑信息（三段引导式发布：对象账号 → 事件事实 → 证据与确认）
- ✅ 支持QQ、微信、B站、抖音、X、Telegram、Discord、Steam、手机号等多平台
- ✅ 通用查询API（公开接口）
- ✅ 可信档案：对象详情档案式层级 + 事件四层展示（事实 / 证据 / 申诉纠错）
- ✅ 治理工作台：响应式申诉队列、待办计数、中文状态/裁决标签、逐项裁决
- ✅ 申诉&审核流程（事件申诉 → outcome 裁决 → 更正/撤回/恶意提交）
- ✅ 完善的用户系统（RBAC权限控制）
- ✅ 证据管理（图片、文件、链接，只读详情预加载）
- ✅ 审计日志
- ✅ 响应式前端（375 / 768 / 1280 三档，键盘可达，reduced-motion）

## 生产部署

### 前置要求

服务器仅需 Docker（≥ 24，含 Compose V2）和 Git，无需安装 Go / Node.js / PostgreSQL。

```bash
docker --version
docker compose version
git --version
```

### 获取代码

```bash
git clone https://github.com/ZhX589/UniBlack.git
cd UniBlack
```

### 配置环境变量

`.env` 已被 gitignore，从生产模板创建：

```bash
cp .env.production .env
nano .env
```

**必改项**（生产环境使用默认值会拒绝启动，由 `config.Validate()` 强制校验）：

| 变量 | 说明 |
|------|------|
| `POSTGRES_PASSWORD` | 数据库强密码 |
| `JWT_SECRET` | JWT 签名密钥（≥ 32 字符） |
| `REFRESH_SECRET` | Refresh Token 密钥（≥ 32 字符） |
| `MINIO_ROOT_PASSWORD` | MinIO 对象存储密码 |
| `API_BASE_URL` | 前端使用的 API 地址（如 `https://api.yourdomain.com`） |

### HTTPS / SSL 配置（生产域名必做）

`nginx/nginx.conf` 默认只监听 HTTP (`:80`)，443 块为注释状态。

```bash
mkdir -p nginx/ssl
# 将证书放入 nginx/ssl/cert.pem 和 nginx/ssl/key.pem
```

编辑 `nginx/nginx.conf`，取消注释 HTTPS server 块并将 `server_name` 改为你的域名，同时启用 80→443 跳转。若仅在内网或 IP 上运行，可保持 HTTP。

### 构建并启动

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

首次构建约 30-60 秒，后端日志正常输出：

```
Database connected and migrations applied
Server starting on port 8080
```

### 服务架构

```
                           ┌──────────┐
      :80 / :443           │  nginx   │ (反向代理 + 限速 10r/s)
  ────────────────────────►│          │
                           └─┬────┬──┘
                  /api/      │    │  / 其他
                    ▼        │    │        ▼
             ┌─────────┐    │    │   ┌──────────┐
             │ backend │◄───┘    └──►│ frontend │
             │   :8080 │            │   :3000  │
             └───┬──┬──┘            └──────────┘
         ┌───────┘  └────────┐
         ▼                   ▼
   ┌──────────┐        ┌─────────┐
   │ postgres │        │  minio  │  (证据文件)
   │  :5432   │        │ :9000   │
   └──────────┘        └─────────┘
```

| 服务 | 数据卷 | 对外端口 | 重启策略 |
|------|--------|----------|----------|
| postgres | `postgres_data` | 无（仅内网） | `unless-stopped` |
| minio | `minio_data` | 无（仅内网） | `unless-stopped` |
| backend | 无（DB 在 PG） | 无（仅内网） | `unless-stopped` |
| frontend | 无 | 无（仅内网） | `unless-stopped` |
| nginx | 无 | **80, 443** | `unless-stopped` |

生产 compose 不暴露 PostgreSQL / MinIO / 后端 / 前端的端口，所有流量经 nginx 单入口。

### 首次初始化管理员

生产环境不会自动 seed 管理员。通过 setup 接口创建第一个管理员：

```bash
# 检查是否需初始化
curl http://your-domain/api/setup/check

# 创建管理员
curl -X POST http://your-domain/api/setup/initialize \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","email":"admin@yourdomain.com","password":"your-secure-password"}'
```

### 数据库迁移

后端每次启动自动执行 golang-migrate（`backend/internal/migrations/` 已打进镜像），无需手动操作。

### 日常运维

```bash
# 查看状态
docker compose -f docker-compose.prod.yml ps

# 查看日志
docker compose -f docker-compose.prod.yml logs -f backend

# 代码更新后重新构建上线
git pull origin main
docker compose -f docker-compose.prod.yml up -d --build

# 重启单个服务
docker compose -f docker-compose.prod.yml restart nginx

# 停止全部
docker compose -f docker-compose.prod.yml down
```

### 数据备份

```bash
# 数据库
docker compose -f docker-compose.prod.yml exec postgres pg_dump -U uniblack uniblack > backup.sql

# 恢复
docker compose -f docker-compose.prod.yml exec -T postgres psql -U uniblack uniblack < backup.sql

# 证据文件（Docker 卷）
docker volume inspect uniblack_minio_data
```

### 常见问题

| 现象 | 原因 | 解决 |
|------|------|------|
| 后端 FATAL: `DATABASE_URL is required` | `.env` 未配 | `nano .env` 补齐变量 |
| 后端 FATAL: `JWT_SECRET must be configured` | 密钥使用默认值 | 改为 ≥ 32 字符随机串 |
| nginx 502 Bad Gateway | 后端尚未就绪 | `docker compose logs backend` 检查迁移是否失败 |
| 上传文件失败 | MinIO bucket 未创建 | `docker compose exec minio mc mb local/uniblack-evidence` |

---

## 开发环境

### 前置要求

| 依赖 | 版本 | 安装 |
|------|------|------|
| Go | ≥ 1.22 | [go.dev/dl](https://go.dev/dl/) |
| Node.js | ≥ 18 | [nodejs.org](https://nodejs.org/) |
| PostgreSQL | ≥ 14 | `sudo apt install postgresql` 或 `brew install postgresql@14` |
| Git | 任意 | 系统自带 |

> 开发模式不需要 MinIO —— 后端在未设 `MINIO_ENDPOINT` 时自动使用本地文件存储 (`./uploads`)。

### 克隆项目

```bash
git clone https://github.com/ZhX589/UniBlack.git
cd UniBlack
```

### 数据库

确保 PostgreSQL 已运行，创建用户和数据库：

```bash
sudo -iu postgres <<'SQL'
psql -c "CREATE USER uniblack WITH PASSWORD 'uniblack';"
psql -c "CREATE DATABASE uniblack OWNER uniblack;"
SQL
```

### 启动后端

```bash
cd backend
go mod tidy
go build -o server ./cmd/server
./server
```

后端在 `http://localhost:8080` 启动。若 PostgreSQL 不在本地或用户名/密码不同，用环境变量覆盖：

```bash
DATABASE_URL="postgres://user:pass@host:5432/uniblack?sslmode=disable" ./server
```

启动后输出 `Database connected and migrations applied` 即迁移完成。

**开发模式自动行为**：
- 管理员账号 `admin@uniblack.local` / `admin123` 自动创建
- 验证码固定为 `123456`（不走真实 SMTP）
- 存储使用 `./uploads` 本地目录
- 人机验证使用内置 Demo 组件

### 启动前端

```bash
# 新开终端
cd frontend
npm install
npm run dev
```

前端在 `http://localhost:3000`，热更新。`next.config.js` 内重写 `/api/*` → `http://localhost:8080`，API 请求自动代理到后端，开发时只开 `:3000` 即可。

### 开发 Compose 快捷启动（不想手动装 PostgreSQL）

开发 compose 全部服务开箱即用（弱口令，**仅限本地**）：

```bash
docker compose up -d
```

暴露端口：后端 `:8080` · 前端 `:3000` · PostgreSQL `:5432` · MinIO Console `:9001`。

也可以只启动数据库，自行跑后端+前端：

```bash
docker compose up -d postgres
cd backend && ./server     # 终端 1
cd frontend && npm run dev  # 终端 2
```

### 开发常用命令

```bash
# 后端
cd backend
go build ./...          # 编译检查
go test ./...           # 运行测试
go vet ./...            # 静态分析

# 前端
cd frontend
npm run lint            # ESLint 检查
npm run typecheck       # TypeScript 类型检查
npm run test:run        # 运行测试
npm run build           # 生产构建检查
```

## API 文档

> **Canonical model is Event-first** (Subject → Account → Event).  
> Legacy Case routes remain until **2026-12-31** with `Deprecation` / `Sunset` headers — see `docs/api/case-event-migration.md` and `docs/product-decisions.md`.

### 公开 API（无需认证）

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/search?q=` | GET | 搜索黑名单（Account-first） |
| `/api/v1/lookup?platform=&value=` | GET | 按平台查询 |
| `/api/v1/subjects` | GET | 列出对象 |
| `/api/v1/subjects/:id` | GET | 对象详情（含公开事件） |
| `/api/v1/events/:id` | GET | 事件详情（含 `evidence` 只读列表，按创建时间序） |
| `/api/v1/cases/:id` | GET | **遗留**案件读；响应含 Deprecation/Sunset，请迁移到 events |
| `/api/v1/statistics` | GET | 统计 |

### 认证 API（需登录）

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/auth/register` | POST | 注册 |
| `/api/auth/login` | POST | 登录 |
| `/api/auth/refresh` | POST | 刷新Token |
| `/api/auth/send-verification-code` | POST | 发邮箱验证码（同邮箱+purpose 60s 限流） |

### 业务 API（需登录）

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/subjects` | GET/POST | 管理对象 |
| `/api/subjects/publish` | POST | 默认发布对象+账号+事件（JSON 或 multipart） |
| `/api/events/:id` | GET | 已登录事件读 |
| `/api/events/:id/appeals` | GET/POST | 事件申诉历史 / 提交申诉 |
| `/api/appeals` | GET | 申诉列表（审核） |
| `/api/appeals/:id/review/resolve` | POST | 申诉裁决（outcome） |
| `/api/sanctions` / `/api/admin/sanctions` | * | 处罚与处罚申诉 |
| `/api/cases`、`/api/submissions` | * | **兼容窗口**遗留接口，勿新接入 |

公开读不依赖 API Key（产品决策：不实现 API Key）。写操作使用会话 JWT + RBAC。

## 文档

详细文档可以参考 `./docs`：

- `docs/roadmap.md`：开发阶段、目标与验收标准（Status 为进度；Goal 为规划）
- `docs/configuration.md`：站点、注册、邮件与人机验证配置
- `docs/product-decisions.md`：锁定的产品决策与 Sunset 检查清单
- `docs/api/case-event-migration.md`：Case → Event 弃用与替代端点
- `DESIGN.md`：前端视觉、交互和可访问性规范
- `docs/frontend-gap-analysis.md`：前端实现进度与差距台账（滚动更新）
- `docs/implementation-gap-analysis.md`：全栈实现进度与差距台账（滚动更新）
- `docs/compose/specs/`：设计规格（规划文档，不随实现改写目标）
- `docs/compose/plans/`：分阶段实施计划（规划文档）
- `docs/compose/reports/`：功能交付最终报告（反映当前实现）

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

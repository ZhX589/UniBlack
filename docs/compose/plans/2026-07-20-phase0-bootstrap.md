# Phase 0 - Project Bootstrap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use compose:subagent (recommended) or compose:execute to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Initialize UniBlack project with complete directory structure, Docker Compose environments, and basic runnable backend/frontend.

**Architecture:** Step-by-step implementation: 1) Project structure, 2) Docker Compose (dev), 3) Docker Compose (prod), 4) Backend basic code, 5) Frontend basic code, 6) Database migration skeleton, 7) Environment configuration.

**Tech Stack:** Go + Echo, React + Next.js, PostgreSQL, MinIO, Docker Compose, golang-migrate

## Global Constraints

- Project name: UniBlack (never CloudBan)
- Go module path: `github.com/ZhX589/UniBlack/backend`
- Backend structure: `handler -> service -> repository -> db/models` (single direction)
- Database migrations: golang-migrate SQL files in `backend/internal/migrations/`
- Environment: `.env` gitignored, `.env.example` committed
- Docker Compose: simple separation (dev + prod files)

---

### Task 1: Create Backend Directory Structure

**Covers:** Phase 0 - Project structure initialization

**Files:**
- Create: `backend/cmd/server/main.go`
- Create: `backend/internal/config/config.go`
- Create: `backend/internal/db/db.go`
- Create: `backend/internal/migrations/` (directory)
- Create: `backend/internal/models/` (directory)
- Create: `backend/internal/repository/` (directory)
- Create: `backend/internal/service/` (directory)
- Create: `backend/internal/handler/` (directory)
- Create: `backend/internal/middleware/` (directory)
- Create: `backend/internal/auth/` (directory)
- Create: `backend/internal/storage/` (directory)
- Create: `backend/pkg/` (directory)
- Create: `backend/go.mod`
- Create: `backend/Dockerfile`

**Interfaces:**
- Consumes: None (initial setup)
- Produces: Complete backend directory structure with placeholder files

- [ ] **Step 1: Create backend directory structure**

```bash
cd /data/Projects/UniBlack
mkdir -p backend/cmd/server
mkdir -p backend/internal/{config,db,migrations,models,repository,service,handler,middleware,auth,storage}
mkdir -p backend/pkg
```

- [ ] **Step 2: Initialize Go module**

```bash
cd /data/Projects/UniBlack/backend
go mod init github.com/ZhX589/UniBlack/backend
```

- [ ] **Step 3: Create basic main.go**

```go
// backend/cmd/server/main.go
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("UniBlack server starting...")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server running on port %s\n", port)
}
```

- [ ] **Step 4: Create Dockerfile**

```dockerfile
# backend/Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
CMD ["./server"]
```

- [ ] **Step 5: Commit changes**

```bash
git add backend/
git commit -m "feat: create backend directory structure and basic main.go"
```

### Task 2: Create Frontend Directory Structure

**Covers:** Phase 0 - Project structure initialization

**Files:**
- Create: `frontend/app/` (directory)
- Create: `frontend/components/ui/` (directory)
- Create: `frontend/components/` (directory)
- Create: `frontend/lib/` (directory)
- Create: `frontend/styles/` (directory)
- Create: `frontend/.env.example`
- Create: `frontend/package.json`
- Create: `frontend/next.config.js`
- Create: `frontend/tailwind.config.js`
- Create: `frontend/postcss.config.js`
- Create: `frontend/Dockerfile`

**Interfaces:**
- Consumes: None (initial setup)
- Produces: Complete frontend directory structure with Next.js configuration

- [ ] **Step 1: Create frontend directory structure**

```bash
cd /data/Projects/UniBlack
mkdir -p frontend/app
mkdir -p frontend/components/ui
mkdir -p frontend/components
mkdir -p frontend/lib
mkdir -p frontend/styles
```

- [ ] **Step 2: Initialize Next.js project**

```bash
cd /data/Projects/UniBlack/frontend
npm init -y
npm install next react react-dom
npm install -D typescript @types/react @types/node tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

- [ ] **Step 3: Create package.json scripts**

```json
{
  "name": "uniblack-frontend",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint"
  },
  "dependencies": {
    "next": "14.0.0",
    "react": "18.2.0",
    "react-dom": "18.2.0"
  },
  "devDependencies": {
    "@types/node": "20.10.0",
    "@types/react": "18.2.0",
    "autoprefixer": "10.4.16",
    "postcss": "8.4.32",
    "tailwindcss": "3.3.6",
    "typescript": "5.3.2"
  }
}
```

- [ ] **Step 4: Create basic page**

```tsx
// frontend/app/page.tsx
export default function Home() {
  return (
    <main>
      <h1>UniBlack</h1>
      <p>云黑名单系统</p>
    </main>
  )
}
```

- [ ] **Step 5: Create Dockerfile**

```dockerfile
# frontend/Dockerfile
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:18-alpine AS runner
WORKDIR /app
ENV NODE_ENV production
COPY --from=builder /app/public ./public
COPY --from=builder --chown=node:node /app/.next ./.next
COPY --from=builder --chown=node:node /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./package.json
USER node
EXPOSE 3000
CMD ["npm", "start"]
```

- [ ] **Step 6: Commit changes**

```bash
git add frontend/
git commit -m "feat: create frontend directory structure and basic Next.js setup"
```

### Task 3: Create Docker Compose Development Environment

**Covers:** Phase 0 - Docker Compose configuration

**Files:**
- Create: `docker-compose.yml`
- Create: `docker-compose.prod.yml`

**Interfaces:**
- Consumes: Backend and frontend Dockerfiles from Tasks 1-2
- Produces: Working Docker Compose configuration for development

- [ ] **Step 1: Create docker-compose.yml (development)**

```yaml
# docker-compose.yml
version: '3.8'

services:
  postgres:
    image: postgres:14-alpine
    environment:
      POSTGRES_USER: uniblack
      POSTGRES_PASSWORD: uniblack
      POSTGRES_DB: uniblack
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U uniblack"]
      interval: 10s
      timeout: 5s
      retries: 5

  minio:
    image: minio/minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "mc", "ready", "local"]
      interval: 10s
      timeout: 5s
      retries: 5

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      PORT: 8080
      DATABASE_URL: postgres://uniblack:uniblack@postgres:5432/uniblack?sslmode=disable
      JWT_SECRET: dev-jwt-secret
      REFRESH_SECRET: dev-refresh-secret
      MINIO_ENDPOINT: minio:9000
      MINIO_ACCESS_KEY: minioadmin
      MINIO_SECRET_KEY: minioadmin
      MINIO_BUCKET: uniblack-evidence
      MINIO_USE_SSL: "false"
    depends_on:
      postgres:
        condition: service_healthy
      minio:
        condition: service_healthy

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    environment:
      NEXT_PUBLIC_API_BASE: http://localhost:8080
    depends_on:
      - backend

volumes:
  postgres_data:
  minio_data:
```

- [ ] **Step 2: Create docker-compose.prod.yml (production)**

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  postgres:
    image: postgres:14-alpine
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  minio:
    image: minio/minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "mc", "ready", "local"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  backend:
    image: ${DOCKER_REGISTRY}/uniblack-backend:${VERSION}
    environment:
      PORT: 8080
      DATABASE_URL: postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable
      JWT_SECRET: ${JWT_SECRET}
      REFRESH_SECRET: ${REFRESH_SECRET}
      MINIO_ENDPOINT: minio:9000
      MINIO_ACCESS_KEY: ${MINIO_ROOT_USER}
      MINIO_SECRET_KEY: ${MINIO_ROOT_PASSWORD}
      MINIO_BUCKET: ${MINIO_BUCKET}
      MINIO_USE_SSL: "false"
    depends_on:
      postgres:
        condition: service_healthy
      minio:
        condition: service_healthy
    restart: unless-stopped

  frontend:
    image: ${DOCKER_REGISTRY}/uniblack-frontend:${VERSION}
    environment:
      NEXT_PUBLIC_API_BASE: ${API_BASE_URL}
    depends_on:
      - backend
    restart: unless-stopped

volumes:
  postgres_data:
  minio_data:
```

- [ ] **Step 3: Commit changes**

```bash
git add docker-compose.yml docker-compose.prod.yml
git commit -m "feat: add Docker Compose configurations for dev and prod"
```

### Task 4: Create Database Migration Skeleton

**Covers:** Phase 0 - golang-migrate setup

**Files:**
- Create: `backend/internal/migrations/000001_init.up.sql`
- Create: `backend/internal/migrations/000001_init.down.sql`
- Create: `backend/internal/db/migrate.go`

**Interfaces:**
- Consumes: Database configuration from environment
- Produces: Migration skeleton and database connection code

- [ ] **Step 1: Create initial migration**

```sql
-- backend/internal/migrations/000001_init.up.sql
-- Initial database schema
-- This will be expanded in Phase 1

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    auth_provider VARCHAR(50) DEFAULT 'local',
    external_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);
```

```sql
-- backend/internal/migrations/000001_init.down.sql
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
```

- [ ] **Step 2: Create database connection code**

```go
// backend/internal/db/db.go
package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func Connect() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
```

- [ ] **Step 3: Create migrate helper**

```go
// backend/internal/db/migrate.go
package db

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(db *sql.DB) error {
	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "./internal/migrations"
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
```

- [ ] **Step 4: Update main.go to use database**

```go
// backend/cmd/server/main.go
package main

import (
	"fmt"
	"os"

	"github.com/ZhX589/UniBlack/backend/internal/db"
)

func main() {
	fmt.Println("UniBlack server starting...")

	// Connect to database
	database, err := db.Connect()
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Run migrations
	if err := db.RunMigrations(database); err != nil {
		fmt.Printf("Failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Database connected and migrations applied")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server running on port %s\n", port)
}
```

- [ ] **Step 5: Update go.mod with dependencies**

```bash
cd /data/Projects/UniBlack/backend
go get github.com/lib/pq
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
```

- [ ] **Step 6: Commit changes**

```bash
git add backend/internal/migrations/ backend/internal/db/ backend/cmd/server/main.go backend/go.mod backend/go.sum
git commit -m "feat: add database migration skeleton and connection code"
```

### Task 5: Create Environment Configuration

**Covers:** Phase 0 - Environment configuration

**Files:**
- Modify: `backend/.env.example`
- Modify: `frontend/.env.example`
- Create: `.env.example` (root)

**Interfaces:**
- Consumes: All previous tasks
- Produces: Complete environment configuration templates

- [ ] **Step 1: Update backend .env.example**

```bash
# backend/.env.example
# UniBlack backend env template
# Copy to backend/.env and fill in. DO NOT commit the real .env.

# Server
PORT=8080
ECHO_MODE=debug

# PostgreSQL
DATABASE_URL=postgres://uniblack:uniblack@localhost:5432/uniblack?sslmode=disable

# JWT / Refresh tokens
JWT_SECRET=change-me-access-secret
REFRESH_SECRET=change-me-refresh-secret
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=7d

# MinIO (S3-compatible) for evidence storage
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=uniblack-evidence
MINIO_USE_SSL=false

# Migrations
MIGRATIONS_PATH=./internal/migrations
```

- [ ] **Step 2: Update frontend .env.example**

```bash
# frontend/.env.example
# UniBlack frontend env template
# Copy to frontend/.env.local and fill in. DO NOT commit the real .env*.

# Base URL of the UniBlack backend API
NEXT_PUBLIC_API_BASE=http://localhost:8080
```

- [ ] **Step 3: Create root .env.example**

```bash
# .env.example
# UniBlack root env template
# Copy to .env and fill in. DO NOT commit the real .env.

# Docker Compose
COMPOSE_PROJECT_NAME=uniblack

# PostgreSQL
POSTGRES_USER=uniblack
POSTGRES_PASSWORD=uniblack
POSTGRES_DB=uniblack

# MinIO
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin

# Backend
JWT_SECRET=change-me-access-secret
REFRESH_SECRET=change-me-refresh-secret

# Frontend
API_BASE_URL=http://localhost:8080

# Docker Registry (for production)
DOCKER_REGISTRY=your-registry.com
VERSION=latest
```

- [ ] **Step 4: Commit changes**

```bash
git add backend/.env.example frontend/.env.example .env.example
git commit -m "feat: update environment configuration templates"
```

### Task 6: Test Docker Compose Development Environment

**Covers:** Phase 0 - Verification

**Files:**
- None (testing only)

**Interfaces:**
- Consumes: All previous tasks
- Produces: Verified working development environment

- [ ] **Step 1: Build and start services**

```bash
cd /data/Projects/UniBlack
docker compose build
docker compose up -d
```

- [ ] **Step 2: Verify services are running**

```bash
docker compose ps
```

Expected output: All services (postgres, minio, backend, frontend) should be running.

- [ ] **Step 3: Test backend health**

```bash
curl http://localhost:8080
```

Expected output: Server response (may be 404 initially, but server should be running)

- [ ] **Step 4: Test frontend**

```bash
curl http://localhost:3000
```

Expected output: HTML response from Next.js

- [ ] **Step 5: Test PostgreSQL connection**

```bash
docker compose exec postgres psql -U uniblack -d uniblack -c "\dt"
```

Expected output: List of tables (users, roles, permissions, etc.)

- [ ] **Step 6: Test MinIO console**

Open browser to `http://localhost:9001`
Login with: minioadmin / minioadmin

- [ ] **Step 7: Stop services**

```bash
docker compose down
```

- [ ] **Step 8: Commit verification results**

```bash
git add .
git commit -m "test: verify Docker Compose development environment works"
```

### Task 7: Add GitHub Actions CI/CD

**Covers:** Phase 0 - CI/CD configuration

**Files:**
- Create: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: All previous tasks
- Produces: GitHub Actions workflow for lint, build, test, migrate

- [ ] **Step 1: Create GitHub Actions workflow**

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main, feature/*, fix/*]
  pull_request:
    branches: [main]

jobs:
  backend:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:14-alpine
        env:
          POSTGRES_USER: uniblack
          POSTGRES_PASSWORD: uniblack
          POSTGRES_DB: uniblack_test
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Build backend
        run: |
          cd backend
          go build ./...

      - name: Run backend tests
        run: |
          cd backend
          go test ./...

      - name: Run migrations
        run: |
          cd backend
          go run ./cmd/server/main.go
        env:
          DATABASE_URL: postgres://uniblack:uniblack@localhost:5432/uniblack_test?sslmode=disable
          JWT_SECRET: test-secret
          REFRESH_SECRET: test-refresh-secret

  frontend:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '18'

      - name: Install dependencies
        run: |
          cd frontend
          npm ci

      - name: Build frontend
        run: |
          cd frontend
          npm run build

      - name: Run frontend lint
        run: |
          cd frontend
          npm run lint

  lint:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest
          working-directory: backend

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '18'

      - name: Run frontend lint
        run: |
          cd frontend
          npm ci
          npm run lint
```

- [ ] **Step 2: Commit changes**

```bash
mkdir -p .github/workflows
git add .github/workflows/ci.yml
git commit -m "ci: add GitHub Actions workflow for lint, build, test"
```

### Task 8: Add Code Formatting Configuration

**Covers:** Phase 0 - Code formatting setup

**Files:**
- Create: `backend/.golangci.yml`
- Create: `frontend/.eslintrc.json`
- Create: `frontend/.prettierrc`
- Create: `frontend/.editorconfig`

**Interfaces:**
- Consumes: All previous tasks
- Produces: Code formatting configuration for backend and frontend

- [ ] **Step 1: Create golangci-lint configuration**

```yaml
# backend/.golangci.yml
run:
  timeout: 5m
  modules-download-mode: readonly

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports
    - misspell
    - unconvert
    - unparam

linters-settings:
  gofmt:
    simplify: true
  goimports:
    local-prefixes: github.com/ZhX589/UniBlack/backend

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
```

- [ ] **Step 2: Create ESLint configuration**

```json
// frontend/.eslintrc.json
{
  "extends": [
    "next/core-web-vitals",
    "eslint:recommended"
  ],
  "rules": {
    "react/no-unescaped-entities": "off",
    "@next/next/no-page-custom-font": "off"
  }
}
```

- [ ] **Step 3: Create Prettier configuration**

```json
// frontend/.prettierrc
{
  "semi": false,
  "trailingComma": "es5",
  "singleQuote": true,
  "printWidth": 100,
  "tabWidth": 2,
  "useTabs": false
}
```

- [ ] **Step 4: Create EditorConfig**

```ini
// frontend/.editorconfig
root = true

[*]
indent_style = space
indent_size = 2
end_of_line = lf
charset = utf-8
trim_trailing_whitespace = true
insert_final_newline = true

[*.md]
trim_trailing_whitespace = false

[Makefile]
indent_style = tab
```

- [ ] **Step 5: Commit changes**

```bash
git add backend/.golangci.yml frontend/.eslintrc.json frontend/.prettierrc frontend/.editorconfig
git commit -m "style: add code formatting configuration for backend and frontend"
```

### Task 9: Create Basic README Quick Start

**Covers:** Phase 0 - Documentation

**Files:**
- Modify: `README.md`

**Interfaces:**
- Consumes: All previous tasks
- Produces: Updated README with Quick Start instructions

- [ ] **Step 1: Update README.md**

```markdown
# UniBlack: 一个可复用的通用云黑系统

[![License](https://shields.io/badge/license-MIT-green)](https://github.com/ZhX589/UniBlack)

UniBlack 是一个可复用的通用云黑系统，支持在线提交&查询、通用查询API、申诉、管理员审核和追责。

> 其中"云黑名单"指的是一个统计曾骗人或做了其他在这个圈子内不被接受的事情的人及其相关社交账号的名单。
> 这仅仅是一个某社区在某时间，对某账号作出了某处理，并公开处理依据、证据、申诉的渠道。一切信息都来源于用户提交，并通过申诉和追责保证可信度、避免诽谤。

---

## 特性

UniBlack 支持
- [ ] 在线提交云黑信息
- [ ] 支持QQ、账号名等多种信息
- [ ] 通用查询API
- [ ] 申诉&审核
- [ ] 完善的用户系统

## 快速上手

### 前置要求

- Docker 和 Docker Compose
- Git

### 启动开发环境

```bash
# 克隆仓库
git clone https://github.com/ZhX589/UniBlack.git
cd UniBlack

# 复制环境变量模板
cp .env.example .env
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env.local

# 启动所有服务
docker compose up -d

# 访问应用
# 前端: http://localhost:3000
# 后端API: http://localhost:8080
# MinIO控制台: http://localhost:9001 (minioadmin/minioadmin)
```

### 停止服务

```bash
docker compose down
```

## 文档

文档可以参考 `./docs`

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
```

- [ ] **Step 2: Commit changes**

```bash
git add README.md
git commit -m "docs: update README with Quick Start instructions"
```

---

## Execution Handoff

Plan saved to `/data/Projects/UniBlack/docs/compose/plans/2026-07-20-phase0-bootstrap.md`.

**Execution approach:** The plan has 9 tasks with clear dependencies. Recommend using **compose:execute** for batch execution with checkpoints, as tasks are sequential and tightly coupled.
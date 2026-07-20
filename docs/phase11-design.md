# Phase 11 - Admin Console & Enhanced Registration 设计文档

## 功能概述

Phase 11 为 UniBlack 添加管理控制台和增强注册功能，使系统更加灵活和安全。

---

## 1. 注册页面增强

### 1.1 邮箱验证

**流程**：
```
用户填写邮箱 → 发送验证码 → 用户输入验证码 → 验证通过 → 完成注册
```

**接口设计**：
```
POST /api/auth/send-verification-code
  Body: { email: string }
  Response: { message: "验证码已发送" }

POST /api/auth/register
  Body: { 
    username: string, 
    email: string, 
    password: string,
    verification_code: string,
    captcha_token: string
  }
```

### 1.2 人机验证

**支持的提供商**：
- Google reCAPTCHA v2/v3
- hCaptcha
- Cloudflare Turnstile

**配置存储**：
```json
{
  "provider": "turnstile",  // recaptcha / hcaptcha / turnstile
  "site_key": "1x00000000000000000000AA",
  "secret_key": "1x0000000000000000000000000000000AA"
}
```

### 1.3 注册协议

**配置存储**：
```json
{
  "enabled": true,
  "title": "用户注册协议",
  "content": "..."
}
```

---

## 2. 管理控制台

### 2.1 页面结构

```
/admin/settings
├── /basic          # 基础配置
├── /security       # 安全配置
├── /auth           # 登录配置
├── /users          # 用户管理
├── /access-lists   # 名单管理
└── /system         # 系统信息
```

### 2.2 基础配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| site.name | string | UniBlack | 项目名称 |
| site.description | string | ... | 项目描述 |
| site.theme_color | string | #3B82F6 | 主题色 |
| site.logo_url | string | "" | Logo URL |
| site.contact_email | string | "" | 联系邮箱 |

### 2.3 安全配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| security.email_verification | boolean | false | 邮箱验证开关 |
| security.smtp_host | string | "" | SMTP 服务器 |
| security.smtp_port | number | 587 | SMTP 端口 |
| security.smtp_username | string | "" | SMTP 用户名 |
| security.smtp_password | string | "" | SMTP 密码 |
| security.smtp_from | string | "" | 发件人地址 |
| security.captcha_enabled | boolean | false | 人机验证开关 |
| security.captcha_provider | string | turnstile | 提供商 |
| security.captcha_site_key | string | "" | Site Key |
| security.captcha_secret_key | string | "" | Secret Key |
| security.rate_limit_public | number | 20 | 公开API限速 (req/s) |
| security.rate_limit_auth | number | 10 | 认证API限速 (req/s) |

### 2.4 登录配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| auth.registration_enabled | boolean | true | 注册开关 |
| auth.oauth_github_enabled | boolean | false | GitHub登录 |
| auth.oauth_github_client_id | string | "" | Client ID |
| auth.oauth_github_client_secret | string | "" | Client Secret |

---

## 3. 用户管理

### 3.1 功能列表

| 功能 | 说明 |
|------|------|
| 用户列表 | 分页、搜索、筛选（角色/状态） |
| 用户详情 | 查看用户信息、登录历史、操作记录 |
| 禁用/启用 | 切换用户状态 |
| 角色分配 | 修改用户角色 |
| 重置密码 | 发送重置密码邮件 |

### 3.2 接口设计

```
GET  /api/admin/users?page=1&page_size=20&search=&role=&status=
GET  /api/admin/users/:id
PUT  /api/admin/users/:id
PUT  /api/admin/users/:id/role
POST /api/admin/users/:id/reset-password
```

---

## 4. 名单管理

### 4.1 白名单

| 类型 | 说明 | 示例 |
|------|------|------|
| IP | 跳过限速 | 192.168.1.1 |
| 用户名 | 跳过查询限制 | vip_user |

### 4.2 黑名单

| 类型 | 说明 | 示例 |
|------|------|------|
| IP | 禁止访问 | 10.0.0.1 |
| 邮箱 | 禁止注册 | spam@example.com |
| 用户名 | 禁止注册 | bad_actor |

### 4.3 接口设计

```
GET    /api/admin/access-lists?type=whitelist&page=1
POST   /api/admin/access-lists
DELETE /api/admin/access-lists/:id
POST   /api/admin/access-lists/import
GET    /api/admin/access-lists/export
```

---

## 5. 初始配置流程

### 5.1 生产环境

**首次启动检测**：
```go
// 检查是否已初始化
func isFirstRun(db *gorm.DB) bool {
    var count int64
    db.Model(&models.User{}).Where("username = ?", "admin").Count(&count)
    return count == 0
}
```

**初始化流程**：
```
1. 检测到首次启动
2. 返回 302 重定向到 /setup
3. 显示初始化页面：
   - 设置 admin 密码
   - 配置站点名称（可选）
   - 配置 SMTP（可选）
4. 完成初始化，跳转到登录页
```

### 5.2 开发环境

**默认配置**：
```go
// backend/internal/seed/seed.go
func SeedAdmin(db *gorm.DB) {
    // 检查是否存在 admin 用户
    var count int64
    db.Model(&models.User{}).Where("username = ?", "admin").Count(&count)
    
    if count == 0 {
        // 创建默认 admin 用户
        admin := &models.User{
            Username:     "admin",
            Email:        "admin@uniblack.local",
            PasswordHash: hashPassword("admin123"),
            AuthProvider: "local",
            IsActive:     true,
        }
        db.Create(admin)
        
        // 分配 admin 角色
        // ...
    }
}
```

---

## 6. 开发路径

### Task 1: 数据库迁移
- 创建 `system_settings` 表
- 创建 `access_lists` 表
- 添加默认配置数据

### Task 2: 系统配置服务
- SystemSettingRepository
- SystemSettingService
- 配置缓存机制

### Task 3: 名单管理服务
- AccessListRepository
- AccessListService

### Task 4: 用户管理 API
- 用户列表/详情/更新/禁用
- 角色分配
- 重置密码

### Task 5: 注册增强
- 邮箱验证码发送/验证
- 人机验证集成
- 注册流程改造

### Task 6: 管理控制台前端
- 基础配置页面
- 安全配置页面
- 登录配置页面
- 用户管理页面
- 名单管理页面

### Task 7: 初始化流程
- 生产环境首次启动检测
- 初始化页面
- 开发环境默认数据

### Task 8: 文档更新
- 更新 README
- 更新 API 文档

---

## 7. 配置生效机制

### 7.1 缓存策略

```go
type ConfigCache struct {
    cache map[string]interface{}
    mu    sync.RWMutex
}

func (c *ConfigCache) Get(key string) interface{} {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.cache[key]
}

func (c *ConfigCache) Set(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.cache[key] = value
}
```

### 7.2 实时生效

配置变更后：
1. 更新数据库
2. 更新缓存
3. 需要重启的配置（如SMTP）给出提示
4. 立即生效的配置（如限速）直接生效

---

## 8. 安全考虑

### 8.1 敏感信息存储

- SMTP 密码、OAuth Secret 等敏感信息在 API 返回时脱敏
- 只显示是否已配置，不显示实际值

```json
{
  "smtp_password": "••••••••",
  "smtp_configured": true
}
```

### 8.2 权限控制

- 只有 admin 角色可以访问管理控制台
- 配置变更记录审计日志

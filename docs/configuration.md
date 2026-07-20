# UniBlack 可配置项说明（注册 / 邮箱 / 人机验证）

设计参考 [NewAPI](https://github.com/QuantumNous/new-api) 的 `Option` 模型：

```
环境变量默认值 (.env)  →  内存 OptionMap  →  数据库 system_settings 覆盖  →  控制台 / API 读写
```

运行时读取顺序：**内存缓存（启动时已 merge DB）→ DB → 环境默认值**。

---

## 配置接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/settings/public` | 公开 | 注册页/前台所需非密钥配置 |
| GET | `/api/admin/settings` | admin | 返回 `schema` + `settings` + `values` |
| GET | `/api/admin/settings/schema` | admin | 仅配置目录（动态表单） |
| PUT | `/api/admin/settings` | admin | 批量更新 `[{ "key", "value" }]` |
| POST | `/api/auth/send-verification-code` | 公开 | 发送注册邮箱验证码 |
| POST | `/api/auth/verify-email` | 公开 | 校验验证码 |
| POST | `/api/auth/register` | 公开 | 注册（可要求 captcha + 验证码） |

### GET `/api/admin/settings` 响应形状

```json
{
  "schema": [
    {
      "key": "security.captcha_provider",
      "category": "security",
      "type": "select",
      "label": "人机验证提供商",
      "options": ["turnstile", "recaptcha", "hcaptcha", "none"],
      "public": true,
      "secret": false
    }
  ],
  "settings": [
    { "key": "security.captcha_secret_key", "value": "\"••••••••\"", "secret": true, "configured": true }
  ],
  "values": {
    "security.captcha_enabled": false,
    "security.captcha_secret_key": { "configured": true, "redacted": true }
  }
}
```

密钥字段只回传是否已配置，不回传明文；PUT 时传空或 `••••••••` 表示不修改。

---

## 环境变量（`.env` / `.env.example`）

| 变量 | 对应 key | 说明 |
|------|----------|------|
| `SITE_NAME` | `site.name` | 项目显示名 |
| `REGISTER_ENABLED` | `auth.registration_enabled` | 是否开放注册 |
| `EMAIL_VERIFICATION_ENABLED` | `security.email_verification` | 邮箱验证 |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USERNAME` / `SMTP_PASSWORD` / `SMTP_FROM` | `security.smtp_*` | 邮件 |
| `SMTP_SSL_ENABLED` | `security.smtp_ssl` | 465 隐式 TLS |
| `CAPTCHA_ENABLED` | `security.captcha_enabled` | 人机验证 |
| `CAPTCHA_PROVIDER` | `security.captcha_provider` | `turnstile` \| `recaptcha` \| `hcaptcha` \| `none` |
| `CAPTCHA_SITE_KEY` / `CAPTCHA_SECRET_KEY` | `security.captcha_*` | 站点密钥 |

完整列表见 `backend/.env.example` 与 `backend/internal/setting/options.go` 中的 `Catalog`。

---

## 控制台

路径：`/admin/settings`

- **基础**：名称、描述、主题色、联系邮箱  
- **安全**：邮箱验证、SMTP、人机验证提供商与 Key、限速  
- **登录**：注册开关、GitHub OAuth 预留  

保存调用 `PUT /api/admin/settings`，立即写入 DB 与内存，无需重启进程。

---

## 注册链路

1. 前端 `GET /api/settings/public` 决定是否展示验证码/验证码输入框。  
2. 若 `security.email_verification`：用户请求 `send-verification-code`，服务端写入 `verification_codes` 并通过 SMTP（或 LogMailer）发出。  
3. 若 `security.captcha_enabled`：前端加载对应 Provider 脚本，提交 `captcha_token`。  
4. `POST /api/auth/register` 服务端按配置校验 captcha secret 与验证码后创建用户。

未配置 SMTP 时默认 **LogMailer**（日志打印验证码，便于开发）。  
开启 captcha 但未配置 secret 时注册返回明确错误，避免静默放行。

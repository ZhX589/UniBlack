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

当前代码未配置 SMTP 时默认 **LogMailer**（日志打印随机验证码，便于开发）；这属于旧实现行为，不满足 Phase 13 的最终目标。
当前代码开启 captcha 且配置 provider 后会调用对应第三方 siteverify；这同样属于旧实现行为，不满足 Phase 13 的“仅演示 captcha”目标。

Phase 13 实施完成前，不应把当前配置行为描述为“生产就绪的人机验证”或“生产缺 SMTP 时强制失败”。

---

## Phase 13 目标验证边界（待实施）

> 当前代码仍会按 `CAPTCHA_PROVIDER` 调用 Turnstile、reCAPTCHA 或 hCaptcha。本节记录已确认的后续设计，不代表这些行为已经变更。

### 演示人机验证

Phase 13 将把项目运行行为调整为内置的演示验证卡：用户确认“我不是自动程序”后由 UniBlack 服务端签发短期、单次、绑定用途的演示令牌。项目本身将**不加载第三方 captcha 脚本，也不请求第三方验证端点**。

控制台仍保留以下配置项，作为部署者未来接入真实 Provider 的配置契约：

- `security.captcha_enabled`
- `security.captcha_provider`
- `security.captcha_site_key`
- `security.captcha_secret_key`

在演示模式下，控制台必须明确显示“仅保存接入配置，本项目运行演示验证”，不得把保存 Site Key/Secret 误称为真实校验已生效。

### 邮箱验证码

Phase 13 将按环境区分：

| 环境 | 验证码行为 |
| --- | --- |
| `APP_ENV=development` | 仅接受固定 `123456`，不发送邮件，也不生成随机真实码 |
| 非开发环境 | 必须配置 SMTP；生成随机、单次使用、10 分钟有效验证码；缺少 SMTP 时明确失败 |

验证码用途将区分 `register`、`submission`、`appeal`，并分别限流；不同用途的验证码不可互用。

完整对象/事件/证据、默认发布、处罚与导出设计见 `docs/compose/specs/2026-07-20-subject-event-governance-design.md`。

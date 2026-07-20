-- Phase 11: Admin Console & Enhanced Registration

-- System settings table
CREATE TABLE IF NOT EXISTS system_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(100) UNIQUE NOT NULL,
    value JSONB NOT NULL DEFAULT '{}',
    description TEXT,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Access lists table (whitelist/blacklist)
CREATE TABLE IF NOT EXISTS access_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(20) NOT NULL CHECK (type IN ('whitelist', 'blacklist')),
    target VARCHAR(50) NOT NULL CHECK (target IN ('ip', 'email', 'username')),
    value VARCHAR(255) NOT NULL,
    reason TEXT,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(type, target, value)
);

CREATE INDEX idx_access_lists_type ON access_lists(type);
CREATE INDEX idx_access_lists_target ON access_lists(target);
CREATE INDEX idx_access_lists_value ON access_lists(value);

-- Add email_verified column to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verification_token VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verification_expires_at TIMESTAMP WITH TIME ZONE;

-- Default system settings
INSERT INTO system_settings (key, value, description) VALUES
    ('site.name', '"UniBlack"', '项目名称'),
    ('site.description', '"一个可复用的通用云黑系统"', '项目描述'),
    ('site.theme_color', '"#3B82F6"', '主题色'),
    ('site.logo_url', '""', 'Logo URL'),
    ('site.contact_email', '""', '联系邮箱'),
    ('security.email_verification', 'false', '邮箱验证开关'),
    ('security.smtp_host', '""', 'SMTP 服务器'),
    ('security.smtp_port', '587', 'SMTP 端口'),
    ('security.smtp_username', '""', 'SMTP 用户名'),
    ('security.smtp_password', '""', 'SMTP 密码'),
    ('security.smtp_from', '""', '发件人地址'),
    ('security.captcha_enabled', 'false', '人机验证开关'),
    ('security.captcha_provider', '"turnstile"', '人机验证提供商'),
    ('security.captcha_site_key', '""', 'Captcha Site Key'),
    ('security.captcha_secret_key', '""', 'Captcha Secret Key'),
    ('security.rate_limit_public', '20', '公开API限速 (req/s)'),
    ('security.rate_limit_auth', '10', '认证API限速 (req/s)'),
    ('auth.registration_enabled', 'true', '注册开关'),
    ('auth.oauth_github_enabled', 'false', 'GitHub登录开关'),
    ('auth.oauth_github_client_id', '""', 'GitHub Client ID'),
    ('auth.oauth_github_client_secret', '""', 'GitHub Client Secret'),
    ('system.initialized', 'false', '系统是否已初始化')
ON CONFLICT (key) DO NOTHING;

-- Seed admin user (password: admin123 in dev mode)
-- This will be handled by the application on first run

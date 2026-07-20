// Package setting provides NewAPI-style system options:
// env defaults → DB override → in-memory OptionMap → console/API.
package setting

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Known option keys (stable for console + public API).
const (
	KeySiteName        = "site.name"
	KeySiteDescription = "site.description"
	KeySiteThemeColor  = "site.theme_color"
	KeySiteLogoURL     = "site.logo_url"
	KeySiteContact     = "site.contact_email"

	KeyEmailVerification = "security.email_verification"
	KeySMTPHost          = "security.smtp_host"
	KeySMTPPort          = "security.smtp_port"
	KeySMTPUsername      = "security.smtp_username"
	KeySMTPPassword      = "security.smtp_password"
	KeySMTPFrom          = "security.smtp_from"
	KeySMTPSSL           = "security.smtp_ssl"
	KeySMTPSkipVerify    = "security.smtp_insecure_skip_verify"

	KeyCaptchaEnabled  = "security.captcha_enabled"
	KeyCaptchaProvider = "security.captcha_provider"
	KeyCaptchaSiteKey  = "security.captcha_site_key"
	KeyCaptchaSecret   = "security.captcha_secret_key"

	KeyRateLimitPublic = "security.rate_limit_public"
	KeyRateLimitAuth   = "security.rate_limit_auth"

	KeyRegisterEnabled = "auth.registration_enabled"
	KeyOAuthGitHubOn   = "auth.oauth_github_enabled"
	KeyOAuthGitHubID   = "auth.oauth_github_client_id"
	KeyOAuthGitHubSec  = "auth.oauth_github_client_secret"

	KeySystemInitialized = "system.initialized"
)

// OptionMeta describes a setting for admin console (rich NewAPI-like options).
type OptionMeta struct {
	Key         string   `json:"key"`
	Category    string   `json:"category"`
	Type        string   `json:"type"` // string | bool | number | secret | select
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Secret      bool     `json:"secret"`
	Options     []string `json:"options,omitempty"`
	Public      bool     `json:"public"`
}

// Catalog is the full option schema for console rendering.
var Catalog = []OptionMeta{
	{Key: KeySiteName, Category: "site", Type: "string", Label: "项目名称", Description: "显示名称（可自定义）", Public: true},
	{Key: KeySiteDescription, Category: "site", Type: "string", Label: "项目描述", Public: true},
	{Key: KeySiteThemeColor, Category: "site", Type: "string", Label: "主题色", Public: true},
	{Key: KeySiteLogoURL, Category: "site", Type: "string", Label: "Logo URL", Public: true},
	{Key: KeySiteContact, Category: "site", Type: "string", Label: "联系邮箱", Public: true},

	{Key: KeyEmailVerification, Category: "security", Type: "bool", Label: "邮箱验证", Description: "注册时要求邮箱验证码", Public: true},
	{Key: KeySMTPHost, Category: "security", Type: "string", Label: "SMTP 服务器"},
	{Key: KeySMTPPort, Category: "security", Type: "number", Label: "SMTP 端口"},
	{Key: KeySMTPUsername, Category: "security", Type: "string", Label: "SMTP 用户名"},
	{Key: KeySMTPPassword, Category: "security", Type: "secret", Label: "SMTP 密码/Token", Secret: true},
	{Key: KeySMTPFrom, Category: "security", Type: "string", Label: "发件人地址"},
	{Key: KeySMTPSSL, Category: "security", Type: "bool", Label: "SMTP SSL", Description: "隐式 TLS（常见 465）"},
	{Key: KeySMTPSkipVerify, Category: "security", Type: "bool", Label: "跳过 TLS 证书校验"},

	{Key: KeyCaptchaEnabled, Category: "security", Type: "bool", Label: "人机验证", Public: true},
	{Key: KeyCaptchaProvider, Category: "security", Type: "select", Label: "人机验证提供商", Options: []string{"turnstile", "recaptcha", "hcaptcha", "none"}, Public: true},
	{Key: KeyCaptchaSiteKey, Category: "security", Type: "string", Label: "Captcha Site Key", Public: true},
	{Key: KeyCaptchaSecret, Category: "security", Type: "secret", Label: "Captcha Secret Key", Secret: true},

	{Key: KeyRateLimitPublic, Category: "security", Type: "number", Label: "公开 API 限速 (req/s)"},
	{Key: KeyRateLimitAuth, Category: "security", Type: "number", Label: "认证 API 限速 (req/s)"},

	{Key: KeyRegisterEnabled, Category: "auth", Type: "bool", Label: "开放注册", Public: true},
	{Key: KeyOAuthGitHubOn, Category: "auth", Type: "bool", Label: "GitHub 登录", Public: true},
	{Key: KeyOAuthGitHubID, Category: "auth", Type: "string", Label: "GitHub Client ID", Public: true},
	{Key: KeyOAuthGitHubSec, Category: "auth", Type: "secret", Label: "GitHub Client Secret", Secret: true},

	{Key: KeySystemInitialized, Category: "system", Type: "bool", Label: "系统已初始化"},
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}

// DefaultMap builds env-overridable defaults (NewAPI InitOptionMap).
// Values are JSON-encoded scalars stored the same way as DB system_settings.value.
func DefaultMap() map[string]string {
	return map[string]string{
		KeySiteName:          mustJSON(envString("SITE_NAME", "UniBlack")),
		KeySiteDescription:   mustJSON(envString("SITE_DESCRIPTION", "一个可复用的通用云黑系统")),
		KeySiteThemeColor:    mustJSON(envString("SITE_THEME_COLOR", "#3B82F6")),
		KeySiteLogoURL:       mustJSON(envString("SITE_LOGO_URL", "")),
		KeySiteContact:       mustJSON(envString("SITE_CONTACT_EMAIL", "")),
		KeyEmailVerification: mustJSON(envBool("EMAIL_VERIFICATION_ENABLED", false)),
		KeySMTPHost:          mustJSON(envString("SMTP_HOST", "")),
		KeySMTPPort:          mustJSON(envInt("SMTP_PORT", 587)),
		KeySMTPUsername:      mustJSON(envString("SMTP_USERNAME", "")),
		KeySMTPPassword:      mustJSON(envString("SMTP_PASSWORD", "")),
		KeySMTPFrom:          mustJSON(envString("SMTP_FROM", "")),
		KeySMTPSSL:           mustJSON(envBool("SMTP_SSL_ENABLED", false)),
		KeySMTPSkipVerify:    mustJSON(envBool("SMTP_INSECURE_SKIP_VERIFY", false)),
		KeyCaptchaEnabled:    mustJSON(envBool("CAPTCHA_ENABLED", false)),
		KeyCaptchaProvider:   mustJSON(envString("CAPTCHA_PROVIDER", "turnstile")),
		KeyCaptchaSiteKey:    mustJSON(envString("CAPTCHA_SITE_KEY", "")),
		KeyCaptchaSecret:     mustJSON(envString("CAPTCHA_SECRET_KEY", "")),
		KeyRateLimitPublic:   mustJSON(envInt("RATE_LIMIT_PUBLIC", 20)),
		KeyRateLimitAuth:     mustJSON(envInt("RATE_LIMIT_AUTH", 10)),
		KeyRegisterEnabled:   mustJSON(envBool("REGISTER_ENABLED", true)),
		KeyOAuthGitHubOn:     mustJSON(envBool("OAUTH_GITHUB_ENABLED", false)),
		KeyOAuthGitHubID:     mustJSON(envString("OAUTH_GITHUB_CLIENT_ID", "")),
		KeyOAuthGitHubSec:    mustJSON(envString("OAUTH_GITHUB_CLIENT_SECRET", "")),
		KeySystemInitialized: mustJSON(envBool("SYSTEM_INITIALIZED", false)),
	}
}

func envString(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// IsSecret reports whether key stores a secret.
func IsSecret(key string) bool {
	for _, m := range Catalog {
		if m.Key == key {
			return m.Secret
		}
	}
	return strings.Contains(key, "password") || strings.Contains(key, "secret") || strings.HasSuffix(key, "_token")
}

// IsPublic reports whether key is safe for public settings API.
func IsPublic(key string) bool {
	for _, m := range Catalog {
		if m.Key == key {
			return m.Public
		}
	}
	return false
}

// Meta returns catalog entry for key.
func Meta(key string) *OptionMeta {
	for i := range Catalog {
		if Catalog[i].Key == key {
			return &Catalog[i]
		}
	}
	return nil
}

// Cache is an in-memory OptionMap (NewAPI OptionMapRWMutex pattern).
type Cache struct {
	mu   sync.RWMutex
	data map[string]string
}

// NewCache starts from env defaults.
func NewCache() *Cache {
	return &Cache{data: DefaultMap()}
}

func (c *Cache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.data[key]
	return v, ok
}

func (c *Cache) Set(key, jsonValue string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data == nil {
		c.data = make(map[string]string)
	}
	c.data[key] = jsonValue
}

func (c *Cache) Snapshot() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]string, len(c.data))
	for k, v := range c.data {
		out[k] = v
	}
	return out
}

// Merge overlays DB values onto the cache.
func (c *Cache) Merge(fromDB map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data == nil {
		c.data = DefaultMap()
	}
	for k, v := range fromDB {
		c.data[k] = v
	}
}

// Decode unmarshals a cached JSON value into dest.
func (c *Cache) Decode(key string, dest any) error {
	raw, ok := c.Get(key)
	if !ok {
		return ErrNotFound
	}
	return json.Unmarshal([]byte(raw), dest)
}

// ErrNotFound when key missing from cache.
var ErrNotFound = errNotFound("option not found")

type errNotFound string

func (e errNotFound) Error() string { return string(e) }

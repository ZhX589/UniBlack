package captcha

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid captcha token")
	ErrNotConfigured = errors.New("captcha not configured")
)

// Provider verifies human-verification tokens from third-party services.
type Provider interface {
	Name() string
	Verify(ctx context.Context, token, remoteIP string) error
}

// Config holds captcha configuration (secret never returned to public clients).
type Config struct {
	Enabled  bool
	Provider string // turnstile | recaptcha | hcaptcha | none
	SiteKey  string
	Secret   string
}

// NewProvider builds a provider from config. When disabled, returns Noop.
func NewProvider(cfg Config) Provider {
	if !cfg.Enabled || cfg.Provider == "" || cfg.Provider == "none" {
		return Noop{}
	}
	switch strings.ToLower(cfg.Provider) {
	case "turnstile":
		return &Turnstile{Secret: cfg.Secret}
	case "recaptcha", "recaptcha_v2", "recaptcha_v3":
		return &Recaptcha{Secret: cfg.Secret}
	case "hcaptcha":
		return &HCaptcha{Secret: cfg.Secret}
	default:
		return Noop{}
	}
}

// Noop always succeeds (used when captcha is disabled).
type Noop struct{}

func (Noop) Name() string { return "none" }
func (Noop) Verify(context.Context, string, string) error { return nil }

// Turnstile verifies Cloudflare Turnstile tokens.
type Turnstile struct {
	Secret string
	Client *http.Client
}

func (t *Turnstile) Name() string { return "turnstile" }

func (t *Turnstile) Verify(ctx context.Context, token, remoteIP string) error {
	if t.Secret == "" {
		return ErrNotConfigured
	}
	if token == "" {
		return ErrInvalidToken
	}
	form := url.Values{}
	form.Set("secret", t.Secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	return postSiteverify(ctx, clientOr(t.Client), "https://challenges.cloudflare.com/turnstile/v0/siteverify", form)
}

// Recaptcha verifies Google reCAPTCHA v2/v3 tokens.
type Recaptcha struct {
	Secret string
	Client *http.Client
}

func (r *Recaptcha) Name() string { return "recaptcha" }

func (r *Recaptcha) Verify(ctx context.Context, token, remoteIP string) error {
	if r.Secret == "" {
		return ErrNotConfigured
	}
	if token == "" {
		return ErrInvalidToken
	}
	form := url.Values{}
	form.Set("secret", r.Secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	return postSiteverify(ctx, clientOr(r.Client), "https://www.google.com/recaptcha/api/siteverify", form)
}

// HCaptcha verifies hCaptcha tokens.
type HCaptcha struct {
	Secret string
	Client *http.Client
}

func (h *HCaptcha) Name() string { return "hcaptcha" }

func (h *HCaptcha) Verify(ctx context.Context, token, remoteIP string) error {
	if h.Secret == "" {
		return ErrNotConfigured
	}
	if token == "" {
		return ErrInvalidToken
	}
	form := url.Values{}
	form.Set("secret", h.Secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	return postSiteverify(ctx, clientOr(h.Client), "https://hcaptcha.com/siteverify", form)
}

type siteverifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

func clientOr(c *http.Client) *http.Client {
	if c != nil {
		return c
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func postSiteverify(ctx context.Context, client *http.Client, endpoint string, form url.Values) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("captcha verify request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	var out siteverifyResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("captcha verify decode: %w", err)
	}
	if !out.Success {
		return ErrInvalidToken
	}
	return nil
}

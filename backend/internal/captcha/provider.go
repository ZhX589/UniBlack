package captcha

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidToken  = errors.New("invalid captcha token")
	ErrNotConfigured = errors.New("captcha not configured")
)

// Provider verifies human-verification tokens.
// UniBlack ships only the built-in Demo provider at runtime.
// Third-party provider keys may still be stored as configuration metadata.
type Provider interface {
	Name() string
	Verify(ctx context.Context, token, remoteIP string) error
}

// Config holds captcha configuration (secret never returned to public clients).
type Config struct {
	Enabled  bool
	Provider string // always treated as demo at runtime
	SiteKey  string
	Secret   string
}

// NewProvider returns the runtime captcha provider.
// When disabled, returns Noop. When enabled, always uses Demo (no third-party HTTP).
func NewProvider(cfg Config) Provider {
	if !cfg.Enabled || cfg.Provider == "" || strings.EqualFold(cfg.Provider, "none") {
		return Noop{}
	}
	// Configuration may still say turnstile/recaptcha/hcaptcha for future wiring,
	// but this project only executes the demo adapter.
	return NewDemo()
}

// Noop always succeeds (used when captcha is disabled).
type Noop struct{}

func (Noop) Name() string                                 { return "none" }
func (Noop) Verify(context.Context, string, string) error { return nil }

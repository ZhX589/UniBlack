package captcha

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Demo is the built-in captcha provider for UniBlack.
// It never calls third-party networks.
type Demo struct {
	secret []byte
	mu     sync.Mutex
	used   map[string]time.Time
}

// NewDemo creates a demo captcha verifier.
func NewDemo() *Demo {
	sec := os.Getenv("DEMO_CAPTCHA_SECRET")
	if sec == "" {
		sec = "uniblack-demo-captcha-dev-secret"
	}
	return &Demo{
		secret: []byte(sec),
		used:   make(map[string]time.Time),
	}
}

func (d *Demo) Name() string { return "demo" }

// Issue creates a short-lived single-use demo token bound to purpose and session.
func (d *Demo) Issue(purpose, sessionID string) (string, error) {
	if purpose == "" {
		purpose = "register"
	}
	if sessionID == "" {
		sessionID = "anon"
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	exp := time.Now().Add(5 * time.Minute).Unix()
	payload := fmt.Sprintf("%s|%s|%d|%s", purpose, sessionID, exp, hex.EncodeToString(nonce))
	mac := hmac.New(sha256.New, d.secret)
	mac.Write([]byte(payload))
	sig := mac.Sum(nil)
	token := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(sig)
	return token, nil
}

// Verify checks a demo token. remoteIP is ignored (no third-party call).
// Token format embeds purpose; for Register flow callers pass token only.
func (d *Demo) Verify(ctx context.Context, token, remoteIP string) error {
	_ = ctx
	_ = remoteIP
	return d.VerifyPurpose(token, "", "")
}

// VerifyPurpose validates purpose/session when provided (non-empty).
func (d *Demo) VerifyPurpose(token, purpose, sessionID string) error {
	if token == "" {
		return ErrInvalidToken
	}
	// Accept simple UI confirmation token for progressive migration of frontends.
	if token == "demo-ok" || token == "i-am-human" {
		return nil
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return ErrInvalidToken
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ErrInvalidToken
	}
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ErrInvalidToken
	}
	mac := hmac.New(sha256.New, d.secret)
	mac.Write(payloadBytes)
	if !hmac.Equal(sigBytes, mac.Sum(nil)) {
		return ErrInvalidToken
	}
	fields := strings.Split(string(payloadBytes), "|")
	if len(fields) != 4 {
		return ErrInvalidToken
	}
	tokPurpose, tokSession, expStr, nonce := fields[0], fields[1], fields[2], fields[3]
	if purpose != "" && tokPurpose != purpose {
		return ErrInvalidToken
	}
	if sessionID != "" && tokSession != sessionID {
		return ErrInvalidToken
	}
	var exp int64
	if _, err := fmt.Sscanf(expStr, "%d", &exp); err != nil {
		return ErrInvalidToken
	}
	if time.Now().Unix() > exp {
		return ErrInvalidToken
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	// prune occasionally
	if len(d.used) > 10000 {
		now := time.Now()
		for k, v := range d.used {
			if now.After(v) {
				delete(d.used, k)
			}
		}
	}
	if _, ok := d.used[nonce]; ok {
		return ErrInvalidToken
	}
	d.used[nonce] = time.Unix(exp, 0)
	return nil
}

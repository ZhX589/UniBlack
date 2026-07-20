package domain

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"
)

// Crockford Base32 alphabet used by ULID (excludes I, L, O, U).
const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// GeneratePublicID returns a new subject public ID: UBS_<ULID>.
func GeneratePublicID() (string, error) {
	ulid, err := newULID()
	if err != nil {
		return "", err
	}
	return "UBS_" + ulid, nil
}

func newULID() (string, error) {
	var entropy [10]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return "", err
	}
	ms := uint64(time.Now().UTC().UnixMilli())

	// 16 bytes: 6 timestamp + 10 entropy
	var raw [16]byte
	raw[0] = byte(ms >> 40)
	raw[1] = byte(ms >> 32)
	raw[2] = byte(ms >> 24)
	raw[3] = byte(ms >> 16)
	raw[4] = byte(ms >> 8)
	raw[5] = byte(ms)
	copy(raw[6:], entropy[:])

	// Encode 128 bits as 26 Crockford Base32 characters.
	out := make([]byte, 26)
	// Process as bit stream from MSB of raw[0].
	var acc uint64
	bitsLeft := 0
	ri := 0
	for oi := 0; oi < 26; oi++ {
		for bitsLeft < 5 {
			if ri < len(raw) {
				acc = (acc << 8) | uint64(raw[ri])
				ri++
				bitsLeft += 8
			} else {
				acc <<= 5 - bitsLeft
				bitsLeft = 5
			}
		}
		bitsLeft -= 5
		out[oi] = crockford[(acc>>bitsLeft)&31]
	}
	return string(out), nil
}

// AccountInput is used when creating accounts for a subject.
type AccountInput struct {
	Platform         string
	PlatformLabel    string
	AccountType      string
	Username         string
	AccountID        string
	CustomAttributes map[string]interface{}
	IsPrimary        bool
}

// AccountDedupKey returns the uniqueness key: prefer platform:account_id.
func AccountDedupKey(platform, username, accountID string) string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if strings.TrimSpace(accountID) != "" {
		return platform + ":" + strings.TrimSpace(accountID)
	}
	return platform + ":" + strings.TrimSpace(username)
}

// ResolveDisplayName picks display name from input or first account username.
func ResolveDisplayName(displayName string, accounts []AccountInput) (string, error) {
	name := strings.TrimSpace(displayName)
	if name != "" {
		return name, nil
	}
	for _, a := range accounts {
		if u := strings.TrimSpace(a.Username); u != "" {
			return u, nil
		}
	}
	return "", fmt.Errorf("display_name required when no account username is provided")
}

// IsValidPublicID reports whether s matches UBS_<ULID> shape (30 chars).
// Historical backfilled IDs may be longer (UBS_ + uuid hex) and are not matched here.
func IsValidPublicID(s string) bool {
	if !strings.HasPrefix(s, "UBS_") || len(s) != 30 {
		return false
	}
	body := s[4:]
	for i := 0; i < len(body); i++ {
		c := body[i]
		if (c >= '0' && c <= '9') || (c >= 'A' && c <= 'H') || (c >= 'J' && c <= 'K') ||
			(c >= 'M' && c <= 'N') || (c >= 'P' && c <= 'T') || (c >= 'V' && c <= 'Z') {
			continue
		}
		return false
	}
	return true
}

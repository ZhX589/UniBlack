package domain

import (
	"errors"
	"net/url"
	"strings"
)

// ValidateAbsoluteHTTPURL requires an absolute http(s) URL with a host.
func ValidateAbsoluteHTTPURL(raw string) error {
	u, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return errors.New("absolute http(s) URL required")
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return nil
	default:
		return errors.New("absolute http(s) URL required")
	}
}

// ValidateLinkEvidence requires a title and absolute http(s) URL.
func ValidateLinkEvidence(title, rawURL string) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("link evidence requires title and absolute http(s) URL")
	}
	if err := ValidateAbsoluteHTTPURL(rawURL); err != nil {
		return errors.New("link evidence requires title and absolute http(s) URL")
	}
	return nil
}

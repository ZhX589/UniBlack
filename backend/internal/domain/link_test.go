package domain

import "testing"

func TestValidateAbsoluteHTTPURL(t *testing.T) {
	if err := ValidateAbsoluteHTTPURL("https://example.test/report"); err != nil {
		t.Fatalf("valid URL rejected: %v", err)
	}
	if err := ValidateAbsoluteHTTPURL("HTTPS://Example.TEST/path"); err != nil {
		t.Fatalf("uppercase scheme rejected: %v", err)
	}
	for _, raw := range []string{"", "/relative", "ftp://example.test", "https://", "http://?"} {
		if err := ValidateAbsoluteHTTPURL(raw); err == nil {
			t.Fatalf("invalid URL %q accepted", raw)
		}
	}
}

func TestValidateLinkEvidence(t *testing.T) {
	if err := ValidateLinkEvidence("report", "https://example.test/a"); err != nil {
		t.Fatalf("valid link rejected: %v", err)
	}
	if err := ValidateLinkEvidence("", "https://example.test/a"); err == nil {
		t.Fatal("empty title accepted")
	}
}

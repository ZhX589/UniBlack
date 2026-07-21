package service

import (
	"strings"
	"testing"
)

func TestValidateTextEvidence(t *testing.T) {
	if err := validateTextEvidence("valid text"); err != nil {
		t.Fatalf("valid text rejected: %v", err)
	}
	if err := validateTextEvidence(""); err == nil {
		t.Fatal("empty text accepted")
	}
	if err := validateTextEvidence(strings.Repeat("x", MaxTextEvidenceBytes+1)); err == nil {
		t.Fatal("oversized text accepted")
	}
	if err := validateTextEvidence(string([]byte{0xff})); err == nil {
		t.Fatal("invalid UTF-8 accepted")
	}
}

package captcha

import (
	"context"
	"testing"
)

func TestDemoIssueAndVerify(t *testing.T) {
	d := NewDemo()
	token, err := d.Issue("register", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.VerifyPurpose(token, "register", "session-1"); err != nil {
		t.Fatalf("verify: %v", err)
	}
	// single-use
	if err := d.VerifyPurpose(token, "register", "session-1"); err == nil {
		t.Fatal("expected second use to fail")
	}
}

func TestDemoRejectsWrongPurpose(t *testing.T) {
	d := NewDemo()
	token, err := d.Issue("register", "s1")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.VerifyPurpose(token, "submission", "s1"); err == nil {
		t.Fatal("expected purpose mismatch")
	}
}

func TestNewProviderUsesDemo(t *testing.T) {
	p := NewProvider(Config{Enabled: true, Provider: "turnstile", Secret: "x"})
	if p.Name() != "demo" {
		t.Fatalf("name=%s", p.Name())
	}
	if err := p.Verify(context.Background(), "i-am-human", ""); err != nil {
		t.Fatal(err)
	}
}

func TestNewProviderNoopWhenDisabled(t *testing.T) {
	p := NewProvider(Config{Enabled: false})
	if p.Name() != "none" {
		t.Fatalf("name=%s", p.Name())
	}
}

package domain

import (
	"regexp"
	"testing"
)

func TestGeneratePublicIDShape(t *testing.T) {
	id, err := GeneratePublicID()
	if err != nil {
		t.Fatalf("GeneratePublicID: %v", err)
	}
	re := regexp.MustCompile(`^UBS_[0-9A-HJKMNP-TV-Z]{26}$`)
	if !re.MatchString(id) {
		t.Fatalf("public id shape invalid: %q", id)
	}
	if !IsValidPublicID(id) {
		t.Fatalf("IsValidPublicID rejected generated id: %q", id)
	}
}

func TestAccountDedupKeyPrefersAccountID(t *testing.T) {
	got := AccountDedupKey("telegram", "Alice", "123")
	if got != "telegram:123" {
		t.Fatalf("got %q", got)
	}
	got = AccountDedupKey("Telegram", "Bob", "")
	if got != "telegram:bob" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveDisplayNameFromFirstUsername(t *testing.T) {
	name, err := ResolveDisplayName("", []AccountInput{{Platform: "qq", Username: "alice"}})
	if err != nil {
		t.Fatal(err)
	}
	if name != "alice" {
		t.Fatalf("got %q", name)
	}
	_, err = ResolveDisplayName("", []AccountInput{{Platform: "qq", AccountID: "1"}})
	if err == nil {
		t.Fatal("expected error when no username and no display name")
	}
}

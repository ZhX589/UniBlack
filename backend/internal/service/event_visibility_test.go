package service

import (
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

func TestEventVisibleTo(t *testing.T) {
	owner, empty := "owner", ""
	for _, status := range []string{"published", "corrected", "draft", "hidden", "withdrawn", "pending", "unknown", ""} {
		for _, requester := range []struct {
			name    string
			id      string
			roles   []string
			private bool
		}{
			{"anonymous", "", nil, false},
			{"other", "other", []string{"user"}, false},
			{"owner", owner, nil, true},
			{"admin", "staff", []string{"user", "admin"}, true},
			{"moderator", "staff", []string{"moderator"}, true},
			{"unrelated-role", "other", []string{"reviewer"}, false},
		} {
			t.Run(status+"/"+requester.name, func(t *testing.T) {
				event := &models.Event{Status: status, SubmittedBy: &owner}
				want := status == "published" || status == "corrected" || requester.private
				if got := EventVisibleTo(event, requester.id, requester.roles); got != want {
					t.Fatalf("visible = %v, want %v", got, want)
				}
			})
		}
	}
	for _, submittedBy := range []*string{nil, &empty} {
		if EventVisibleTo(&models.Event{Status: "hidden", SubmittedBy: submittedBy}, "", nil) {
			t.Fatal("anonymous cannot own a nonpublic event")
		}
	}
	if EventVisibleTo(nil, owner, []string{"admin"}) {
		t.Fatal("nil event is not visible")
	}
}

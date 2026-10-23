package service

import (
	"context"
	"strings"
	"testing"
)

func TestPublishFileBudgetExactAndOverLimits(t *testing.T) {
	data := make([]byte, MaxPublishFileBytes+1)
	file := func(n int) PublishFileEvidenceRequest {
		return PublishFileEvidenceRequest{Filename: "proof.bin", Content: data[:n]}
	}
	for _, tc := range []struct {
		name  string
		files []PublishFileEvidenceRequest
		want  string
	}{
		{"64 files", make([]PublishFileEvidenceRequest, 64), ""},
		{"65 files", make([]PublishFileEvidenceRequest, 65), "too many"},
		{"64 MiB", []PublishFileEvidenceRequest{file(20 << 20), file(20 << 20), file(20 << 20), file(4 << 20)}, ""},
		{"64 MiB plus one", []PublishFileEvidenceRequest{file(20 << 20), file(20 << 20), file(20 << 20), file((4 << 20) + 1)}, "total evidence"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePublishFileBudget(tc.files)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v", err)
			}
		})
	}
	if err := file(MaxPublishFileBytes).Validate(1); err != nil {
		t.Fatal(err)
	}
	if err := file(MaxPublishFileBytes + 1).Validate(1); err == nil {
		t.Fatal("20MiB+1 accepted")
	}
}

func TestPublishBudgetRejectedBeforeDependencies(t *testing.T) {
	// Nil dependencies deliberately prove invalid requests cannot reach account,
	// verification, repository or storage side effects through a direct call.
	svc := &EventService{}
	_, err := svc.Publish(context.Background(), PublishSubjectRequest{FileEvidence: make([]PublishFileEvidenceRequest, 65)}, "user")
	if err == nil || !strings.Contains(err.Error(), "too many") {
		t.Fatalf("error=%v", err)
	}
}

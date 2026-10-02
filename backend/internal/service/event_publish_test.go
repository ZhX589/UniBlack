package service

import (
	"context"
	"strings"
	"testing"

	"github.com/ZhX589/UniBlack/backend/internal/repository"
)

func TestPublishValidatesWholeEvidenceBatchBeforeUpload(t *testing.T) {
	for _, tc := range []struct {
		name  string
		text  []PublishTextEvidenceRequest
		files []PublishFileEvidenceRequest
		links []PublishLinkEvidenceRequest
	}{
		{name: "later empty text", text: []PublishTextEvidenceRequest{{Text: "first valid"}, {Text: ""}}},
		{name: "later invalid UTF8", text: []PublishTextEvidenceRequest{{Text: "first valid"}, {Text: string([]byte{0xff})}}},
		{name: "later oversized text", text: []PublishTextEvidenceRequest{{Text: "first valid"}, {Text: strings.Repeat("x", MaxTextEvidenceBytes+1)}}},
		{name: "later invalid text event", text: []PublishTextEvidenceRequest{{Text: "first valid"}, {EventIndex: 1, Text: "second"}}},
		{name: "file after text", text: []PublishTextEvidenceRequest{{Text: "first valid"}}, files: []PublishFileEvidenceRequest{{Filename: "invalid.exe", Content: []byte("file")}}},
		{name: "later file", files: []PublishFileEvidenceRequest{{Filename: "valid.png", Content: []byte("file")}, {Filename: "invalid.exe", Content: []byte("file")}}},
		{name: "invalid link", text: []PublishTextEvidenceRequest{{Text: "first valid"}}, links: []PublishLinkEvidenceRequest{{Title: "report", URL: "/relative"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &recordingStorage{}
			s := NewEventService(repository.NewEventRepository(nil, store), nil, nil, nil, nil, nil)
			req := PublishSubjectRequest{
				Accounts:     []PublishAccountRequest{{Platform: "test", Username: "alice"}},
				Events:       []PublishEventRequest{{Title: "event", Details: "details"}},
				TextEvidence: tc.text, FileEvidence: tc.files, LinkEvidence: tc.links,
			}
			if _, err := s.Publish(context.Background(), req, "user-id"); err == nil {
				t.Fatal("invalid evidence accepted")
			}
			if store.uploadedKey != "" || store.deletedKey != "" {
				t.Fatalf("validation caused storage side effects: upload %q delete %q", store.uploadedKey, store.deletedKey)
			}
		})
	}
}

func TestPublishTextEvidenceRequiresEventIndex(t *testing.T) {
	request := PublishTextEvidenceRequest{EventIndex: -1, Text: "evidence"}
	if err := request.Validate(1); err == nil {
		t.Fatal("expected invalid event index to be rejected")
	}
}

func TestPublishFileEvidenceValidate(t *testing.T) {
	ok := PublishFileEvidenceRequest{
		EventIndex: 0,
		Title:      "shot",
		Filename:   "a.png",
		Content:    []byte("png-bytes"),
	}
	if err := ok.Validate(1); err != nil {
		t.Fatalf("valid file rejected: %v", err)
	}
	if err := (PublishFileEvidenceRequest{EventIndex: 2, Filename: "a.png", Content: []byte("x")}).Validate(1); err == nil {
		t.Fatal("expected bad event index")
	}
	if err := (PublishFileEvidenceRequest{EventIndex: 0, Filename: "a.exe", Content: []byte("x")}).Validate(1); err == nil {
		t.Fatal("expected rejected extension")
	}
	big := make([]byte, MaxPublishFileBytes+1)
	if err := (PublishFileEvidenceRequest{EventIndex: 0, Filename: "a.bin", Content: big}).Validate(1); err == nil {
		t.Fatal("expected oversized file rejection")
	}
	if err := (PublishFileEvidenceRequest{EventIndex: 0, Filename: "a.png", Content: nil}).Validate(1); err == nil {
		t.Fatal("expected empty content rejection")
	}
}

func TestPublishEvidenceCounters(t *testing.T) {
	textN := make([]int, 2)
	fileN := make([]int, 2)
	k1 := nextEvidenceKey("UBS_X", 0, ".txt", textN, fileN)
	k2 := nextEvidenceKey("UBS_X", 0, ".png", textN, fileN)
	k3 := nextEvidenceKey("UBS_X", 0, ".txt", textN, fileN)
	if !strings.Contains(k1, "_E001_T001.txt") {
		t.Fatalf("k1=%s", k1)
	}
	if !strings.Contains(k2, "_E001_F001.png") {
		t.Fatalf("k2=%s", k2)
	}
	if !strings.Contains(k3, "_E001_T002.txt") {
		t.Fatalf("k3=%s", k3)
	}
}

func TestPublishLinkEvidenceRequiresAbsoluteHTTPURL(t *testing.T) {
	valid := CreateEventLinkEvidenceRequest{Title: "report", Description: "source", URL: "https://example.test/evidence"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid link rejected: %v", err)
	}
	for _, raw := range []string{"", "/relative", "ftp://example.test/file"} {
		if err := (CreateEventLinkEvidenceRequest{Title: "report", URL: raw}).Validate(); err == nil {
			t.Fatalf("invalid link %q accepted", raw)
		}
	}
	if err := (PublishLinkEvidenceRequest{EventIndex: 0, Title: "report", URL: "https://example.test/a"}).Validate(1); err != nil {
		t.Fatalf("publish link rejected: %v", err)
	}
	if err := (PublishLinkEvidenceRequest{EventIndex: 3, Title: "report", URL: "https://example.test/a"}).Validate(1); err == nil {
		t.Fatal("expected invalid publish link event index")
	}
}

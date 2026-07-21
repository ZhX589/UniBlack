package service

import (
	"strings"
	"testing"
)

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

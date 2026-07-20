package service

import "testing"

func TestPublishTextEvidenceRequiresEventIndex(t *testing.T) {
	request := PublishTextEvidenceRequest{EventIndex: -1, Text: "evidence"}
	if err := request.Validate(1); err == nil {
		t.Fatal("expected invalid event index to be rejected")
	}
}

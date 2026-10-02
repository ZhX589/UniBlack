package exporter

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"testing"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

func TestArchiveEventMetadataRoundTripAndLegacyDefaults(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	note := "corrected explanation"
	event := models.Event{Title: "title", Details: "facts", Status: "corrected", Severity: 5, OccurredFrom: &from, OccurredTo: &to, CorrectionNote: &note}
	m := Manifest{SchemaVersion: 1, PublicID: "UBS_TEST", DisplayName: "test", Events: []EventManifest{eventManifest(event)}}
	decoded, _, err := readArchive(mustZip(t, m, nil))
	if err != nil {
		t.Fatal(err)
	}
	restored := importedEvent(decoded.Events[0], "actor")
	event.SubmittedBy = restored.SubmittedBy
	if !reflect.DeepEqual(event, restored) {
		t.Fatalf("metadata changed: %+v vs %+v", event, restored)
	}
	legacy := importedEvent(EventManifest{Title: "old", Details: "old"}, "actor")
	if legacy.Severity != 1 || legacy.Status != "published" {
		t.Fatalf("legacy=%+v", legacy)
	}
}

func TestArchiveAcceptsPublishedFileBudgetAndRetainsBounds(t *testing.T) {
	key := evidencePrefix("UBS_TEST") + "proof.bin"
	for _, n := range []int{17 << 20, maxArchiveEntryBytes, maxArchiveEntryBytes + 1} {
		body := bytes.Repeat([]byte{'x'}, n)
		sum := sha256.Sum256(body)
		m := Manifest{SchemaVersion: 1, PublicID: "UBS_TEST", DisplayName: "test", Events: []EventManifest{{Title: "t", Status: "published", Evidence: []EvidenceManifest{{Type: "file", FileName: key, SHA256: hex.EncodeToString(sum[:]), Title: "proof", Description: "detail"}}}}}
		decoded, _, err := readArchive(mustZip(t, m, map[string][]byte{key: body}))
		if n <= maxArchiveEntryBytes {
			if err != nil {
				t.Fatal(err)
			}
			if decoded.Events[0].Evidence[0].Title != "proof" {
				t.Fatal("metadata lost")
			}
		} else if err == nil {
			t.Fatal("oversized entry accepted")
		}
	}
}

func TestArchiveRejectsInvalidOptionalEventMetadata(t *testing.T) {
	from := time.Now()
	to := from.Add(-time.Hour)
	for _, event := range []EventManifest{{Severity: -1}, {Severity: 6}, {OccurredFrom: &from, OccurredTo: &to}} {
		m := Manifest{SchemaVersion: 1, PublicID: "UBS_TEST", DisplayName: "test", Events: []EventManifest{event}}
		if _, _, err := readArchive(mustZip(t, m, nil)); err == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
}

package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	exporter "github.com/ZhX589/UniBlack/backend/internal/export"
	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"github.com/ZhX589/UniBlack/backend/internal/storage"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func attachEvidence(t *testing.T, s *EvidenceService, kind, body string) (*models.Evidence, error) {
	t.Helper()
	if kind == "text" {
		return s.CreateEventTextEvidence(context.Background(), "event-id", "UBS_TEST", 1, 1, body, "proof", "user-id")
	}
	return s.CreateEventFileEvidence(context.Background(), "event-id", "UBS_TEST", 1, 1, strings.NewReader(body), "proof.txt", int64(len(body)), "proof", "user-id")
}

func TestAttachmentRepeatedClientNumbersPreserveExistingObjects(t *testing.T) {
	for _, kind := range []string{"text", "file"} {
		t.Run(kind, func(t *testing.T) {
			store := storage.NewLocalStorage(t.TempDir(), "https://storage.test")
			repo := &duplicateEvidenceRepository{}
			s := NewEvidenceService(repo, nil, store)
			// Also protect a pre-fix / Publish object with the same client numbers.
			legacy := storage.BuildEvidenceKey("UBS_TEST", 1, 1, ".txt")
			if _, err := store.Upload(context.Background(), legacy, strings.NewReader("legacy"), "text/plain"); err != nil {
				t.Fatal(err)
			}
			first, err := attachEvidence(t, s, kind, "first")
			if err != nil {
				t.Fatal(err)
			}
			second, err := attachEvidence(t, s, kind, "second")
			if err != nil {
				t.Fatal(err)
			}
			if *first.StorageKey == *second.StorageKey || *first.StorageKey == legacy {
				t.Fatal("client numbers reused physical identity")
			}
			metadataErr := errors.New("metadata failed")
			repo.createErr = metadataErr
			if _, err := attachEvidence(t, s, kind, "failed"); !errors.Is(err, metadataErr) {
				t.Fatalf("error = %v", err)
			}
			for key, want := range map[string]string{legacy: "legacy", *first.StorageKey: "first", *second.StorageKey: "second"} {
				rc, err := store.Open(context.Background(), key)
				if err != nil {
					t.Fatal(err)
				}
				body, err := io.ReadAll(rc)
				rc.Close()
				if err != nil || string(body) != want {
					t.Fatalf("object %s = %q, err %v", key, body, err)
				}
			}
		})
	}
}

func TestAttachmentMetadataFailureDeletesOnlyOwnObject(t *testing.T) {
	for _, kind := range []string{"text", "file"} {
		t.Run(kind, func(t *testing.T) {
			metadataErr := errors.New("metadata failed")
			store := &recordingStorage{}
			s := NewEvidenceService(&duplicateEvidenceRepository{createErr: metadataErr}, nil, store)
			if _, err := attachEvidence(t, s, kind, "failed"); !errors.Is(err, metadataErr) {
				t.Fatalf("error = %v", err)
			}
			if store.uploadedKey == "" || store.deletedKey != store.uploadedKey {
				t.Fatalf("upload %q, delete %q", store.uploadedKey, store.deletedKey)
			}
			if store.uploadedKey == storage.BuildEvidenceKey("UBS_TEST", 1, 1, ".txt") {
				t.Fatal("cleanup targets shared client-number key")
			}
		})
	}
}

func TestAttachmentConcurrentKeysAreIndependent(t *testing.T) {
	const n = 64
	keys := make(chan string, n)
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key, err := newAttachmentKey("UBS_TEST", 1, 1, ".txt")
			keys <- key
			errs <- err
		}()
	}
	wg.Wait()
	close(keys)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	seen := map[string]bool{}
	for key := range keys {
		if seen[key] {
			t.Fatalf("duplicate key %s", key)
		}
		seen[key] = true
	}
}

func TestAttachmentKeysAcceptedByArchivePreview(t *testing.T) {
	// DryRun needs no live DB. Preview still executes the real ZIP/path/hash
	// validation; the dry-run subject lookup is deliberately a reported conflict.
	db, err := gorm.Open(postgres.Open("host=localhost user=test dbname=test sslmode=disable"), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	archive := exporter.NewArchiveService(repository.NewSubjectRepository(db), nil, nil, nil)
	store := storage.NewLocalStorage(t.TempDir(), "https://storage.test")
	s := NewEvidenceService(&duplicateEvidenceRepository{}, nil, store)
	for _, kind := range []string{"text", "file"} {
		t.Run(kind, func(t *testing.T) {
			ev, err := attachEvidence(t, s, kind, "archive proof")
			if err != nil {
				t.Fatal(err)
			}
			manifest := exporter.Manifest{SchemaVersion: 1, PublicID: "UBS_TEST", DisplayName: "test", Events: []exporter.EventManifest{{Title: "event", Details: "details", Status: "published", Evidence: []exporter.EvidenceManifest{{Type: ev.Type, FileName: *ev.StorageKey, SHA256: *ev.SHA256, OriginalFilename: *ev.OriginalFilename}}}}}
			raw, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			zw := zip.NewWriter(&buf)
			w, err := zw.Create("manifest.json")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write(raw); err != nil {
				t.Fatal(err)
			}
			w, err = zw.Create(*ev.StorageKey)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte("archive proof")); err != nil {
				t.Fatal(err)
			}
			if err := zw.Close(); err != nil {
				t.Fatal(err)
			}
			preview, err := archive.PreviewImport(&buf)
			if err != nil {
				t.Fatalf("new attachment archive rejected: %v", err)
			}
			if preview.PublicID != "UBS_TEST" {
				t.Fatalf("preview = %+v", preview)
			}
		})
	}
}

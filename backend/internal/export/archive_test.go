package exporter

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func TestImportManifestRejectsExistingPublicID(t *testing.T) {
	manifest := Manifest{SchemaVersion: 1, PublicID: "UBS_01ABCDEFGHJKMNPQRSTVWXYZ", DisplayName: "archive test"}
	if err := validateManifest(manifest); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
	if err := validateManifest(Manifest{SchemaVersion: 1}); err == nil {
		t.Fatal("manifest without public ID accepted")
	}
}

func TestReadArchiveRejectsPathEscapeAndMissingHash(t *testing.T) {
	publicID := "UBS_01ABCDEFGHJKMNPQRSTVWXYZ"
	body := []byte("hello")
	sum := sha256.Sum256(body)
	manifest := Manifest{
		SchemaVersion: 1,
		PublicID:      publicID,
		DisplayName:   "t",
		Events: []EventManifest{{
			Title:   "e",
			Details: "d",
			Status:  "published",
			Evidence: []EvidenceManifest{{
				FileName: "../escape.txt",
				SHA256:   hex.EncodeToString(sum[:]),
				Type:     "file",
			}},
		}},
	}
	raw, _ := json.Marshal(manifest)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("manifest.json")
	_, _ = w.Write(raw)
	w, _ = zw.Create("../escape.txt")
	_, _ = w.Write(body)
	_ = zw.Close()
	if _, _, err := readArchive(&buf); err == nil {
		t.Fatal("expected path escape rejection")
	}
}

func TestReadArchiveRequiresEvidenceHash(t *testing.T) {
	publicID := "UBS_01ABCDEFGHJKMNPQRSTVWXYZ"
	key := evidencePrefix(publicID) + "UBS_01ABCDEFGHJKMNPQRSTVWXYZ_E001_F001.bin"
	body := []byte("hello")
	manifest := Manifest{
		SchemaVersion: 1,
		PublicID:      publicID,
		DisplayName:   "t",
		Events: []EventManifest{{
			Title:   "e",
			Details: "d",
			Status:  "published",
			Evidence: []EvidenceManifest{{
				FileName: key,
				Type:     "file",
			}},
		}},
	}
	raw, _ := json.Marshal(manifest)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("manifest.json")
	_, _ = w.Write(raw)
	w, _ = zw.Create(key)
	_, _ = w.Write(body)
	_ = zw.Close()
	if _, _, err := readArchive(&buf); err == nil || !strings.Contains(err.Error(), "missing evidence hash") {
		t.Fatalf("expected missing hash error, got %v", err)
	}
}

func TestValidateEvidenceNameNamespace(t *testing.T) {
	if err := validateEvidenceName("UBS_X", "subjects/UBS_X/evidence/a.txt"); err != nil {
		t.Fatal(err)
	}
	if err := validateEvidenceName("UBS_X", "subjects/UBS_Y/evidence/a.txt"); err == nil {
		t.Fatal("expected foreign namespace rejection")
	}
}

func TestReadArchivePreservesLinkMetadataWithoutArchiveBody(t *testing.T) {
	manifest := Manifest{SchemaVersion: 1, PublicID: "UBS_01ABCDEFGHJKMNPQRSTVWXYZ", DisplayName: "t", Events: []EventManifest{{Title: "e", Details: "d", Status: "published", Evidence: []EvidenceManifest{{Type: "link", Title: "report", Description: "original report", URL: "https://example.test/report"}}}}}
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
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	got, files, err := readArchive(&buf)
	if err != nil {
		t.Fatal(err)
	}
	link := got.Events[0].Evidence[0]
	if link.Title != "report" || link.Description != "original report" || link.URL != "https://example.test/report" {
		t.Fatalf("link metadata changed: %#v", link)
	}
	if link.SHA256 != "" || link.FileName != "" {
		t.Fatalf("link retained body metadata: %#v", link)
	}
	if len(files) != 1 {
		t.Fatalf("link created unexpected archive body: %v", files)
	}
}

func TestReadArchiveRejectsLinkBodyMetadataAndUnreferencedFiles(t *testing.T) {
	base := Manifest{
		SchemaVersion: 1,
		PublicID:      "UBS_01ABCDEFGHJKMNPQRSTVWXYZ",
		DisplayName:   "t",
		Events: []EventManifest{{
			Title:   "e",
			Details: "d",
			Status:  "published",
			Evidence: []EvidenceManifest{{
				Type:  "link",
				Title: "report",
				URL:   "https://example.test/report",
			}},
		}},
	}

	withHash := base
	withHash.Events = append([]EventManifest(nil), base.Events...)
	withHash.Events[0].Evidence = append([]EvidenceManifest(nil), base.Events[0].Evidence...)
	withHash.Events[0].Evidence[0].SHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, _, err := readArchive(mustZip(t, withHash, nil)); err == nil {
		t.Fatal("expected link hash rejection")
	}

	if _, _, err := readArchive(mustZip(t, base, map[string][]byte{
		"subjects/UBS_01ABCDEFGHJKMNPQRSTVWXYZ/evidence/orphan.bin": []byte("x"),
	})); err == nil {
		t.Fatal("expected unreferenced body rejection")
	}

	invalidURL := base
	invalidURL.Events = append([]EventManifest(nil), base.Events...)
	invalidURL.Events[0].Evidence = append([]EvidenceManifest(nil), base.Events[0].Evidence...)
	invalidURL.Events[0].Evidence[0].URL = "https://"
	if _, _, err := readArchive(mustZip(t, invalidURL, nil)); err == nil {
		t.Fatal("expected invalid link URL rejection")
	}
}

func mustZip(t *testing.T, manifest Manifest, extras map[string][]byte) *bytes.Buffer {
	t.Helper()
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
	for name, content := range extras {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf
}

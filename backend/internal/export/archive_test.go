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

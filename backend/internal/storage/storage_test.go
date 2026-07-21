package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStorageRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir, "http://localhost/uploads")
	ctx := context.Background()
	key := "subjects/UBS_TEST/evidence/UBS_TEST_E001_T001.txt"
	body := []byte("hello evidence")
	url, err := s.Upload(ctx, key, bytes.NewReader(body), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(url, key) {
		t.Fatalf("url=%s", url)
	}
	rc, err := s.Open(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(got, body) {
		t.Fatalf("got %q", got)
	}
	// survives "restart" by re-opening same path
	path, err := s.Path(key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected deleted, err=%v", err)
	}
}

func TestPathEscapeDenied(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir, "http://x")
	_, err := s.Upload(context.Background(), "../outside.txt", strings.NewReader("x"), "text/plain")
	if err == nil {
		t.Fatal("expected path escape error")
	}
	// ensure nothing written outside
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "outside.txt")); err == nil {
		t.Fatal("escaped write")
	}
}

func TestBuildEvidenceKey(t *testing.T) {
	got := BuildEvidenceKey("UBS_01ABC", 1, 2, ".txt")
	want := "subjects/UBS_01ABC/evidence/UBS_01ABC_E001_T002.txt"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

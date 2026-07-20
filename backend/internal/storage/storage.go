package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Storage defines the interface for file storage.
type Storage interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error)
	Delete(ctx context.Context, key string) error
	GetURL(key string) string
	// Open reads a stored object by key.
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	// Path returns the absolute filesystem path for a key (local only).
	Path(key string) (string, error)
}

// LocalStorage implements Storage using local filesystem.
type LocalStorage struct {
	basePath string
	baseURL  string
}

// NewLocalStorage creates a new local storage under basePath.
func NewLocalStorage(basePath, baseURL string) *LocalStorage {
	return &LocalStorage{
		basePath: basePath,
		baseURL:  strings.TrimRight(baseURL, "/"),
	}
}

func (s *LocalStorage) resolve(key string) (string, error) {
	if key == "" || strings.Contains(key, "\x00") {
		return "", fmt.Errorf("invalid storage key")
	}
	// Reject absolute and parent-segment keys before join.
	if filepath.IsAbs(key) || strings.HasPrefix(key, "/") || strings.HasPrefix(key, `\`) {
		return "", fmt.Errorf("invalid storage key")
	}
	parts := strings.Split(filepath.ToSlash(key), "/")
	for _, p := range parts {
		if p == ".." || p == "." {
			return "", fmt.Errorf("path escape denied")
		}
	}
	clean := filepath.Clean(filepath.FromSlash(strings.Join(parts, "/")))
	if clean == "." || clean == "" || strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("invalid storage key")
	}
	full := filepath.Join(s.basePath, clean)
	baseAbs, err := filepath.Abs(s.basePath)
	if err != nil {
		return "", err
	}
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	sep := string(os.PathSeparator)
	if fullAbs != baseAbs && !strings.HasPrefix(fullAbs, baseAbs+sep) {
		return "", fmt.Errorf("path escape denied")
	}
	return fullAbs, nil
}

func (s *LocalStorage) Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	_ = ctx
	_ = contentType
	full, err := s.resolve(key)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}
	tmp := full + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, reader); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, full); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return s.GetURL(key), nil
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	_ = ctx
	full, err := s.resolve(key)
	if err != nil {
		return err
	}
	err = os.Remove(full)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *LocalStorage) GetURL(key string) string {
	return s.baseURL + "/" + strings.TrimPrefix(key, "/")
}

func (s *LocalStorage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	_ = ctx
	full, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	return os.Open(full)
}

func (s *LocalStorage) Path(key string) (string, error) {
	return s.resolve(key)
}

// BuildEvidenceKey builds a stable archive key for subject evidence.
// Example: subjects/UBS_xxx/evidence/UBS_xxx_E001_T002.txt
func BuildEvidenceKey(publicID string, eventNumber, evidenceNumber int, ext string) string {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	kind := "F"
	if strings.EqualFold(ext, ".txt") {
		kind = "T"
	}
	return fmt.Sprintf("subjects/%s/evidence/%s_E%03d_%s%03d%s",
		publicID, publicID, eventNumber, kind, evidenceNumber, ext)
}

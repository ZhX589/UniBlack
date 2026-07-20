package storage

import (
	"context"
	"io"
)

// Storage defines the interface for file storage
type Storage interface {
	// Upload uploads a file and returns the URL
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error)

	// Delete deletes a file
	Delete(ctx context.Context, key string) error

	// GetURL returns the URL for a file
	GetURL(key string) string
}

// LocalStorage implements Storage using local filesystem
type LocalStorage struct {
	basePath string
	baseURL  string
}

// NewLocalStorage creates a new local storage
func NewLocalStorage(basePath, baseURL string) *LocalStorage {
	return &LocalStorage{
		basePath: basePath,
		baseURL:  baseURL,
	}
}

func (s *LocalStorage) Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	// Implementation would save to local filesystem
	// For now, return a placeholder URL
	return s.baseURL + "/" + key, nil
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	// Implementation would delete from local filesystem
	return nil
}

func (s *LocalStorage) GetURL(key string) string {
	return s.baseURL + "/" + key
}

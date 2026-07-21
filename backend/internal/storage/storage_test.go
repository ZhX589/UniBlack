package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeS3Client struct {
	bucketExists   bool
	bucketErr      error
	makeBucketErr  error
	putErr         error
	getErr         error
	deleteErr      error
	objects        map[string][]byte
	madeBucket     string
	putBucket      string
	putKey         string
	putContentType string
	deletedKey     string
}

func (c *fakeS3Client) BucketExists(_ context.Context, _ string) (bool, error) {
	return c.bucketExists, c.bucketErr
}

func (c *fakeS3Client) MakeBucket(_ context.Context, bucket string) error {
	c.madeBucket = bucket
	return c.makeBucketErr
}

func (c *fakeS3Client) PutObject(_ context.Context, bucket, key string, reader io.Reader, contentType string) error {
	if c.putErr != nil {
		return c.putErr
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	if c.objects == nil {
		c.objects = make(map[string][]byte)
	}
	c.objects[key] = body
	c.putBucket = bucket
	c.putKey = key
	c.putContentType = contentType
	return nil
}

func (c *fakeS3Client) GetObject(_ context.Context, _ string, key string) (io.ReadCloser, error) {
	if c.getErr != nil {
		return nil, c.getErr
	}
	body, ok := c.objects[key]
	if !ok {
		return nil, errors.New("object not found")
	}
	return io.NopCloser(bytes.NewReader(body)), nil
}

func (c *fakeS3Client) RemoveObject(_ context.Context, _ string, key string) error {
	if c.deleteErr != nil {
		return c.deleteErr
	}
	delete(c.objects, key)
	c.deletedKey = key
	return nil
}

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

func TestS3StorageValidatesConfiguration(t *testing.T) {
	if _, err := NewS3Storage("", "access", "secret", "evidence", false, ""); err == nil {
		t.Fatal("expected endpoint validation error")
	}
	if _, err := NewS3Storage("minio.test:9000", "", "secret", "evidence", false, ""); err == nil {
		t.Fatal("expected access key validation error")
	}
}

func TestS3StoragePathIsUnavailable(t *testing.T) {
	var _ Storage = (*S3Storage)(nil)
	store := &S3Storage{}
	if _, err := store.Path("subjects/UBS_TEST/evidence/file.txt"); err == nil {
		t.Fatal("expected remote path error")
	}
}

func TestS3StorageRejectsUnsafeKeys(t *testing.T) {
	client := &fakeS3Client{bucketExists: true}
	store, err := newS3Storage("minio.test:9000", "access", "secret", "evidence", false, "", func(string, string, string, bool) (s3Client, error) {
		return client, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"/absolute.txt", `\absolute.txt`, "../outside.txt", "subjects/./file.txt", "subjects/../file.txt", `subjects\file.txt`} {
		t.Run(key, func(t *testing.T) {
			if _, err := store.Upload(context.Background(), key, strings.NewReader("x"), "text/plain"); err == nil {
				t.Fatal("unsafe key accepted")
			}
		})
	}
}

func TestS3StorageEscapesObjectKeyInURL(t *testing.T) {
	store := &S3Storage{endpoint: "minio.test:9000", bucket: "evidence", publicBase: "https://cdn.test/evidence"}
	key := "subjects/name with spaces/evidence/#proof?.txt"
	if got, want := store.GetURL(key), "https://cdn.test/evidence/subjects/name%20with%20spaces/evidence/%23proof%3F.txt"; got != want {
		t.Fatalf("GetURL(%q) = %q, want %q", key, got, want)
	}
}

func TestS3StorageCreatesMissingBucketAndOperatesOnObjects(t *testing.T) {
	client := &fakeS3Client{}
	store, err := newS3Storage("minio.test:9000", "access", "secret", "evidence", false, "https://cdn.test/evidence", func(string, string, string, bool) (s3Client, error) {
		return client, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if client.madeBucket != "evidence" {
		t.Fatalf("made bucket = %q, want evidence", client.madeBucket)
	}

	ctx := context.Background()
	key := "subjects/UBS_TEST/evidence/file.txt"
	url, err := store.Upload(ctx, key, strings.NewReader("evidence"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://cdn.test/evidence/subjects/UBS_TEST/evidence/file.txt" {
		t.Fatalf("upload URL = %q", url)
	}
	if client.putBucket != "evidence" || client.putKey != key || client.putContentType != "text/plain" {
		t.Fatalf("put = bucket %q, key %q, content type %q", client.putBucket, client.putKey, client.putContentType)
	}

	object, err := store.Open(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(object)
	if err != nil {
		t.Fatal(err)
	}
	if err := object.Close(); err != nil {
		t.Fatal(err)
	}
	if string(body) != "evidence" {
		t.Fatalf("opened body = %q, want evidence", body)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if client.deletedKey != key {
		t.Fatalf("deleted key = %q, want %q", client.deletedKey, key)
	}
}

func TestS3StoragePropagatesClientFailures(t *testing.T) {
	bucketErr := errors.New("bucket unavailable")
	if _, err := newS3Storage("minio.test:9000", "access", "secret", "evidence", false, "", func(string, string, string, bool) (s3Client, error) {
		return &fakeS3Client{bucketErr: bucketErr}, nil
	}); !errors.Is(err, bucketErr) {
		t.Fatalf("bucket error = %v, want %v", err, bucketErr)
	}
	makeBucketErr := errors.New("create bucket failed")
	if _, err := newS3Storage("minio.test:9000", "access", "secret", "evidence", false, "", func(string, string, string, bool) (s3Client, error) {
		return &fakeS3Client{makeBucketErr: makeBucketErr}, nil
	}); !errors.Is(err, makeBucketErr) {
		t.Fatalf("make bucket error = %v, want %v", err, makeBucketErr)
	}

	putErr := errors.New("put failed")
	getErr := errors.New("get failed")
	deleteErr := errors.New("delete failed")
	store, err := newS3Storage("minio.test:9000", "access", "secret", "evidence", false, "", func(string, string, string, bool) (s3Client, error) {
		return &fakeS3Client{bucketExists: true, putErr: putErr, getErr: getErr, deleteErr: deleteErr}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Upload(context.Background(), "file.txt", strings.NewReader("x"), "text/plain"); !errors.Is(err, putErr) {
		t.Fatalf("upload error = %v, want %v", err, putErr)
	}
	if _, err := store.Open(context.Background(), "file.txt"); !errors.Is(err, getErr) {
		t.Fatalf("open error = %v, want %v", err, getErr)
	}
	if err := store.Delete(context.Background(), "file.txt"); !errors.Is(err, deleteErr) {
		t.Fatalf("delete error = %v, want %v", err, deleteErr)
	}
}

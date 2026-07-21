package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Storage implements Storage using an S3-compatible object store.
type S3Storage struct {
	client     s3Client
	bucket     string
	endpoint   string
	useSSL     bool
	publicBase string
}

// s3Client is the small subset of object-store behavior used by S3Storage.
// Keeping it narrow permits deterministic storage tests without a MinIO server.
type s3Client interface {
	BucketExists(ctx context.Context, bucket string) (bool, error)
	MakeBucket(ctx context.Context, bucket string) error
	PutObject(ctx context.Context, bucket, key string, reader io.Reader, contentType string) error
	GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	RemoveObject(ctx context.Context, bucket, key string) error
}

type s3ClientFactory func(endpoint, accessKey, secretKey string, useSSL bool) (s3Client, error)

type minioS3Client struct {
	client *minio.Client
}

func newMinioS3Client(endpoint, accessKey, secretKey string, useSSL bool) (s3Client, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	return &minioS3Client{client: client}, nil
}

func (c *minioS3Client) BucketExists(ctx context.Context, bucket string) (bool, error) {
	return c.client.BucketExists(ctx, bucket)
}

func (c *minioS3Client) MakeBucket(ctx context.Context, bucket string) error {
	return c.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}

func (c *minioS3Client) PutObject(ctx context.Context, bucket, key string, reader io.Reader, contentType string) error {
	_, err := c.client.PutObject(ctx, bucket, key, reader, -1, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (c *minioS3Client) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	object, err := c.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	if _, err := object.Stat(); err != nil {
		_ = object.Close()
		return nil, err
	}
	return object, nil
}

func (c *minioS3Client) RemoveObject(ctx context.Context, bucket, key string) error {
	return c.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

// NewS3Storage verifies that bucket access is available before returning.
func NewS3Storage(endpoint, accessKey, secretKey, bucket string, useSSL bool, publicBase string) (*S3Storage, error) {
	return newS3Storage(endpoint, accessKey, secretKey, bucket, useSSL, publicBase, newMinioS3Client)
}

func newS3Storage(endpoint, accessKey, secretKey, bucket string, useSSL bool, publicBase string, createClient s3ClientFactory) (*S3Storage, error) {
	endpoint = strings.TrimSpace(endpoint)
	accessKey = strings.TrimSpace(accessKey)
	secretKey = strings.TrimSpace(secretKey)
	bucket = strings.TrimSpace(bucket)
	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return nil, fmt.Errorf("S3 endpoint, access key, secret key, and bucket are required")
	}

	client, err := createClient(endpoint, accessKey, secretKey, useSSL)
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("check S3 bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket); err != nil {
			return nil, fmt.Errorf("create S3 bucket: %w", err)
		}
	}
	return &S3Storage{client: client, bucket: bucket, endpoint: endpoint, useSSL: useSSL, publicBase: strings.TrimRight(publicBase, "/")}, nil
}

func (s *S3Storage) Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	if err := validateObjectKey(key); err != nil {
		return "", err
	}
	if err := s.client.PutObject(ctx, s.bucket, key, reader, contentType); err != nil {
		return "", err
	}
	return s.GetURL(key), nil
}

func (s *S3Storage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := validateObjectKey(key); err != nil {
		return nil, err
	}
	return s.client.GetObject(ctx, s.bucket, key)
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	if err := validateObjectKey(key); err != nil {
		return err
	}
	return s.client.RemoveObject(ctx, s.bucket, key)
}

func validateObjectKey(key string) error {
	if strings.TrimSpace(key) == "" || strings.Contains(key, "\x00") || strings.Contains(key, `\`) || strings.HasPrefix(key, "/") || path.IsAbs(key) {
		return fmt.Errorf("invalid storage key")
	}
	for _, segment := range strings.Split(key, "/") {
		if segment == "." || segment == ".." {
			return fmt.Errorf("invalid storage key")
		}
	}
	return nil
}

func (s *S3Storage) GetURL(key string) string {
	escapedKey := escapeObjectKey(key)
	if s.publicBase != "" {
		return s.publicBase + "/" + escapedKey
	}
	scheme := "http"
	if s.useSSL {
		scheme = "https"
	}
	return scheme + "://" + s.endpoint + "/" + s.bucket + "/" + escapedKey
}

func escapeObjectKey(key string) string {
	parts := strings.Split(strings.TrimLeft(key, "/"), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func (s *S3Storage) Path(key string) (string, error) {
	return "", fmt.Errorf("storage path is unavailable for remote S3 objects")
}

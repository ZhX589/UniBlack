package main

import (
	"errors"
	"fmt"

	"github.com/ZhX589/UniBlack/backend/internal/config"
	"github.com/ZhX589/UniBlack/backend/internal/storage"
)

var errS3EndpointRequired = errors.New("MINIO_ENDPOINT is required outside development")

type storageConstructors struct {
	s3    func(string, string, string, string, bool, string) (storage.Storage, error)
	local func(string, string) storage.Storage
}

func defaultStorageConstructors() storageConstructors {
	return storageConstructors{
		s3: func(endpoint, accessKey, secretKey, bucket string, useSSL bool, publicBase string) (storage.Storage, error) {
			return storage.NewS3Storage(endpoint, accessKey, secretKey, bucket, useSSL, publicBase)
		},
		local: func(basePath, baseURL string) storage.Storage {
			return storage.NewLocalStorage(basePath, baseURL)
		},
	}
}

func selectStorage(cfg *config.Config, constructors storageConstructors) (storage.Storage, error) {
	if cfg.MinioEndpoint != "" {
		store, err := constructors.s3(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucket, cfg.MinioUseSSL, cfg.MinioPublicBase)
		if err != nil {
			return nil, fmt.Errorf("S3 storage initialization failed: %w", err)
		}
		return store, nil
	}
	if cfg.Environment != "development" && cfg.Environment != "dev" {
		return nil, errS3EndpointRequired
	}
	return constructors.local("./uploads", "http://localhost:8080/uploads"), nil
}

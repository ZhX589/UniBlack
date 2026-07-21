package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	defaultDatabaseURL    = "postgres://uniblack:uniblack@localhost:5432/uniblack?sslmode=disable"
	defaultJWTSecret      = "change-me-access-secret"
	defaultRefreshSecret  = "change-me-refresh-secret"
	defaultMinioEndpoint  = "localhost:9000"
	defaultMinioAccessKey = "minioadmin"
	defaultMinioSecretKey = "minioadmin"
	defaultMinioBucket    = "uniblack-evidence"
)

type Config struct {
	Environment     string
	Port            string
	DatabaseURL     string
	JWTSecret       string
	RefreshSecret   string
	MinioEndpoint   string
	MinioAccessKey  string
	MinioSecretKey  string
	MinioBucket     string
	MinioUseSSL     bool
	MinioPublicBase string
}

func Load() *Config {
	return &Config{
		Environment:     strings.ToLower(os.Getenv("GO_ENV")),
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", defaultDatabaseURL),
		JWTSecret:       getEnv("JWT_SECRET", defaultJWTSecret),
		RefreshSecret:   getEnv("REFRESH_SECRET", defaultRefreshSecret),
		MinioEndpoint:   getEnv("MINIO_ENDPOINT", defaultMinioEndpoint),
		MinioAccessKey:  getEnv("MINIO_ACCESS_KEY", defaultMinioAccessKey),
		MinioSecretKey:  getEnv("MINIO_SECRET_KEY", defaultMinioSecretKey),
		MinioBucket:     getEnv("MINIO_BUCKET", defaultMinioBucket),
		MinioUseSSL:     getBoolEnv("MINIO_USE_SSL", false),
		MinioPublicBase: getEnv("MINIO_PUBLIC_BASE", ""),
	}
}

// Validate rejects deployment settings that would make production unsafe.
func (c Config) Validate() error {
	if c.Environment != "production" {
		if c.Environment != "" && c.Environment != "development" && c.Environment != "dev" && c.Environment != "test" {
			return fmt.Errorf("GO_ENV must be development, test, or production")
		}
		return nil
	}
	if strings.TrimSpace(c.DatabaseURL) == "" || c.DatabaseURL == defaultDatabaseURL {
		return fmt.Errorf("DATABASE_URL is required in production")
	}
	if strings.TrimSpace(c.JWTSecret) == "" || c.JWTSecret == defaultJWTSecret {
		return fmt.Errorf("JWT_SECRET must be configured in production")
	}
	if strings.TrimSpace(c.RefreshSecret) == "" || c.RefreshSecret == defaultRefreshSecret {
		return fmt.Errorf("REFRESH_SECRET must be configured in production")
	}
	if strings.TrimSpace(c.MinioEndpoint) == "" || strings.TrimSpace(c.MinioAccessKey) == "" || strings.TrimSpace(c.MinioSecretKey) == "" || strings.TrimSpace(c.MinioBucket) == "" || c.MinioEndpoint == defaultMinioEndpoint || c.MinioAccessKey == defaultMinioAccessKey || c.MinioSecretKey == defaultMinioSecretKey || c.MinioBucket == defaultMinioBucket {
		return fmt.Errorf("MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY, and MINIO_BUCKET are required in production")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

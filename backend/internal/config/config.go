package config

import (
	"os"
)

type Config struct {
	Port           string
	DatabaseURL    string
	JWTSecret      string
	RefreshSecret  string
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool
}

func Load() *Config {
	return &Config{
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://uniblack:uniblack@localhost:5432/uniblack?sslmode=disable"),
		JWTSecret:      getEnv("JWT_SECRET", "change-me-access-secret"),
		RefreshSecret:  getEnv("REFRESH_SECRET", "change-me-refresh-secret"),
		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:    getEnv("MINIO_BUCKET", "uniblack-evidence"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

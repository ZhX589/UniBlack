package config

import (
	"os"
	"testing"
)

func TestValidateRejectsProductionDefaultsAndMissingRequiredConfiguration(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "default JWT secrets",
			cfg:  Config{Environment: "production", DatabaseURL: "postgres://db", JWTSecret: defaultJWTSecret, RefreshSecret: defaultRefreshSecret},
		},
		{
			name: "missing database URL",
			cfg:  Config{Environment: "production", JWTSecret: "access", RefreshSecret: "refresh"},
		},
		{
			name: "unknown environment",
			cfg:  Config{Environment: "staging", DatabaseURL: "postgres://db", JWTSecret: "access", RefreshSecret: "refresh"},
		},
		{
			name: "missing object storage credentials",
			cfg:  Config{Environment: "production", DatabaseURL: "postgres://db", JWTSecret: "access", RefreshSecret: "refresh", MinioEndpoint: "minio:9000", MinioBucket: "evidence"},
		},
		{
			name: "default MinIO access key",
			cfg:  Config{Environment: "production", DatabaseURL: "postgres://db", JWTSecret: "access", RefreshSecret: "refresh", MinioEndpoint: "minio:9000", MinioAccessKey: defaultMinioAccessKey, MinioSecretKey: "real-secret", MinioBucket: "evidence"},
		},
		{
			name: "default MinIO secret key",
			cfg:  Config{Environment: "production", DatabaseURL: "postgres://db", JWTSecret: "access", RefreshSecret: "refresh", MinioEndpoint: "minio:9000", MinioAccessKey: "real-key", MinioSecretKey: defaultMinioSecretKey, MinioBucket: "evidence"},
		},
		{
			name: "whitespace MinIO access key",
			cfg:  Config{Environment: "production", DatabaseURL: "postgres://db", JWTSecret: "access", RefreshSecret: "refresh", MinioEndpoint: "minio:9000", MinioAccessKey: "  ", MinioSecretKey: "real-secret", MinioBucket: "evidence"},
		},
		{
			name: "whitespace MinIO secret key",
			cfg:  Config{Environment: "production", DatabaseURL: "postgres://db", JWTSecret: "access", RefreshSecret: "refresh", MinioEndpoint: "minio:9000", MinioAccessKey: "real-key", MinioSecretKey: "  ", MinioBucket: "evidence"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.cfg.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want configuration error")
			}
		})
	}
}

func TestValidateAllowsExplicitProductionConfiguration(t *testing.T) {
	cfg := Config{Environment: "production", DatabaseURL: "postgres://db", JWTSecret: "access", RefreshSecret: "refresh", MinioEndpoint: "minio:9000", MinioAccessKey: "access", MinioSecretKey: "secret", MinioBucket: "evidence"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

// Default endpoint/bucket are names, not secrets; explicit credentials must
// still validate even if those locations keep their bundled defaults.
func TestValidateAllowsDefaultEndpointOrBucketWithExplicitCredentials(t *testing.T) {
	cases := []Config{
		{Environment: "production", DatabaseURL: "postgres://db", JWTSecret: "access", RefreshSecret: "refresh", MinioEndpoint: defaultMinioEndpoint, MinioAccessKey: "real-key", MinioSecretKey: "real-secret", MinioBucket: "evidence"},
		{Environment: "production", DatabaseURL: "postgres://db", JWTSecret: "access", RefreshSecret: "refresh", MinioEndpoint: "minio:9000", MinioAccessKey: "real-key", MinioSecretKey: "real-secret", MinioBucket: defaultMinioBucket},
	}
	for _, cfg := range cases {
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	}
}

// Load() substitutes development MinIO credentials when the env vars are
// absent; production validation must therefore reject the resulting config.
func TestLoadRejectsProductionDefaultsWhenMinioCredentialsAbsent(t *testing.T) {
	t.Setenv("GO_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://db")
	t.Setenv("JWT_SECRET", "access")
	t.Setenv("REFRESH_SECRET", "refresh")
	t.Setenv("MINIO_ENDPOINT", "minio:9000")
	t.Setenv("MINIO_BUCKET", "evidence")
	for _, key := range []string{"MINIO_ACCESS_KEY", "MINIO_SECRET_KEY"} {
		// Setenv registers restoration of the original environment, including
		// whether the variable existed, before testing an absent value.
		t.Setenv(key, "unused")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	cfg := Load()
	if cfg.MinioAccessKey != defaultMinioAccessKey || cfg.MinioSecretKey != defaultMinioSecretKey {
		t.Fatal("absent credentials did not exercise development defaults")
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Load().Validate() accepted production defaults")
	}
}

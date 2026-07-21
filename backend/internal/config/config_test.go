package config

import "testing"

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
			name: "default object storage credentials",
			cfg:  Config{Environment: "production", DatabaseURL: "postgres://db", JWTSecret: "access", RefreshSecret: "refresh", MinioEndpoint: "localhost:9000", MinioAccessKey: "minioadmin", MinioSecretKey: "minioadmin", MinioBucket: "uniblack-evidence"},
		},
		{
			name: "default object storage endpoint",
			cfg:  Config{Environment: "production", DatabaseURL: "postgres://db", JWTSecret: "access", RefreshSecret: "refresh", MinioEndpoint: defaultMinioEndpoint, MinioAccessKey: "access", MinioSecretKey: "secret", MinioBucket: "evidence"},
		},
		{
			name: "default object storage bucket",
			cfg:  Config{Environment: "production", DatabaseURL: "postgres://db", JWTSecret: "access", RefreshSecret: "refresh", MinioEndpoint: "minio:9000", MinioAccessKey: "access", MinioSecretKey: "secret", MinioBucket: defaultMinioBucket},
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

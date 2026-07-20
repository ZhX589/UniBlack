package auth

import (
	"context"
)

// SubjectIdentity represents the authenticated user's identity
type SubjectIdentity struct {
	UserID    string   `json:"user_id"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	Provider  string   `json:"provider"`
}

// Credentials represents the authentication credentials
type Credentials struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// AuthProvider defines the interface for authentication providers
// Implementations: JWT (local), OAuth2 (github, discord, etc.)
type AuthProvider interface {
	// Name returns the provider name (e.g., "local", "oauth:github")
	Name() string

	// Verify validates credentials and returns the user identity
	Verify(ctx context.Context, creds Credentials) (*SubjectIdentity, error)

	// GenerateTokens creates a new token pair for the user
	GenerateTokens(ctx context.Context, identity *SubjectIdentity) (*TokenPair, error)

	// RefreshToken validates a refresh token and generates new tokens
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)

	// ValidateToken validates an access token and returns the identity
	ValidateToken(ctx context.Context, accessToken string) (*SubjectIdentity, error)
}

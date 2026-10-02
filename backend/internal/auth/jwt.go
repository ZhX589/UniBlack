package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
)

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret        string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	Issuer        string
}

// JWTProvider implements AuthProvider using JWT
type JWTProvider struct {
	config JWTConfig
}

// NewJWTProvider creates a new JWT auth provider
func NewJWTProvider(config JWTConfig) *JWTProvider {
	if config.AccessTTL == 0 {
		config.AccessTTL = 15 * time.Minute
	}
	if config.RefreshTTL == 0 {
		config.RefreshTTL = 7 * 24 * time.Hour
	}
	if config.Issuer == "" {
		config.Issuer = "uniblack"
	}
	return &JWTProvider{config: config}
}

func (p *JWTProvider) Name() string {
	return "local"
}

func (p *JWTProvider) Verify(ctx context.Context, creds Credentials) (*SubjectIdentity, error) {
	// This is handled by the service layer
	return nil, errors.New("use service layer for verification")
}

// Claims represents JWT claims
type Claims struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	AuthVersion int64    `json:"auth_version"`
	TokenUse    string   `json:"token_use"`
	jwt.RegisteredClaims
}

func (p *JWTProvider) GenerateTokens(ctx context.Context, identity *SubjectIdentity) (*TokenPair, error) {
	now := time.Now()

	// Generate access token
	accessClaims := &Claims{
		UserID:      identity.UserID,
		Username:    identity.Username,
		Email:       identity.Email,
		Roles:       identity.Roles,
		AuthVersion: identity.AuthVersion,
		TokenUse:    "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    p.config.Issuer,
			Subject:   identity.UserID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(p.config.AccessTTL)),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenStr, err := accessToken.SignedString([]byte(p.config.Secret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token
	refreshClaims := &Claims{
		UserID:      identity.UserID,
		AuthVersion: identity.AuthVersion,
		TokenUse:    "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    p.config.Issuer,
			Subject:   identity.UserID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(p.config.RefreshTTL)),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString([]byte(p.config.RefreshSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenStr,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int64(p.config.AccessTTL.Seconds()),
	}, nil
}

func (p *JWTProvider) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	identity, err := p.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	return p.GenerateTokens(ctx, identity)
}

func (p *JWTProvider) ValidateRefreshToken(ctx context.Context, refreshToken string) (*SubjectIdentity, error) {
	claims, err := p.validateToken(refreshToken, p.config.RefreshSecret)
	if err != nil {
		return nil, err
	}

	if claims.TokenUse != "" && claims.TokenUse != "refresh" {
		return nil, ErrInvalidToken
	}
	return &SubjectIdentity{UserID: claims.UserID, Provider: p.Name(), AuthVersion: claims.AuthVersion}, nil
}

func (p *JWTProvider) ValidateToken(ctx context.Context, accessToken string) (*SubjectIdentity, error) {
	claims, err := p.validateToken(accessToken, p.config.Secret)
	if err != nil {
		return nil, err
	}

	if claims.TokenUse != "" && claims.TokenUse != "access" {
		return nil, ErrInvalidToken
	}
	return &SubjectIdentity{
		AuthVersion: claims.AuthVersion,
		UserID:      claims.UserID,
		Username:    claims.Username,
		Email:       claims.Email,
		Roles:       claims.Roles,
		Provider:    p.Name(),
	}, nil
}

func (p *JWTProvider) validateToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(p.config.Issuer), jwt.WithExpirationRequired())

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid || claims.UserID == "" || claims.Subject != claims.UserID || claims.AuthVersion < 0 {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

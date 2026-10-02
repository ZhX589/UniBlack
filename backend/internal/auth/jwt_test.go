package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTSeparatesCredentialUseEvenWithEqualSecrets(t *testing.T) {
	p := NewJWTProvider(JWTConfig{Secret: "same-secret", RefreshSecret: "same-secret"})
	pair, err := p.GenerateTokens(context.Background(), &SubjectIdentity{UserID: "user", AuthVersion: 7})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.ValidateToken(context.Background(), pair.RefreshToken); err == nil {
		t.Fatal("refresh accepted as access")
	}
	if _, err := p.ValidateRefreshToken(context.Background(), pair.AccessToken); err == nil {
		t.Fatal("access accepted as refresh")
	}
	identity, err := p.ValidateRefreshToken(context.Background(), pair.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if identity.AuthVersion != 7 {
		t.Fatalf("version=%d", identity.AuthVersion)
	}
}

func TestJWTRejectsInvalidCredentialContracts(t *testing.T) {
	p := NewJWTProvider(JWTConfig{Secret: "access-secret", RefreshSecret: "refresh-secret", Issuer: "uniblack"})
	for _, tc := range []struct {
		name    string
		method  jwt.SigningMethod
		issuer  string
		expires bool
	}{
		{"wrong algorithm", jwt.SigningMethodHS384, "uniblack", true},
		{"wrong issuer", jwt.SigningMethodHS256, "other", true},
		{"missing expiry", jwt.SigningMethodHS256, "uniblack", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := &Claims{UserID: "user-id", RegisteredClaims: jwt.RegisteredClaims{Issuer: tc.issuer, Subject: "user-id"}}
			if tc.expires {
				claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour))
			}
			for _, refresh := range []bool{false, true} {
				secret := "access-secret"
				if refresh {
					secret = "refresh-secret"
				}
				raw, err := jwt.NewWithClaims(tc.method, claims).SignedString([]byte(secret))
				if err != nil {
					t.Fatal(err)
				}
				if refresh {
					if _, err := p.ValidateRefreshToken(context.Background(), raw); err == nil {
						t.Fatal("invalid refresh accepted")
					}
				} else if _, err := p.ValidateToken(context.Background(), raw); err == nil {
					t.Fatal("invalid access accepted")
				}
			}
		})
	}
}

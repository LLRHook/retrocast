package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims defines the JWT payload for access tokens.
type Claims struct {
	UserID int64 `json:"user_id,string"`
	jwt.RegisteredClaims
}

// TokenService manages JWT access tokens and opaque refresh tokens.
type TokenService struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// NewTokenService creates a TokenService with the given HMAC secret.
func NewTokenService(secret string) *TokenService {
	return &TokenService{
		secret:        []byte(secret),
		accessExpiry:  15 * time.Minute,
		refreshExpiry: 7 * 24 * time.Hour,
	}
}

// RefreshExpiry returns the configured refresh token expiry duration.
func (ts *TokenService) RefreshExpiry() time.Duration {
	return ts.refreshExpiry
}

// GenerateAccessToken creates a signed JWT with the given user ID.
func (ts *TokenService) GenerateAccessToken(userID int64) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ts.accessExpiry)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(ts.secret)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}
	return signed, nil
}

// GenerateRefreshToken creates a random opaque token (32 bytes, hex-encoded).
func (ts *TokenService) GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating refresh token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// ValidateAccessToken parses and validates a JWT, returning the claims.
func (ts *TokenService) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return ts.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

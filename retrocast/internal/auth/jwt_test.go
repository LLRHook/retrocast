package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndValidateAccessToken(t *testing.T) {
	ts := NewTokenService("test-secret-key")

	token, err := ts.GenerateAccessToken(42)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateAccessToken() returned empty token")
	}

	claims, err := ts.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("UserID = %d, want 42", claims.UserID)
	}
}

func TestRejectExpiredToken(t *testing.T) {
	ts := &TokenService{
		secret:        []byte("test-secret-key"),
		accessExpiry:  -1 * time.Second, // already expired
		refreshExpiry: 7 * 24 * time.Hour,
	}

	token, err := ts.GenerateAccessToken(1)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error: %v", err)
	}

	_, err = ts.ValidateAccessToken(token)
	if err == nil {
		t.Error("ValidateAccessToken() should reject expired token")
	}
}

func TestRejectTamperedToken(t *testing.T) {
	ts := NewTokenService("test-secret-key")

	token, err := ts.GenerateAccessToken(1)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error: %v", err)
	}

	// Tamper with a character in the middle of the signature to avoid
	// base64 padding-bit ambiguity at the last position. For HMAC-SHA256
	// (32 bytes), the last base64url char has 2 padding bits that Go's
	// decoder ignores, so changing only those bits won't alter the
	// decoded signature (~6% of runs).
	sigStart := strings.LastIndex(token, ".") + 1
	mid := sigStart + (len(token)-sigStart)/2
	b := token[mid]
	if b == 'A' {
		b = 'B'
	} else {
		b = 'A'
	}
	tampered := token[:mid] + string(b) + token[mid+1:]

	_, err = ts.ValidateAccessToken(tampered)
	if err == nil {
		t.Error("ValidateAccessToken() should reject tampered token")
	}
}

func TestRejectWrongSigningMethod(t *testing.T) {
	// Create a token with a different signing method (none)
	claims := Claims{
		UserID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signing with none: %v", err)
	}

	ts := NewTokenService("test-secret-key")
	_, err = ts.ValidateAccessToken(tokenString)
	if err == nil {
		t.Error("ValidateAccessToken() should reject token with 'none' signing method")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	ts := NewTokenService("test-secret-key")

	token, err := ts.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error: %v", err)
	}

	// Should be 64 hex characters (32 bytes)
	if len(token) != 64 {
		t.Errorf("refresh token length = %d, want 64", len(token))
	}

	// Two tokens should differ
	token2, _ := ts.GenerateRefreshToken()
	if token == token2 {
		t.Error("two refresh tokens should not be identical")
	}
}

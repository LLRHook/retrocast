package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/models"
	redisclient "github.com/victorivanov/retrocast/internal/redis"
	"github.com/victorivanov/retrocast/internal/service"
)

func newTestRedis(t *testing.T) *redisclient.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb, err := redisclient.NewClient("redis://" + mr.Addr())
	if err != nil {
		t.Fatalf("creating test redis client: %v", err)
	}
	t.Cleanup(func() { rdb.Close() })
	return rdb
}

func newTestAuthHandler(t *testing.T, users *mockUserRepo) *AuthHandler {
	t.Helper()
	rdb := newTestRedis(t)
	tokens := auth.NewTokenService("test-secret")
	sf := testSnowflake()
	svc := service.NewAuthService(users, tokens, rdb, sf)
	return NewAuthHandler(svc)
}

func TestRegister_Success(t *testing.T) {
	users := &mockUserRepo{}
	h := newTestAuthHandler(t, users)

	body := strings.NewReader(`{"username":"testuser","password":"password123"}`)
	c, rec := newTestContext(http.MethodPost, "/api/v1/auth/register", body)

	if err := h.Register(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp authResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh_token")
	}
	// User is interface{} now; check via map assertion.
	userMap, ok := resp.User.(map[string]interface{})
	if !ok {
		t.Fatal("expected user to be a map")
	}
	if userMap["username"] != "testuser" {
		t.Errorf("expected username 'testuser', got %v", userMap["username"])
	}
}

func TestRegister_DuplicateUsername(t *testing.T) {
	users := &mockUserRepo{
		GetByUsernameFn: func(_ context.Context, username string) (*models.User, error) {
			if username == "taken" {
				return &models.User{ID: 1, Username: "taken"}, nil
			}
			return nil, nil
		},
	}
	h := newTestAuthHandler(t, users)

	body := strings.NewReader(`{"username":"taken","password":"password123"}`)
	c, rec := newTestContext(http.MethodPost, "/api/v1/auth/register", body)

	if err := h.Register(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if errResp.Error.Code != "USERNAME_TAKEN" {
		t.Errorf("expected error code 'USERNAME_TAKEN', got %q", errResp.Error.Code)
	}
}

func TestRegister_InvalidInput(t *testing.T) {
	users := &mockUserRepo{}
	h := newTestAuthHandler(t, users)

	tests := []struct {
		name     string
		body     string
		wantCode string
	}{
		{
			name:     "short username",
			body:     `{"username":"a","password":"password123"}`,
			wantCode: "INVALID_USERNAME",
		},
		{
			name:     "short password",
			body:     `{"username":"validuser","password":"12345"}`,
			wantCode: "INVALID_PASSWORD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newTestContext(http.MethodPost, "/api/v1/auth/register", strings.NewReader(tt.body))

			if err := h.Register(c); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
			}

			var errResp ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if errResp.Error.Code != tt.wantCode {
				t.Errorf("expected error code %q, got %q", tt.wantCode, errResp.Error.Code)
			}
		})
	}
}

func TestLogin_Success(t *testing.T) {
	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("hashing password: %v", err)
	}

	users := &mockUserRepo{
		GetByUsernameFn: func(_ context.Context, username string) (*models.User, error) {
			if username == "testuser" {
				return &models.User{ID: 100, Username: "testuser", PasswordHash: hash}, nil
			}
			return nil, nil
		},
	}
	h := newTestAuthHandler(t, users)

	body := strings.NewReader(`{"username":"testuser","password":"password123"}`)
	c, rec := newTestContext(http.MethodPost, "/api/v1/auth/login", body)

	if err := h.Login(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp authResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	userMap, ok := resp.User.(map[string]interface{})
	if !ok {
		t.Fatal("expected user to be a map")
	}
	if userMap["username"] != "testuser" {
		t.Errorf("expected username 'testuser', got %v", userMap["username"])
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	hash, _ := auth.HashPassword("correctpassword")

	users := &mockUserRepo{
		GetByUsernameFn: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{ID: 100, Username: "testuser", PasswordHash: hash}, nil
		},
	}
	h := newTestAuthHandler(t, users)

	body := strings.NewReader(`{"username":"testuser","password":"wrongpassword"}`)
	c, rec := newTestContext(http.MethodPost, "/api/v1/auth/login", body)

	if err := h.Login(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if errResp.Error.Code != "INVALID_CREDENTIALS" {
		t.Errorf("expected error code 'INVALID_CREDENTIALS', got %q", errResp.Error.Code)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	users := &mockUserRepo{}
	h := newTestAuthHandler(t, users)

	body := strings.NewReader(`{"username":"nonexistent","password":"password123"}`)
	c, rec := newTestContext(http.MethodPost, "/api/v1/auth/login", body)

	if err := h.Login(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if errResp.Error.Code != "INVALID_CREDENTIALS" {
		t.Errorf("expected error code 'INVALID_CREDENTIALS', got %q", errResp.Error.Code)
	}
}

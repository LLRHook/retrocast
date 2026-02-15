package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/service"
)

func TestGetMe(t *testing.T) {
	users := &mockUserRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.User, error) {
			if id == 1 {
				return &models.User{ID: 1, Username: "testuser", DisplayName: "Test User"}, nil
			}
			return nil, nil
		},
	}
	svc := service.NewUserService(users)
	h := NewUserHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/users/@me", nil)
	setAuthUser(c, 1)

	if err := h.GetMe(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp struct {
		Data models.User `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Data.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", resp.Data.Username)
	}
	if resp.Data.DisplayName != "Test User" {
		t.Errorf("expected display_name 'Test User', got %q", resp.Data.DisplayName)
	}
}

func TestGetMe_NotFound(t *testing.T) {
	users := &mockUserRepo{} // GetByIDFn returns nil, nil by default
	svc := service.NewUserService(users)
	h := NewUserHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/users/@me", nil)
	setAuthUser(c, 999)

	if err := h.GetMe(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if errResp.Error.Code != "NOT_FOUND" {
		t.Errorf("expected error code 'NOT_FOUND', got %q", errResp.Error.Code)
	}
}

func TestGetMe_InternalError(t *testing.T) {
	users := &mockUserRepo{
		GetByIDFn: func(_ context.Context, _ int64) (*models.User, error) {
			return nil, fmt.Errorf("db connection lost")
		},
	}
	svc := service.NewUserService(users)
	h := NewUserHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/users/@me", nil)
	setAuthUser(c, 1)

	if err := h.GetMe(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d: %s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}
}

func TestUpdateMe(t *testing.T) {
	var updated bool
	users := &mockUserRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.User, error) {
			if id == 1 {
				return &models.User{ID: 1, Username: "testuser", DisplayName: "Old Name"}, nil
			}
			return nil, nil
		},
		UpdateFn: func(_ context.Context, user *models.User) error {
			updated = true
			if user.DisplayName != "New Name" {
				t.Errorf("expected display_name 'New Name', got %q", user.DisplayName)
			}
			return nil
		},
	}
	svc := service.NewUserService(users)
	h := NewUserHandler(svc)

	body := strings.NewReader(`{"display_name":"New Name"}`)
	c, rec := newTestContext(http.MethodPatch, "/api/v1/users/@me", body)
	setAuthUser(c, 1)

	if err := h.UpdateMe(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !updated {
		t.Error("expected user update to be called")
	}

	var resp struct {
		Data models.User `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Data.DisplayName != "New Name" {
		t.Errorf("expected display_name 'New Name', got %q", resp.Data.DisplayName)
	}
}

func TestUpdateMe_AvatarHash(t *testing.T) {
	var updated bool
	users := &mockUserRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.User, error) {
			if id == 1 {
				return &models.User{ID: 1, Username: "testuser", DisplayName: "Test"}, nil
			}
			return nil, nil
		},
		UpdateFn: func(_ context.Context, user *models.User) error {
			updated = true
			if user.AvatarHash == nil || *user.AvatarHash != "abc123" {
				t.Errorf("expected avatar_hash 'abc123', got %v", user.AvatarHash)
			}
			return nil
		},
	}
	svc := service.NewUserService(users)
	h := NewUserHandler(svc)

	body := strings.NewReader(`{"avatar":"abc123"}`)
	c, rec := newTestContext(http.MethodPatch, "/api/v1/users/@me", body)
	setAuthUser(c, 1)

	if err := h.UpdateMe(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !updated {
		t.Error("expected user update to be called")
	}
}

func TestUpdateMe_EmptyBody(t *testing.T) {
	users := &mockUserRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.User, error) {
			if id == 1 {
				return &models.User{ID: 1, Username: "testuser", DisplayName: "Test"}, nil
			}
			return nil, nil
		},
	}
	svc := service.NewUserService(users)
	h := NewUserHandler(svc)

	// Empty JSON object: no fields to update, but bind succeeds. Update is still called.
	body := strings.NewReader(`{}`)
	c, rec := newTestContext(http.MethodPatch, "/api/v1/users/@me", body)
	setAuthUser(c, 1)

	if err := h.UpdateMe(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestUpdateMe_DisplayNameTooLong(t *testing.T) {
	users := &mockUserRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.User, error) {
			if id == 1 {
				return &models.User{ID: 1, Username: "testuser", DisplayName: "Test"}, nil
			}
			return nil, nil
		},
	}
	svc := service.NewUserService(users)
	h := NewUserHandler(svc)

	longName := strings.Repeat("a", 33)
	body := strings.NewReader(fmt.Sprintf(`{"display_name":"%s"}`, longName))
	c, rec := newTestContext(http.MethodPatch, "/api/v1/users/@me", body)
	setAuthUser(c, 1)

	if err := h.UpdateMe(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if errResp.Error.Code != "INVALID_DISPLAY_NAME" {
		t.Errorf("expected error code 'INVALID_DISPLAY_NAME', got %q", errResp.Error.Code)
	}
}

func TestUpdateMe_DisplayNameEmpty(t *testing.T) {
	users := &mockUserRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.User, error) {
			if id == 1 {
				return &models.User{ID: 1, Username: "testuser", DisplayName: "Test"}, nil
			}
			return nil, nil
		},
	}
	svc := service.NewUserService(users)
	h := NewUserHandler(svc)

	body := strings.NewReader(`{"display_name":""}`)
	c, rec := newTestContext(http.MethodPatch, "/api/v1/users/@me", body)
	setAuthUser(c, 1)

	if err := h.UpdateMe(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if errResp.Error.Code != "INVALID_DISPLAY_NAME" {
		t.Errorf("expected error code 'INVALID_DISPLAY_NAME', got %q", errResp.Error.Code)
	}
}

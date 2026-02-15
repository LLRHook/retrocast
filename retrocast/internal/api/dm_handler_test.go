package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/service"
)

// ---------------------------------------------------------------------------
// Shared test constants and helpers for DM handler tests
// ---------------------------------------------------------------------------

const (
	testRecipientID int64 = 4000
	testDMChannelID int64 = 7000
)

// newDMHandler wires up a DMHandler with the given mocks via the service layer.
func newDMHandler(dms *mockDMChannelRepo, users *mockUserRepo, gw *mockGateway) *DMHandler {
	svc := service.NewDMService(dms, users, testSnowflake(), gw)
	return NewDMHandler(svc)
}

// recipientUser returns a test recipient user.
func recipientUser() *models.User {
	return &models.User{
		ID:          testRecipientID,
		Username:    "recipient",
		DisplayName: "Recipient",
		CreatedAt:   time.Now(),
	}
}

// recipientFoundRepo returns a mockUserRepo that finds the recipient by ID.
func recipientFoundRepo() *mockUserRepo {
	recipient := recipientUser()
	return &mockUserRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.User, error) {
			if id == testRecipientID {
				return recipient, nil
			}
			return nil, nil
		},
	}
}

// newDMChannel returns a test DM channel with both sender and recipient.
func newDMChannel(id int64) *models.DMChannel {
	return &models.DMChannel{
		ID:   id,
		Type: models.DMTypeDM,
		Recipients: []models.User{
			{ID: testUserID, Username: "sender"},
			{ID: testRecipientID, Username: "recipient"},
		},
		CreatedAt: time.Now(),
	}
}

// ---------------------------------------------------------------------------
// CreateDM tests
// ---------------------------------------------------------------------------

func TestCreateDM_Success(t *testing.T) {
	gw := &mockGateway{}
	users := recipientFoundRepo()

	dms := &mockDMChannelRepo{
		GetOrCreateDMFn: func(_ context.Context, user1ID, user2ID, newID int64) (*models.DMChannel, error) {
			// Simulate new DM by returning channel with ID == newID.
			return newDMChannel(newID), nil
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels",
		strings.NewReader(`{"recipient_id":"4000"}`))
	setAuthUser(c, testUserID)

	err := h.CreateDM(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result models.DMChannel
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if result.Type != models.DMTypeDM {
		t.Fatalf("expected type %d, got %d", models.DMTypeDM, result.Type)
	}
	if len(result.Recipients) != 2 {
		t.Fatalf("expected 2 recipients, got %d", len(result.Recipients))
	}

	// New DM dispatches CHANNEL_CREATE to both users.
	if len(gw.events) != 2 {
		t.Fatalf("expected 2 gateway events, got %d", len(gw.events))
	}
	for _, ev := range gw.events {
		if ev.Event != gateway.EventChannelCreate {
			t.Fatalf("expected CHANNEL_CREATE event, got %s", ev.Event)
		}
	}
	// Verify events target the correct users.
	userIDs := map[int64]bool{gw.events[0].UserID: true, gw.events[1].UserID: true}
	if !userIDs[testUserID] || !userIDs[testRecipientID] {
		t.Fatalf("expected events for user %d and %d, got %+v", testUserID, testRecipientID, gw.events)
	}
}

func TestCreateDM_ExistingChannel(t *testing.T) {
	gw := &mockGateway{}
	users := recipientFoundRepo()

	existingDM := newDMChannel(testDMChannelID)
	dms := &mockDMChannelRepo{
		GetOrCreateDMFn: func(_ context.Context, user1ID, user2ID, newID int64) (*models.DMChannel, error) {
			// Return existing DM: ID does NOT match newID.
			return existingDM, nil
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels",
		strings.NewReader(`{"recipient_id":"4000"}`))
	setAuthUser(c, testUserID)

	err := h.CreateDM(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Existing DM should NOT dispatch any gateway events.
	if len(gw.events) != 0 {
		t.Fatalf("expected 0 gateway events for existing DM, got %d", len(gw.events))
	}

	var result models.DMChannel
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if result.ID != testDMChannelID {
		t.Fatalf("expected existing channel ID %d, got %d", testDMChannelID, result.ID)
	}
}

func TestCreateDM_SelfDM(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}
	dms := &mockDMChannelRepo{}

	h := newDMHandler(dms, users, gw)

	// recipient_id matches the authenticated user.
	body := `{"recipient_id":"3000"}`
	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels", strings.NewReader(body))
	setAuthUser(c, testUserID)

	_ = h.CreateDM(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "INVALID_RECIPIENT" {
		t.Fatalf("expected error code INVALID_RECIPIENT, got %s", errResp.Error.Code)
	}
}

func TestCreateDM_InvalidBody_MissingRecipient(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}
	dms := &mockDMChannelRepo{}

	h := newDMHandler(dms, users, gw)

	// Empty object -- recipient_id will be empty string, ParseInt fails.
	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels", strings.NewReader(`{}`))
	setAuthUser(c, testUserID)

	_ = h.CreateDM(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "INVALID_RECIPIENT" {
		t.Fatalf("expected error code INVALID_RECIPIENT, got %s", errResp.Error.Code)
	}
}

func TestCreateDM_InvalidBody_NonNumericRecipient(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}
	dms := &mockDMChannelRepo{}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels",
		strings.NewReader(`{"recipient_id":"not-a-number"}`))
	setAuthUser(c, testUserID)

	_ = h.CreateDM(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "INVALID_RECIPIENT" {
		t.Fatalf("expected error code INVALID_RECIPIENT, got %s", errResp.Error.Code)
	}
}

func TestCreateDM_InvalidBody_ZeroRecipient(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}
	dms := &mockDMChannelRepo{}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels",
		strings.NewReader(`{"recipient_id":"0"}`))
	setAuthUser(c, testUserID)

	_ = h.CreateDM(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "INVALID_RECIPIENT" {
		t.Fatalf("expected error code INVALID_RECIPIENT, got %s", errResp.Error.Code)
	}
}

func TestCreateDM_RecipientNotFound(t *testing.T) {
	gw := &mockGateway{}
	dms := &mockDMChannelRepo{}

	// User repo returns nil for any ID (recipient does not exist).
	users := &mockUserRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.User, error) {
			return nil, nil
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels",
		strings.NewReader(`{"recipient_id":"4000"}`))
	setAuthUser(c, testUserID)

	_ = h.CreateDM(c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "NOT_FOUND" {
		t.Fatalf("expected error code NOT_FOUND, got %s", errResp.Error.Code)
	}
}

func TestCreateDM_RepoError_UserLookup(t *testing.T) {
	gw := &mockGateway{}
	dms := &mockDMChannelRepo{}

	users := &mockUserRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.User, error) {
			return nil, errors.New("db connection lost")
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels",
		strings.NewReader(`{"recipient_id":"4000"}`))
	setAuthUser(c, testUserID)

	_ = h.CreateDM(c)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "INTERNAL" {
		t.Fatalf("expected error code INTERNAL, got %s", errResp.Error.Code)
	}
}

func TestCreateDM_RepoError_GetOrCreateDM(t *testing.T) {
	gw := &mockGateway{}
	users := recipientFoundRepo()

	dms := &mockDMChannelRepo{
		GetOrCreateDMFn: func(_ context.Context, user1ID, user2ID, newID int64) (*models.DMChannel, error) {
			return nil, errors.New("db write failed")
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels",
		strings.NewReader(`{"recipient_id":"4000"}`))
	setAuthUser(c, testUserID)

	_ = h.CreateDM(c)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "INTERNAL" {
		t.Fatalf("expected error code INTERNAL, got %s", errResp.Error.Code)
	}
}

// ---------------------------------------------------------------------------
// ListDMs tests
// ---------------------------------------------------------------------------

func TestListDMs_Success(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}

	channels := []models.DMChannel{
		{
			ID:   testDMChannelID,
			Type: models.DMTypeDM,
			Recipients: []models.User{
				{ID: testUserID, Username: "sender"},
				{ID: testRecipientID, Username: "recipient"},
			},
			CreatedAt: time.Now(),
		},
	}

	dms := &mockDMChannelRepo{
		GetByUserIDFn: func(_ context.Context, userID int64) ([]models.DMChannel, error) {
			if userID == testUserID {
				return channels, nil
			}
			return nil, nil
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodGet, "/api/v1/users/@me/channels", nil)
	setAuthUser(c, testUserID)

	err := h.ListDMs(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result []models.DMChannel
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 DM channel, got %d", len(result))
	}
	if len(result[0].Recipients) != 2 {
		t.Fatalf("expected 2 recipients, got %d", len(result[0].Recipients))
	}
}

func TestListDMs_Empty(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}

	// Repo returns nil to simulate no DM channels.
	dms := &mockDMChannelRepo{
		GetByUserIDFn: func(_ context.Context, userID int64) ([]models.DMChannel, error) {
			return nil, nil
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodGet, "/api/v1/users/@me/channels", nil)
	setAuthUser(c, testUserID)

	err := h.ListDMs(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Must be an empty JSON array, not null.
	var result []models.DMChannel
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if result == nil {
		t.Fatal("expected empty array, got nil")
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 DM channels, got %d", len(result))
	}

	// Verify the raw JSON is [] not null.
	raw := strings.TrimSpace(rec.Body.String())
	if raw != "[]" {
		t.Fatalf("expected raw JSON to be [], got %s", raw)
	}
}

func TestListDMs_RepoError(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}

	dms := &mockDMChannelRepo{
		GetByUserIDFn: func(_ context.Context, userID int64) ([]models.DMChannel, error) {
			return nil, errors.New("db timeout")
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodGet, "/api/v1/users/@me/channels", nil)
	setAuthUser(c, testUserID)

	_ = h.ListDMs(c)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "INTERNAL" {
		t.Fatalf("expected error code INTERNAL, got %s", errResp.Error.Code)
	}
}

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
)

const (
	testDMChannelID  int64 = 8000
	testRecipientID  int64 = 4000
)

func TestCreateDM(t *testing.T) {
	gw := &mockGateway{}
	sf := testSnowflake()

	recipient := &models.User{
		ID:       testRecipientID,
		Username: "recipient",
		DisplayName: "Recipient",
		CreatedAt: time.Now(),
	}

	users := &mockUserRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.User, error) {
			if id == testRecipientID {
				return recipient, nil
			}
			return nil, nil
		},
	}

	dmChannel := &models.DMChannel{
		ID:   testDMChannelID,
		Type: models.DMTypeDM,
		Recipients: []models.User{
			{ID: testUserID, Username: "sender"},
			*recipient,
		},
		CreatedAt: time.Now(),
	}

	dms := &mockDMChannelRepo{
		GetOrCreateDMFn: func(_ context.Context, user1ID, user2ID, newID int64) (*models.DMChannel, error) {
			// Simulate creating a new DM — return with newID to signal it was just created.
			dmChannel.ID = newID
			return dmChannel, nil
		},
	}

	h := NewDMHandler(dms, users, sf, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels", strings.NewReader(`{"recipient_id":"`+
		"4000"+`"}`))
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

	// New DM should dispatch CHANNEL_CREATE to both users.
	if len(gw.events) != 2 {
		t.Fatalf("expected 2 gateway events, got %d", len(gw.events))
	}
	for _, ev := range gw.events {
		if ev.Event != gateway.EventChannelCreate {
			t.Fatalf("expected CHANNEL_CREATE event, got %s", ev.Event)
		}
	}
}

func TestCreateDM_Existing(t *testing.T) {
	gw := &mockGateway{}
	sf := testSnowflake()

	recipient := &models.User{
		ID:       testRecipientID,
		Username: "recipient",
		DisplayName: "Recipient",
		CreatedAt: time.Now(),
	}

	users := &mockUserRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.User, error) {
			if id == testRecipientID {
				return recipient, nil
			}
			return nil, nil
		},
	}

	existingDM := &models.DMChannel{
		ID:   testDMChannelID,
		Type: models.DMTypeDM,
		Recipients: []models.User{
			{ID: testUserID, Username: "sender"},
			*recipient,
		},
		CreatedAt: time.Now(),
	}

	dms := &mockDMChannelRepo{
		GetOrCreateDMFn: func(_ context.Context, user1ID, user2ID, newID int64) (*models.DMChannel, error) {
			// Return existing DM — ID does NOT match newID.
			return existingDM, nil
		},
	}

	h := NewDMHandler(dms, users, sf, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels", strings.NewReader(`{"recipient_id":"4000"}`))
	setAuthUser(c, testUserID)

	err := h.CreateDM(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Existing DM should NOT dispatch any events.
	if len(gw.events) != 0 {
		t.Fatalf("expected 0 gateway events for existing DM, got %d", len(gw.events))
	}
}

func TestListDMs(t *testing.T) {
	gw := &mockGateway{}
	sf := testSnowflake()
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

	h := NewDMHandler(dms, users, sf, gw)

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

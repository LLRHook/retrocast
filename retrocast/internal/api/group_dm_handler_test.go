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
)

// ---------------------------------------------------------------------------
// Shared test constants and helpers for group DM handler tests
// ---------------------------------------------------------------------------

const (
	testRecipient2ID    int64 = 4001
	testRecipient3ID    int64 = 4002
	testGroupDMID       int64 = 8000
)

func newGroupDMChannel(id, ownerID int64, recipientIDs ...int64) *models.DMChannel {
	owner := ownerID
	recipients := make([]models.User, 0, len(recipientIDs)+1)
	recipients = append(recipients, models.User{ID: ownerID, Username: "owner"})
	for _, rid := range recipientIDs {
		recipients = append(recipients, models.User{ID: rid, Username: "member"})
	}
	return &models.DMChannel{
		ID:         id,
		Type:       models.DMTypeGroupDM,
		OwnerID:    &owner,
		Recipients: recipients,
		CreatedAt:  time.Now(),
	}
}

// multiUserRepo returns a mockUserRepo that finds users by a set of known IDs.
func multiUserRepo(knownIDs ...int64) *mockUserRepo {
	known := make(map[int64]bool, len(knownIDs))
	for _, id := range knownIDs {
		known[id] = true
	}
	return &mockUserRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.User, error) {
			if known[id] {
				return &models.User{
					ID:        id,
					Username:  "user",
					CreatedAt: time.Now(),
				}, nil
			}
			return nil, nil
		},
	}
}

// ---------------------------------------------------------------------------
// CreateGroupDM tests (POST /users/@me/channels with recipient_ids)
// ---------------------------------------------------------------------------

func TestCreateGroupDM_Success(t *testing.T) {
	gw := &mockGateway{}
	users := multiUserRepo(testRecipientID, testRecipient2ID)

	groupDM := newGroupDMChannel(0, testUserID, testRecipientID, testRecipient2ID)

	dms := &mockDMChannelRepo{
		CreateFn: func(_ context.Context, dm *models.DMChannel) error {
			return nil
		},
		GetByIDFn: func(_ context.Context, id int64) (*models.DMChannel, error) {
			groupDM.ID = id
			return groupDM, nil
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels",
		strings.NewReader(`{"recipient_ids":["4000","4001"]}`))
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
	if result.Type != models.DMTypeGroupDM {
		t.Fatalf("expected type %d, got %d", models.DMTypeGroupDM, result.Type)
	}
	if result.OwnerID == nil {
		t.Fatal("expected owner_id to be set")
	}

	// All 3 members should receive CHANNEL_CREATE.
	if len(gw.events) != 3 {
		t.Fatalf("expected 3 gateway events, got %d", len(gw.events))
	}
	for _, ev := range gw.events {
		if ev.Event != gateway.EventChannelCreate {
			t.Fatalf("expected CHANNEL_CREATE event, got %s", ev.Event)
		}
	}
}

func TestCreateGroupDM_EmptyRecipientIDs(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}
	dms := &mockDMChannelRepo{}

	h := newDMHandler(dms, users, gw)

	// Empty recipient_ids array — should fall through to 1-on-1 path with empty recipient_id.
	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels",
		strings.NewReader(`{"recipient_ids":[]}`))
	setAuthUser(c, testUserID)

	_ = h.CreateDM(c)
	// With empty recipient_ids, falls through to 1-on-1 DM with empty recipient_id -> bad request.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateGroupDM_TooManyRecipients(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}
	dms := &mockDMChannelRepo{}

	h := newDMHandler(dms, users, gw)

	// 10 recipient IDs (max is 9 since owner is the 10th).
	ids := `["1","2","3","4","5","6","7","8","9","10"]`
	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels",
		strings.NewReader(`{"recipient_ids":`+ids+`}`))
	setAuthUser(c, testUserID)

	_ = h.CreateDM(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "INVALID_RECIPIENTS" {
		t.Fatalf("expected error code INVALID_RECIPIENTS, got %s", errResp.Error.Code)
	}
}

func TestCreateGroupDM_DuplicateRecipient(t *testing.T) {
	gw := &mockGateway{}
	users := multiUserRepo(testRecipientID)
	dms := &mockDMChannelRepo{}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels",
		strings.NewReader(`{"recipient_ids":["4000","4000"]}`))
	setAuthUser(c, testUserID)

	_ = h.CreateDM(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "INVALID_RECIPIENTS" {
		t.Fatalf("expected error code INVALID_RECIPIENTS, got %s", errResp.Error.Code)
	}
}

func TestCreateGroupDM_SelfInRecipients(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}
	dms := &mockDMChannelRepo{}

	h := newDMHandler(dms, users, gw)

	// Include self (testUserID=3000) in recipient_ids.
	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels",
		strings.NewReader(`{"recipient_ids":["3000","4000"]}`))
	setAuthUser(c, testUserID)

	_ = h.CreateDM(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "INVALID_RECIPIENTS" {
		t.Fatalf("expected error code INVALID_RECIPIENTS, got %s", errResp.Error.Code)
	}
}

func TestCreateGroupDM_RecipientNotFound(t *testing.T) {
	gw := &mockGateway{}
	// Only testRecipientID exists, testRecipient2ID does not.
	users := multiUserRepo(testRecipientID)
	dms := &mockDMChannelRepo{}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels",
		strings.NewReader(`{"recipient_ids":["4000","4001"]}`))
	setAuthUser(c, testUserID)

	_ = h.CreateDM(c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateGroupDM_RepoError(t *testing.T) {
	gw := &mockGateway{}
	users := multiUserRepo(testRecipientID, testRecipient2ID)

	dms := &mockDMChannelRepo{
		CreateFn: func(_ context.Context, dm *models.DMChannel) error {
			return errors.New("db write failed")
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPost, "/api/v1/users/@me/channels",
		strings.NewReader(`{"recipient_ids":["4000","4001"]}`))
	setAuthUser(c, testUserID)

	_ = h.CreateDM(c)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// AddGroupDMMember tests (PUT /channels/:id/recipients/:user_id)
// ---------------------------------------------------------------------------

func TestAddGroupDMMember_Success(t *testing.T) {
	gw := &mockGateway{}
	users := multiUserRepo(testRecipient2ID)

	groupDM := newGroupDMChannel(testGroupDMID, testUserID, testRecipientID)

	dms := &mockDMChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.DMChannel, error) {
			if id == testGroupDMID {
				return groupDM, nil
			}
			return nil, nil
		},
		IsRecipientFn: func(_ context.Context, channelID, userID int64) (bool, error) {
			return false, nil
		},
		GetRecipientIDsFn: func(_ context.Context, channelID int64) ([]int64, error) {
			return []int64{testUserID, testRecipientID}, nil
		},
		AddRecipientFn: func(_ context.Context, channelID, userID int64) error {
			return nil
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/8000/recipients/4001", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("8000", "4001")
	setAuthUser(c, testUserID)

	err := h.AddGroupDMMember(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAddGroupDMMember_NotOwner(t *testing.T) {
	gw := &mockGateway{}
	users := multiUserRepo(testRecipient2ID)

	groupDM := newGroupDMChannel(testGroupDMID, testUserID, testRecipientID)

	dms := &mockDMChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.DMChannel, error) {
			if id == testGroupDMID {
				return groupDM, nil
			}
			return nil, nil
		},
	}

	h := newDMHandler(dms, users, gw)

	// Caller is testRecipientID (not the owner).
	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/8000/recipients/4001", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("8000", "4001")
	setAuthUser(c, testRecipientID)

	_ = h.AddGroupDMMember(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "FORBIDDEN" {
		t.Fatalf("expected error code FORBIDDEN, got %s", errResp.Error.Code)
	}
}

func TestAddGroupDMMember_NotGroupDM(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}

	// Regular 1-on-1 DM.
	regularDM := newDMChannel(testDMChannelID)

	dms := &mockDMChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.DMChannel, error) {
			if id == testDMChannelID {
				return regularDM, nil
			}
			return nil, nil
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/7000/recipients/4001", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("7000", "4001")
	setAuthUser(c, testUserID)

	_ = h.AddGroupDMMember(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "INVALID_CHANNEL" {
		t.Fatalf("expected error code INVALID_CHANNEL, got %s", errResp.Error.Code)
	}
}

func TestAddGroupDMMember_AlreadyMember(t *testing.T) {
	gw := &mockGateway{}
	users := multiUserRepo(testRecipientID)

	groupDM := newGroupDMChannel(testGroupDMID, testUserID, testRecipientID)

	dms := &mockDMChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.DMChannel, error) {
			if id == testGroupDMID {
				return groupDM, nil
			}
			return nil, nil
		},
		IsRecipientFn: func(_ context.Context, channelID, userID int64) (bool, error) {
			return true, nil // already a member
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/8000/recipients/4000", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("8000", "4000")
	setAuthUser(c, testUserID)

	_ = h.AddGroupDMMember(c)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "ALREADY_MEMBER" {
		t.Fatalf("expected error code ALREADY_MEMBER, got %s", errResp.Error.Code)
	}
}

func TestAddGroupDMMember_MaxMembers(t *testing.T) {
	gw := &mockGateway{}
	users := multiUserRepo(testRecipient2ID)

	groupDM := newGroupDMChannel(testGroupDMID, testUserID, testRecipientID)

	dms := &mockDMChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.DMChannel, error) {
			if id == testGroupDMID {
				return groupDM, nil
			}
			return nil, nil
		},
		IsRecipientFn: func(_ context.Context, channelID, userID int64) (bool, error) {
			return false, nil
		},
		GetRecipientIDsFn: func(_ context.Context, channelID int64) ([]int64, error) {
			// Already at max (10 members).
			return make([]int64, 10), nil
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/8000/recipients/4001", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("8000", "4001")
	setAuthUser(c, testUserID)

	_ = h.AddGroupDMMember(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "MAX_MEMBERS" {
		t.Fatalf("expected error code MAX_MEMBERS, got %s", errResp.Error.Code)
	}
}

func TestAddGroupDMMember_ChannelNotFound(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}
	dms := &mockDMChannelRepo{}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/9999/recipients/4001", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("9999", "4001")
	setAuthUser(c, testUserID)

	_ = h.AddGroupDMMember(c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAddGroupDMMember_InvalidChannelID(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}
	dms := &mockDMChannelRepo{}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodPut, "/api/v1/channels/abc/recipients/4001", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("abc", "4001")
	setAuthUser(c, testUserID)

	_ = h.AddGroupDMMember(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// RemoveGroupDMMember tests (DELETE /channels/:id/recipients/:user_id)
// ---------------------------------------------------------------------------

func TestRemoveGroupDMMember_OwnerRemovesMember(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}

	groupDM := newGroupDMChannel(testGroupDMID, testUserID, testRecipientID, testRecipient2ID)

	dms := &mockDMChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.DMChannel, error) {
			if id == testGroupDMID {
				return groupDM, nil
			}
			return nil, nil
		},
		IsRecipientFn: func(_ context.Context, channelID, userID int64) (bool, error) {
			return true, nil
		},
		RemoveRecipientFn: func(_ context.Context, channelID, userID int64) error {
			return nil
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/channels/8000/recipients/4000", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("8000", "4000")
	setAuthUser(c, testUserID)

	err := h.RemoveGroupDMMember(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	// Should have CHANNEL_UPDATE for remaining members + CHANNEL_DELETE for removed user.
	hasDelete := false
	for _, ev := range gw.events {
		if ev.Event == gateway.EventChannelDelete && ev.UserID == testRecipientID {
			hasDelete = true
		}
	}
	if !hasDelete {
		t.Fatal("expected CHANNEL_DELETE event for removed user")
	}
}

func TestRemoveGroupDMMember_MemberLeaves(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}

	groupDM := newGroupDMChannel(testGroupDMID, testUserID, testRecipientID, testRecipient2ID)

	dms := &mockDMChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.DMChannel, error) {
			if id == testGroupDMID {
				return groupDM, nil
			}
			return nil, nil
		},
		IsRecipientFn: func(_ context.Context, channelID, userID int64) (bool, error) {
			return true, nil
		},
		RemoveRecipientFn: func(_ context.Context, channelID, userID int64) error {
			return nil
		},
	}

	h := newDMHandler(dms, users, gw)

	// Non-owner removes self (leaving).
	c, rec := newTestContext(http.MethodDelete, "/api/v1/channels/8000/recipients/4000", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("8000", "4000")
	setAuthUser(c, testRecipientID) // caller == target

	err := h.RemoveGroupDMMember(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRemoveGroupDMMember_NonOwnerCannotRemoveOther(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}

	groupDM := newGroupDMChannel(testGroupDMID, testUserID, testRecipientID, testRecipient2ID)

	dms := &mockDMChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.DMChannel, error) {
			if id == testGroupDMID {
				return groupDM, nil
			}
			return nil, nil
		},
	}

	h := newDMHandler(dms, users, gw)

	// Non-owner tries to remove another member.
	c, rec := newTestContext(http.MethodDelete, "/api/v1/channels/8000/recipients/4001", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("8000", "4001")
	setAuthUser(c, testRecipientID)

	_ = h.RemoveGroupDMMember(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Code != "FORBIDDEN" {
		t.Fatalf("expected error code FORBIDDEN, got %s", errResp.Error.Code)
	}
}

func TestRemoveGroupDMMember_NotGroupDM(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}

	regularDM := newDMChannel(testDMChannelID)

	dms := &mockDMChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.DMChannel, error) {
			if id == testDMChannelID {
				return regularDM, nil
			}
			return nil, nil
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/channels/7000/recipients/4000", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("7000", "4000")
	setAuthUser(c, testUserID)

	_ = h.RemoveGroupDMMember(c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRemoveGroupDMMember_ChannelNotFound(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}
	dms := &mockDMChannelRepo{}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/channels/9999/recipients/4000", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("9999", "4000")
	setAuthUser(c, testUserID)

	_ = h.RemoveGroupDMMember(c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRemoveGroupDMMember_CallerNotMember(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}

	groupDM := newGroupDMChannel(testGroupDMID, testUserID, testRecipientID)

	callerID := int64(9999)
	dms := &mockDMChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.DMChannel, error) {
			if id == testGroupDMID {
				// Temporarily make caller the owner to pass the owner check,
				// but they're not a recipient.
				gd := *groupDM
				gd.OwnerID = &callerID
				return &gd, nil
			}
			return nil, nil
		},
		IsRecipientFn: func(_ context.Context, channelID, userID int64) (bool, error) {
			if userID == callerID {
				return false, nil
			}
			return true, nil
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/channels/8000/recipients/4000", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("8000", "4000")
	setAuthUser(c, callerID)

	_ = h.RemoveGroupDMMember(c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRemoveGroupDMMember_TargetNotMember(t *testing.T) {
	gw := &mockGateway{}
	users := &mockUserRepo{}

	groupDM := newGroupDMChannel(testGroupDMID, testUserID, testRecipientID)

	dms := &mockDMChannelRepo{
		GetByIDFn: func(_ context.Context, id int64) (*models.DMChannel, error) {
			if id == testGroupDMID {
				return groupDM, nil
			}
			return nil, nil
		},
		IsRecipientFn: func(_ context.Context, channelID, userID int64) (bool, error) {
			if userID == testUserID {
				return true, nil // caller is member
			}
			return false, nil // target is not member
		},
	}

	h := newDMHandler(dms, users, gw)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/channels/8000/recipients/4001", nil)
	c.SetParamNames("id", "user_id")
	c.SetParamValues("8000", "4001")
	setAuthUser(c, testUserID)

	_ = h.RemoveGroupDMMember(c)
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

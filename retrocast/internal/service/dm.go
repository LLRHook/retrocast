package service

import (
	"context"
	"strconv"
	"time"

	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/snowflake"
)

const maxGroupDMRecipients = 10

// DMService handles DM channel business logic.
type DMService struct {
	dms     database.DMChannelRepository
	users   database.UserRepository
	sf      *snowflake.Generator
	gateway gateway.Dispatcher
}

// NewDMService creates a DMService.
func NewDMService(
	dms database.DMChannelRepository,
	users database.UserRepository,
	sf *snowflake.Generator,
	gw gateway.Dispatcher,
) *DMService {
	return &DMService{
		dms:     dms,
		users:   users,
		sf:      sf,
		gateway: gw,
	}
}

// CreateDM creates or retrieves a DM channel between two users.
func (s *DMService) CreateDM(ctx context.Context, userID int64, recipientIDStr string) (*models.DMChannel, error) {
	recipientID, err := strconv.ParseInt(recipientIDStr, 10, 64)
	if err != nil || recipientID == 0 {
		return nil, BadRequest("INVALID_RECIPIENT", "invalid recipient_id")
	}

	if recipientID == userID {
		return nil, BadRequest("INVALID_RECIPIENT", "cannot create DM with yourself")
	}

	recipient, err := s.users.GetByID(ctx, recipientID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if recipient == nil {
		return nil, NotFound("NOT_FOUND", "recipient not found")
	}

	newID := s.sf.Generate().Int64()
	dm, err := s.dms.GetOrCreateDM(ctx, userID, recipientID, newID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	if dm.ID == newID {
		s.gateway.DispatchToUser(userID, gateway.EventChannelCreate, dm)
		s.gateway.DispatchToUser(recipientID, gateway.EventChannelCreate, dm)
	}

	return dm, nil
}

// CreateGroupDM creates a new group DM channel with the given recipients.
func (s *DMService) CreateGroupDM(ctx context.Context, userID int64, recipientIDStrs []string) (*models.DMChannel, error) {
	if len(recipientIDStrs) < 1 {
		return nil, BadRequest("INVALID_RECIPIENTS", "at least 1 recipient is required")
	}
	if len(recipientIDStrs) > maxGroupDMRecipients-1 {
		return nil, BadRequest("INVALID_RECIPIENTS", "too many recipients (max 9)")
	}

	recipientIDs := make([]int64, 0, len(recipientIDStrs))
	seen := map[int64]bool{userID: true}
	for _, idStr := range recipientIDStrs {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id == 0 {
			return nil, BadRequest("INVALID_RECIPIENTS", "invalid recipient_id")
		}
		if seen[id] {
			return nil, BadRequest("INVALID_RECIPIENTS", "duplicate recipient_id")
		}
		seen[id] = true
		recipientIDs = append(recipientIDs, id)
	}

	// Validate all recipients exist.
	for _, rid := range recipientIDs {
		u, err := s.users.GetByID(ctx, rid)
		if err != nil {
			return nil, Internal("INTERNAL", "internal server error")
		}
		if u == nil {
			return nil, NotFound("NOT_FOUND", "recipient not found")
		}
	}

	// Build recipient list including the owner.
	allIDs := append([]int64{userID}, recipientIDs...)
	recipients := make([]models.User, len(allIDs))
	for i, id := range allIDs {
		recipients[i] = models.User{ID: id}
	}

	newID := s.sf.Generate().Int64()
	ownerID := userID
	dm := &models.DMChannel{
		ID:         newID,
		Type:       models.DMTypeGroupDM,
		OwnerID:    &ownerID,
		Recipients: recipients,
		CreatedAt:  time.Now(),
	}

	if err := s.dms.Create(ctx, dm); err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	// Re-fetch to get full user data.
	dm, err := s.dms.GetByID(ctx, newID)
	if err != nil || dm == nil {
		return nil, Internal("INTERNAL", "internal server error")
	}

	// Notify all members.
	for _, id := range allIDs {
		s.gateway.DispatchToUser(id, gateway.EventChannelCreate, dm)
	}

	return dm, nil
}

// AddGroupDMMember adds a user to a group DM. Only the owner can add members.
func (s *DMService) AddGroupDMMember(ctx context.Context, callerID, channelID, userID int64) error {
	dm, err := s.dms.GetByID(ctx, channelID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if dm == nil {
		return NotFound("NOT_FOUND", "channel not found")
	}
	if dm.Type != models.DMTypeGroupDM {
		return BadRequest("INVALID_CHANNEL", "not a group DM")
	}
	if dm.OwnerID == nil || *dm.OwnerID != callerID {
		return Forbidden("FORBIDDEN", "only the group owner can add members")
	}

	// Check target user exists.
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if u == nil {
		return NotFound("NOT_FOUND", "user not found")
	}

	// Check not already a member.
	isMember, err := s.dms.IsRecipient(ctx, channelID, userID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if isMember {
		return Conflict("ALREADY_MEMBER", "user is already a member")
	}

	// Check member count limit.
	memberIDs, err := s.dms.GetRecipientIDs(ctx, channelID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if len(memberIDs) >= maxGroupDMRecipients {
		return BadRequest("MAX_MEMBERS", "group DM is full (max 10 members)")
	}

	if err := s.dms.AddRecipient(ctx, channelID, userID); err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	// Re-fetch to get updated channel.
	dm, err = s.dms.GetByID(ctx, channelID)
	if err != nil || dm == nil {
		return Internal("INTERNAL", "internal server error")
	}

	// Notify all current members (including the new one).
	for _, r := range dm.Recipients {
		s.gateway.DispatchToUser(r.ID, gateway.EventChannelUpdate, dm)
	}

	return nil
}

// RemoveGroupDMMember removes a user from a group DM.
// The owner can remove anyone. Non-owners can only remove themselves (leave).
func (s *DMService) RemoveGroupDMMember(ctx context.Context, callerID, channelID, userID int64) error {
	dm, err := s.dms.GetByID(ctx, channelID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if dm == nil {
		return NotFound("NOT_FOUND", "channel not found")
	}
	if dm.Type != models.DMTypeGroupDM {
		return BadRequest("INVALID_CHANNEL", "not a group DM")
	}

	// Permission check: owner can remove anyone, non-owner can only leave.
	isOwner := dm.OwnerID != nil && *dm.OwnerID == callerID
	if callerID != userID && !isOwner {
		return Forbidden("FORBIDDEN", "only the group owner can remove members")
	}

	// Check caller is a member.
	callerIsMember, err := s.dms.IsRecipient(ctx, channelID, callerID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if !callerIsMember {
		return Forbidden("FORBIDDEN", "you are not a member of this group")
	}

	// Check target is a member.
	isMember, err := s.dms.IsRecipient(ctx, channelID, userID)
	if err != nil {
		return Internal("INTERNAL", "internal server error")
	}
	if !isMember {
		return NotFound("NOT_FOUND", "user is not a member")
	}

	if err := s.dms.RemoveRecipient(ctx, channelID, userID); err != nil {
		return Internal("INTERNAL", "internal server error")
	}

	// Re-fetch to get updated channel.
	dm, err = s.dms.GetByID(ctx, channelID)
	if err != nil || dm == nil {
		return Internal("INTERNAL", "internal server error")
	}

	// Notify remaining members.
	for _, r := range dm.Recipients {
		s.gateway.DispatchToUser(r.ID, gateway.EventChannelUpdate, dm)
	}
	// Notify the removed user.
	s.gateway.DispatchToUser(userID, gateway.EventChannelDelete, dm)

	return nil
}

// ListDMs returns all DM channels for the user.
func (s *DMService) ListDMs(ctx context.Context, userID int64) ([]models.DMChannel, error) {
	channels, err := s.dms.GetByUserID(ctx, userID)
	if err != nil {
		return nil, Internal("INTERNAL", "internal server error")
	}
	if channels == nil {
		channels = []models.DMChannel{}
	}
	return channels, nil
}

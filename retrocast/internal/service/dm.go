package service

import (
	"context"
	"strconv"

	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/snowflake"
)

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

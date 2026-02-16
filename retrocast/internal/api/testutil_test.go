package api

import (
	"context"
	"io"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/snowflake"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestContext(method, path string, body io.Reader) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func setAuthUser(c echo.Context, userID int64) {
	c.Set("user_id", userID)
}

func testSnowflake() *snowflake.Generator {
	sf, _ := snowflake.NewGenerator(1, 1)
	return sf
}

// ---------------------------------------------------------------------------
// Mock gateway dispatcher
// ---------------------------------------------------------------------------

type dispatchedEvent struct {
	GuildID      int64
	UserID       int64
	ExceptUserID int64
	Event        string
	Data         any
}

type mockGateway struct {
	mu     sync.Mutex
	events []dispatchedEvent
}

func (m *mockGateway) DispatchToGuild(guildID int64, event string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, dispatchedEvent{GuildID: guildID, Event: event, Data: data})
}

func (m *mockGateway) DispatchToUser(userID int64, event string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, dispatchedEvent{UserID: userID, Event: event, Data: data})
}

func (m *mockGateway) DispatchToGuildExcept(guildID int64, exceptUserID int64, event string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, dispatchedEvent{GuildID: guildID, ExceptUserID: exceptUserID, Event: event, Data: data})
}

func (m *mockGateway) SubscribeToGuild(userID, guildID int64) {}

func (m *mockGateway) UnsubscribeFromGuild(userID, guildID int64) {}

// ---------------------------------------------------------------------------
// Mock repositories
// ---------------------------------------------------------------------------

// mockUserRepo implements database.UserRepository.
type mockUserRepo struct {
	CreateFn      func(ctx context.Context, user *models.User) error
	GetByIDFn     func(ctx context.Context, id int64) (*models.User, error)
	GetByUsernameFn func(ctx context.Context, username string) (*models.User, error)
	UpdateFn      func(ctx context.Context, user *models.User) error
	DeleteFn      func(ctx context.Context, id int64) error
}

func (m *mockUserRepo) Create(ctx context.Context, user *models.User) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, user)
	}
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	if m.GetByUsernameFn != nil {
		return m.GetByUsernameFn(ctx, username)
	}
	return nil, nil
}

func (m *mockUserRepo) Update(ctx context.Context, user *models.User) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, user)
	}
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id int64) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

// mockGuildRepo implements database.GuildRepository.
type mockGuildRepo struct {
	CreateFn    func(ctx context.Context, guild *models.Guild) error
	GetByIDFn   func(ctx context.Context, id int64) (*models.Guild, error)
	UpdateFn    func(ctx context.Context, guild *models.Guild) error
	DeleteFn    func(ctx context.Context, id int64) error
	GetByUserIDFn func(ctx context.Context, userID int64) ([]models.Guild, error)
}

func (m *mockGuildRepo) Create(ctx context.Context, guild *models.Guild) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, guild)
	}
	return nil
}

func (m *mockGuildRepo) GetByID(ctx context.Context, id int64) (*models.Guild, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockGuildRepo) Update(ctx context.Context, guild *models.Guild) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, guild)
	}
	return nil
}

func (m *mockGuildRepo) Delete(ctx context.Context, id int64) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *mockGuildRepo) GetByUserID(ctx context.Context, userID int64) ([]models.Guild, error) {
	if m.GetByUserIDFn != nil {
		return m.GetByUserIDFn(ctx, userID)
	}
	return nil, nil
}

// mockChannelRepo implements database.ChannelRepository.
type mockChannelRepo struct {
	CreateFn     func(ctx context.Context, channel *models.Channel) error
	GetByIDFn    func(ctx context.Context, id int64) (*models.Channel, error)
	GetByGuildIDFn func(ctx context.Context, guildID int64) ([]models.Channel, error)
	UpdateFn     func(ctx context.Context, channel *models.Channel) error
	DeleteFn     func(ctx context.Context, id int64) error
}

func (m *mockChannelRepo) Create(ctx context.Context, channel *models.Channel) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, channel)
	}
	return nil
}

func (m *mockChannelRepo) GetByID(ctx context.Context, id int64) (*models.Channel, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockChannelRepo) GetByGuildID(ctx context.Context, guildID int64) ([]models.Channel, error) {
	if m.GetByGuildIDFn != nil {
		return m.GetByGuildIDFn(ctx, guildID)
	}
	return nil, nil
}

func (m *mockChannelRepo) Update(ctx context.Context, channel *models.Channel) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, channel)
	}
	return nil
}

func (m *mockChannelRepo) Delete(ctx context.Context, id int64) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

// mockRoleRepo implements database.RoleRepository.
type mockRoleRepo struct {
	CreateFn     func(ctx context.Context, role *models.Role) error
	GetByIDFn    func(ctx context.Context, id int64) (*models.Role, error)
	GetByGuildIDFn func(ctx context.Context, guildID int64) ([]models.Role, error)
	UpdateFn     func(ctx context.Context, role *models.Role) error
	DeleteFn     func(ctx context.Context, id int64) error
	GetByMemberFn func(ctx context.Context, guildID, userID int64) ([]models.Role, error)
}

func (m *mockRoleRepo) Create(ctx context.Context, role *models.Role) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, role)
	}
	return nil
}

func (m *mockRoleRepo) GetByID(ctx context.Context, id int64) (*models.Role, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockRoleRepo) GetByGuildID(ctx context.Context, guildID int64) ([]models.Role, error) {
	if m.GetByGuildIDFn != nil {
		return m.GetByGuildIDFn(ctx, guildID)
	}
	return nil, nil
}

func (m *mockRoleRepo) Update(ctx context.Context, role *models.Role) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, role)
	}
	return nil
}

func (m *mockRoleRepo) Delete(ctx context.Context, id int64) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *mockRoleRepo) GetByMember(ctx context.Context, guildID, userID int64) ([]models.Role, error) {
	if m.GetByMemberFn != nil {
		return m.GetByMemberFn(ctx, guildID, userID)
	}
	return nil, nil
}

// mockMemberRepo implements database.MemberRepository.
type mockMemberRepo struct {
	CreateFn         func(ctx context.Context, member *models.Member) error
	GetByGuildAndUserFn func(ctx context.Context, guildID, userID int64) (*models.Member, error)
	GetByGuildIDFn   func(ctx context.Context, guildID int64, limit, offset int) ([]models.Member, error)
	UpdateFn         func(ctx context.Context, member *models.Member) error
	DeleteFn         func(ctx context.Context, guildID, userID int64) error
	AddRoleFn        func(ctx context.Context, guildID, userID, roleID int64) error
	RemoveRoleFn     func(ctx context.Context, guildID, userID, roleID int64) error
}

func (m *mockMemberRepo) Create(ctx context.Context, member *models.Member) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, member)
	}
	return nil
}

func (m *mockMemberRepo) GetByGuildAndUser(ctx context.Context, guildID, userID int64) (*models.Member, error) {
	if m.GetByGuildAndUserFn != nil {
		return m.GetByGuildAndUserFn(ctx, guildID, userID)
	}
	return nil, nil
}

func (m *mockMemberRepo) GetByGuildID(ctx context.Context, guildID int64, limit, offset int) ([]models.Member, error) {
	if m.GetByGuildIDFn != nil {
		return m.GetByGuildIDFn(ctx, guildID, limit, offset)
	}
	return nil, nil
}

func (m *mockMemberRepo) Update(ctx context.Context, member *models.Member) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, member)
	}
	return nil
}

func (m *mockMemberRepo) Delete(ctx context.Context, guildID, userID int64) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, guildID, userID)
	}
	return nil
}

func (m *mockMemberRepo) AddRole(ctx context.Context, guildID, userID, roleID int64) error {
	if m.AddRoleFn != nil {
		return m.AddRoleFn(ctx, guildID, userID, roleID)
	}
	return nil
}

func (m *mockMemberRepo) RemoveRole(ctx context.Context, guildID, userID, roleID int64) error {
	if m.RemoveRoleFn != nil {
		return m.RemoveRoleFn(ctx, guildID, userID, roleID)
	}
	return nil
}

// mockMessageRepo implements database.MessageRepository.
type mockMessageRepo struct {
	CreateFn         func(ctx context.Context, msg *models.Message) error
	GetByIDFn        func(ctx context.Context, id int64) (*models.MessageWithAuthor, error)
	GetByChannelIDFn func(ctx context.Context, channelID int64, before *int64, limit int) ([]models.MessageWithAuthor, error)
	UpdateFn         func(ctx context.Context, msg *models.Message) error
	DeleteFn         func(ctx context.Context, id int64) error
	SearchMessagesFn func(ctx context.Context, guildID int64, query string, authorID *int64, before *time.Time, after *time.Time, limit int) ([]models.MessageWithAuthor, error)
}

func (m *mockMessageRepo) Create(ctx context.Context, msg *models.Message) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, msg)
	}
	return nil
}

func (m *mockMessageRepo) GetByID(ctx context.Context, id int64) (*models.MessageWithAuthor, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockMessageRepo) GetByChannelID(ctx context.Context, channelID int64, before *int64, limit int) ([]models.MessageWithAuthor, error) {
	if m.GetByChannelIDFn != nil {
		return m.GetByChannelIDFn(ctx, channelID, before, limit)
	}
	return nil, nil
}

func (m *mockMessageRepo) Update(ctx context.Context, msg *models.Message) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, msg)
	}
	return nil
}

func (m *mockMessageRepo) Delete(ctx context.Context, id int64) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *mockMessageRepo) SearchMessages(ctx context.Context, guildID int64, query string, authorID *int64, before *time.Time, after *time.Time, limit int) ([]models.MessageWithAuthor, error) {
	if m.SearchMessagesFn != nil {
		return m.SearchMessagesFn(ctx, guildID, query, authorID, before, after, limit)
	}
	return nil, nil
}

// mockInviteRepo implements database.InviteRepository.
type mockInviteRepo struct {
	CreateFn        func(ctx context.Context, invite *models.Invite) error
	GetByCodeFn     func(ctx context.Context, code string) (*models.Invite, error)
	GetByGuildIDFn  func(ctx context.Context, guildID int64) ([]models.Invite, error)
	IncrementUsesFn func(ctx context.Context, code string) error
	DeleteFn        func(ctx context.Context, code string) error
}

func (m *mockInviteRepo) Create(ctx context.Context, invite *models.Invite) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, invite)
	}
	return nil
}

func (m *mockInviteRepo) GetByCode(ctx context.Context, code string) (*models.Invite, error) {
	if m.GetByCodeFn != nil {
		return m.GetByCodeFn(ctx, code)
	}
	return nil, nil
}

func (m *mockInviteRepo) GetByGuildID(ctx context.Context, guildID int64) ([]models.Invite, error) {
	if m.GetByGuildIDFn != nil {
		return m.GetByGuildIDFn(ctx, guildID)
	}
	return nil, nil
}

func (m *mockInviteRepo) IncrementUses(ctx context.Context, code string) error {
	if m.IncrementUsesFn != nil {
		return m.IncrementUsesFn(ctx, code)
	}
	return nil
}

func (m *mockInviteRepo) Delete(ctx context.Context, code string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, code)
	}
	return nil
}

// mockChannelOverrideRepo implements database.ChannelOverrideRepository.
type mockChannelOverrideRepo struct {
	SetFn          func(ctx context.Context, override *models.ChannelOverride) error
	GetByChannelFn func(ctx context.Context, channelID int64) ([]models.ChannelOverride, error)
	DeleteFn       func(ctx context.Context, channelID, roleID int64) error
}

func (m *mockChannelOverrideRepo) Set(ctx context.Context, override *models.ChannelOverride) error {
	if m.SetFn != nil {
		return m.SetFn(ctx, override)
	}
	return nil
}

func (m *mockChannelOverrideRepo) GetByChannel(ctx context.Context, channelID int64) ([]models.ChannelOverride, error) {
	if m.GetByChannelFn != nil {
		return m.GetByChannelFn(ctx, channelID)
	}
	return nil, nil
}

func (m *mockChannelOverrideRepo) Delete(ctx context.Context, channelID, roleID int64) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, channelID, roleID)
	}
	return nil
}

// mockBanRepo implements database.BanRepository.
type mockBanRepo struct {
	CreateFn         func(ctx context.Context, ban *models.Ban) error
	GetByGuildAndUserFn func(ctx context.Context, guildID, userID int64) (*models.Ban, error)
	GetByGuildIDFn   func(ctx context.Context, guildID int64) ([]models.Ban, error)
	DeleteFn         func(ctx context.Context, guildID, userID int64) error
}

func (m *mockBanRepo) Create(ctx context.Context, ban *models.Ban) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, ban)
	}
	return nil
}

func (m *mockBanRepo) GetByGuildAndUser(ctx context.Context, guildID, userID int64) (*models.Ban, error) {
	if m.GetByGuildAndUserFn != nil {
		return m.GetByGuildAndUserFn(ctx, guildID, userID)
	}
	return nil, nil
}

func (m *mockBanRepo) GetByGuildID(ctx context.Context, guildID int64) ([]models.Ban, error) {
	if m.GetByGuildIDFn != nil {
		return m.GetByGuildIDFn(ctx, guildID)
	}
	return nil, nil
}

func (m *mockBanRepo) Delete(ctx context.Context, guildID, userID int64) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, guildID, userID)
	}
	return nil
}

// mockDMChannelRepo implements database.DMChannelRepository.
type mockDMChannelRepo struct {
	CreateFn          func(ctx context.Context, dm *models.DMChannel) error
	GetByIDFn         func(ctx context.Context, id int64) (*models.DMChannel, error)
	GetByUserIDFn     func(ctx context.Context, userID int64) ([]models.DMChannel, error)
	GetOrCreateDMFn   func(ctx context.Context, user1ID, user2ID, newID int64) (*models.DMChannel, error)
	AddRecipientFn    func(ctx context.Context, channelID, userID int64) error
	RemoveRecipientFn func(ctx context.Context, channelID, userID int64) error
	IsRecipientFn     func(ctx context.Context, channelID, userID int64) (bool, error)
	GetRecipientIDsFn func(ctx context.Context, channelID int64) ([]int64, error)
}

// mockReadStateRepo implements database.ReadStateRepository.
type mockReadStateRepo struct {
	UpsertFn              func(ctx context.Context, userID, channelID, lastMessageID int64) error
	GetByUserFn           func(ctx context.Context, userID int64) ([]models.ReadState, error)
	GetByUserAndChannelFn func(ctx context.Context, userID, channelID int64) (*models.ReadState, error)
	IncrementMentionCountFn func(ctx context.Context, userID, channelID int64) error
}

func (m *mockReadStateRepo) Upsert(ctx context.Context, userID, channelID, lastMessageID int64) error {
	if m.UpsertFn != nil {
		return m.UpsertFn(ctx, userID, channelID, lastMessageID)
	}
	return nil
}

func (m *mockReadStateRepo) GetByUser(ctx context.Context, userID int64) ([]models.ReadState, error) {
	if m.GetByUserFn != nil {
		return m.GetByUserFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockReadStateRepo) GetByUserAndChannel(ctx context.Context, userID, channelID int64) (*models.ReadState, error) {
	if m.GetByUserAndChannelFn != nil {
		return m.GetByUserAndChannelFn(ctx, userID, channelID)
	}
	return nil, nil
}

func (m *mockReadStateRepo) IncrementMentionCount(ctx context.Context, userID, channelID int64) error {
	if m.IncrementMentionCountFn != nil {
		return m.IncrementMentionCountFn(ctx, userID, channelID)
	}
	return nil
}

// mockReactionRepo implements database.ReactionRepository.
type mockReactionRepo struct {
	AddFn                func(ctx context.Context, messageID, userID int64, emoji string) error
	RemoveFn             func(ctx context.Context, messageID, userID int64, emoji string) error
	GetByMessageFn       func(ctx context.Context, messageID int64) ([]models.Reaction, error)
	GetCountsByMessageFn func(ctx context.Context, messageID, currentUserID int64) ([]models.ReactionCount, error)
	GetUsersByReactionFn func(ctx context.Context, messageID int64, emoji string, limit int) ([]int64, error)
}

func (m *mockReactionRepo) Add(ctx context.Context, messageID, userID int64, emoji string) error {
	if m.AddFn != nil {
		return m.AddFn(ctx, messageID, userID, emoji)
	}
	return nil
}

func (m *mockReactionRepo) Remove(ctx context.Context, messageID, userID int64, emoji string) error {
	if m.RemoveFn != nil {
		return m.RemoveFn(ctx, messageID, userID, emoji)
	}
	return nil
}

func (m *mockReactionRepo) GetByMessage(ctx context.Context, messageID int64) ([]models.Reaction, error) {
	if m.GetByMessageFn != nil {
		return m.GetByMessageFn(ctx, messageID)
	}
	return nil, nil
}

func (m *mockReactionRepo) GetCountsByMessage(ctx context.Context, messageID, currentUserID int64) ([]models.ReactionCount, error) {
	if m.GetCountsByMessageFn != nil {
		return m.GetCountsByMessageFn(ctx, messageID, currentUserID)
	}
	return nil, nil
}

func (m *mockReactionRepo) GetUsersByReaction(ctx context.Context, messageID int64, emoji string, limit int) ([]int64, error) {
	if m.GetUsersByReactionFn != nil {
		return m.GetUsersByReactionFn(ctx, messageID, emoji, limit)
	}
	return nil, nil
}

func (m *mockDMChannelRepo) Create(ctx context.Context, dm *models.DMChannel) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, dm)
	}
	return nil
}

func (m *mockDMChannelRepo) GetByID(ctx context.Context, id int64) (*models.DMChannel, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockDMChannelRepo) GetByUserID(ctx context.Context, userID int64) ([]models.DMChannel, error) {
	if m.GetByUserIDFn != nil {
		return m.GetByUserIDFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockDMChannelRepo) GetOrCreateDM(ctx context.Context, user1ID, user2ID, newID int64) (*models.DMChannel, error) {
	if m.GetOrCreateDMFn != nil {
		return m.GetOrCreateDMFn(ctx, user1ID, user2ID, newID)
	}
	return nil, nil
}

func (m *mockDMChannelRepo) AddRecipient(ctx context.Context, channelID, userID int64) error {
	if m.AddRecipientFn != nil {
		return m.AddRecipientFn(ctx, channelID, userID)
	}
	return nil
}

func (m *mockDMChannelRepo) RemoveRecipient(ctx context.Context, channelID, userID int64) error {
	if m.RemoveRecipientFn != nil {
		return m.RemoveRecipientFn(ctx, channelID, userID)
	}
	return nil
}

func (m *mockDMChannelRepo) IsRecipient(ctx context.Context, channelID, userID int64) (bool, error) {
	if m.IsRecipientFn != nil {
		return m.IsRecipientFn(ctx, channelID, userID)
	}
	return false, nil
}

func (m *mockDMChannelRepo) GetRecipientIDs(ctx context.Context, channelID int64) ([]int64, error) {
	if m.GetRecipientIDsFn != nil {
		return m.GetRecipientIDsFn(ctx, channelID)
	}
	return nil, nil
}

// mockVoiceStateRepo implements database.VoiceStateRepository.
type mockVoiceStateRepo struct {
	UpsertFn       func(ctx context.Context, state *models.VoiceState) error
	DeleteFn       func(ctx context.Context, guildID, userID int64) error
	GetByChannelFn func(ctx context.Context, channelID int64) ([]models.VoiceState, error)
	GetByGuildFn   func(ctx context.Context, guildID int64) ([]models.VoiceState, error)
	GetByUserFn    func(ctx context.Context, guildID, userID int64) (*models.VoiceState, error)
}

func (m *mockVoiceStateRepo) Upsert(ctx context.Context, state *models.VoiceState) error {
	if m.UpsertFn != nil {
		return m.UpsertFn(ctx, state)
	}
	return nil
}

func (m *mockVoiceStateRepo) Delete(ctx context.Context, guildID, userID int64) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, guildID, userID)
	}
	return nil
}

func (m *mockVoiceStateRepo) GetByChannel(ctx context.Context, channelID int64) ([]models.VoiceState, error) {
	if m.GetByChannelFn != nil {
		return m.GetByChannelFn(ctx, channelID)
	}
	return nil, nil
}

func (m *mockVoiceStateRepo) GetByGuild(ctx context.Context, guildID int64) ([]models.VoiceState, error) {
	if m.GetByGuildFn != nil {
		return m.GetByGuildFn(ctx, guildID)
	}
	return nil, nil
}

func (m *mockVoiceStateRepo) GetByUser(ctx context.Context, guildID, userID int64) (*models.VoiceState, error) {
	if m.GetByUserFn != nil {
		return m.GetByUserFn(ctx, guildID, userID)
	}
	return nil, nil
}

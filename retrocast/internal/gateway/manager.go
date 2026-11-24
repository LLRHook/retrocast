package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/models"
	"github.com/victorivanov/retrocast/internal/permissions"
	"github.com/victorivanov/retrocast/internal/redis"
)

const (
	replayBufferSize = 100
)

// Manager manages all active WebSocket connections and event routing.
type Manager struct {
	mu            sync.RWMutex
	offlineEpoch  map[int64]uint64
	connections   map[int64]map[*Connection]bool // userID → connection
	subscriptions map[int64]map[int64]bool       // guildID → set of userIDs
	sessions      map[string]*Connection         // sessionID → connection

	visibility map[[3]int64]bool
	checker    interface {
		RequireChannelPermission(context.Context, int64, int64, int64, permissions.Permission) error
	}

	tokens     *auth.TokenService
	guilds     database.GuildRepository
	readStates database.ReadStateRepository
	redis      *redis.Client
}

// NewManager creates a new gateway Manager.
func NewManager(
	tokens *auth.TokenService,
	guilds database.GuildRepository,
	readStates database.ReadStateRepository,
	redisClient *redis.Client,
) *Manager {
	return &Manager{
		connections:   make(map[int64]map[*Connection]bool),
		subscriptions: make(map[int64]map[int64]bool),
		sessions:      make(map[string]*Connection),
		visibility:    make(map[[3]int64]bool),
		offlineEpoch:  make(map[int64]uint64),
		tokens:        tokens,
		guilds:        guilds,
		readStates:    readStates,
		redis:         redisClient,
	}
}

func (m *Manager) SetPermissionChecker(p interface {
	RequireChannelPermission(context.Context, int64, int64, int64, permissions.Permission) error
}) {
	m.checker = p
}
func (m *Manager) register(c *Connection) { m.mu.Lock(); defer m.mu.Unlock(); m.registerLocked(c) }
func (m *Manager) registerLocked(c *Connection) {
	m.pruneLocked()
	if m.connections[c.UserID] == nil {
		m.connections[c.UserID] = make(map[*Connection]bool)
	}
	m.offlineEpoch[c.UserID]++
	m.connections[c.UserID][c] = true
	m.sessions[c.SessionID] = c
	if c.guildIDs == nil {
		c.guildIDs = make(map[int64]bool)
	}
}
func (m *Manager) pruneLocked() {
	for id, c := range m.sessions {
		if !c.expires.IsZero() && time.Now().After(c.expires) {
			delete(m.sessions, id)
		}
	}
}
func (m *Manager) unregister(c *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	conns := m.connections[c.UserID]
	if !conns[c] {
		return
	}
	delete(conns, c)
	c.expires = time.Now().Add(5 * time.Minute)
	if len(conns) > 0 {
		return
	}
	delete(m.connections, c.UserID)
	var guilds []int64
	for id, members := range m.subscriptions {
		if members[c.UserID] {
			guilds = append(guilds, id)
		}
		delete(members, c.UserID)
		if len(members) == 0 {
			delete(m.subscriptions, id)
		}
	}
	m.offlineEpoch[c.UserID]++
	go m.clearPresenceWithGrace(c.UserID, guilds, m.offlineEpoch[c.UserID])
}
func (m *Manager) clearPresenceWithGrace(userID int64, guilds []int64, epoch uint64) {
	time.Sleep(10 * time.Second)
	m.mu.Lock()
	if len(m.connections[userID]) > 0 || m.offlineEpoch[userID] != epoch {
		m.mu.Unlock()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := m.redis.SetPresence(ctx, userID, "offline")
	cancel()
	if err == nil {
		for _, guildID := range guilds {
			for _, c := range m.sessions {
				if c.guildIDs[guildID] {
					m.sendLocked(c, EventPresenceUpdate, PresenceUpdateData{UserID: userID, Status: "offline"})
				}
			}
		}
	}
	m.mu.Unlock()
}
func (m *Manager) subscribeLocked(userID, guildID int64) {
	if len(m.connections[userID]) == 0 {
		return
	}
	if m.subscriptions[guildID] == nil {
		m.subscriptions[guildID] = make(map[int64]bool)
	}
	m.subscriptions[guildID][userID] = true
	for _, c := range m.sessions {
		if c.UserID == userID {
			if c.guildIDs == nil {
				c.guildIDs = make(map[int64]bool)
			}
			c.guildIDs[guildID] = true
		}
	}
}
func (m *Manager) SubscribeToGuild(userID, guildID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribeLocked(userID, guildID)
}
func (m *Manager) UnsubscribeFromGuild(userID, guildID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.subscriptions[guildID], userID)
	if len(m.subscriptions[guildID]) == 0 {
		delete(m.subscriptions, guildID)
	}
	for _, c := range m.sessions {
		if c.UserID == userID {
			delete(c.guildIDs, guildID)
			c.replay = nil
			c.replayFloor = c.sequence.Load()
		}
	}
	m.invalidateLocked(guildID)
}
func (m *Manager) InvalidateGuildPermissions(guildID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invalidateLocked(guildID)
}
func (m *Manager) invalidateLocked(guildID int64) {
	for key := range m.visibility {
		if key[0] == guildID {
			delete(m.visibility, key)
		}
	}
	for _, c := range m.sessions {
		if c.guildIDs[guildID] {
			c.replay = nil
			c.replayFloor = c.sequence.Load()
		}
	}
}
func (m *Manager) sendLocked(c *Connection, event string, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	seq := c.NextSequence()
	p := GatewayPayload{Op: OpDispatch, Data: raw, Sequence: &seq, Event: &event}
	c.replay = append(c.replay, p)
	if len(c.replay) > replayBufferSize {
		c.replayFloor = *c.replay[0].Sequence
		c.replay = c.replay[1:]
	}
	if m.connections[c.UserID][c] {
		c.SendPayload(p)
	}
}
func (m *Manager) DispatchToUser(userID int64, event string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked()
	for _, c := range m.sessions {
		if c.UserID == userID {
			m.sendLocked(c, event, data)
		}
	}
}
func (m *Manager) DispatchToGuild(guildID int64, event string, data interface{}) {
	m.DispatchToGuildExcept(guildID, 0, event, data)
}
func (m *Manager) DispatchToGuildExcept(guildID, exceptUserID int64, event string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked()
	switch event {
	case EventGuildRoleCreate, EventGuildRoleUpdate, EventGuildRoleDelete, EventGuildMemberUpdate, EventGuildMemberAdd, EventGuildMemberRemove, EventChannelCreate, EventChannelUpdate:
		m.invalidateLocked(guildID)
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	var channelID int64
	field := fields["channel_id"]
	if event == EventChannelCreate || event == EventChannelUpdate || event == EventChannelDelete {
		field = fields["id"]
	}
	if len(field) > 0 && string(field) != "null" {
		var str string
		if json.Unmarshal(field, &str) == nil {
			channelID, _ = strconv.ParseInt(str, 10, 64)
		} else {
			_ = json.Unmarshal(field, &channelID)
		}
	}
	for _, c := range m.sessions {
		if c.UserID == exceptUserID || !c.guildIDs[guildID] {
			continue
		}
		if channelID != 0 && m.checker != nil {
			key := [3]int64{guildID, channelID, c.UserID}
			visible, ok := m.visibility[key]
			if !ok && event != EventChannelDelete {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				visible = m.checker.RequireChannelPermission(ctx, guildID, channelID, c.UserID, permissions.PermViewChannel) == nil
				cancel()
				m.visibility[key] = visible
			}
			if !visible {
				continue
			}
		}
		m.sendLocked(c, event, data)
	}
	if event == EventChannelDelete {
		m.invalidateLocked(guildID)
	}
	if event == EventGuildDelete {
		delete(m.subscriptions, guildID)
		m.invalidateLocked(guildID)
		for _, c := range m.sessions {
			delete(c.guildIDs, guildID)
		}
	}
}
func (m *Manager) sendToGuildInternal(guildID int64, event Event) {
	m.DispatchToGuild(guildID, event.Name, event.Data)
}

// handleIdentify processes an IDENTIFY payload from a client.
func (m *Manager) handleIdentify(c *Connection, data json.RawMessage) {
	var identify IdentifyData
	if err := json.Unmarshal(data, &identify); err != nil {
		slog.Error("invalid identify data", "error", err)
		c.Close()
		return
	}

	claims, err := m.tokens.ValidateAccessToken(identify.Token)
	if err != nil {
		slog.Warn("invalid token in identify", "error", err)
		c.Close()
		return
	}

	c.UserID = claims.UserID
	c.SessionID = uuid.NewString()
	c.presenceStatus = "online"

	// Get user's guilds and subscribe.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	guilds, err := m.guilds.GetByUserID(ctx, c.UserID)
	if err != nil {
		slog.Error("failed to get guilds for user", "userID", c.UserID, "error", err)
		c.Close()
		return
	}

	guildIDs := make([]int64, len(guilds))
	for i, g := range guilds {
		guildIDs[i] = g.ID
	}

	// Set presence to online.
	if err := m.redis.SetPresence(ctx, c.UserID, "online"); err != nil {
		slog.Error("failed to set presence", "userID", c.UserID, "error", err)
	}

	// Fetch read states for READY payload.
	var readStates []models.ReadState
	if m.readStates != nil {
		rs, err := m.readStates.GetByUser(ctx, c.UserID)
		if err != nil {
			slog.Error("failed to get read states", "userID", c.UserID, "error", err)
		} else {
			readStates = rs
		}
	}

	// Serialize READY before this session becomes eligible for live dispatch.
	snapshot := m.presenceSnapshot(guildIDs)
	m.mu.Lock()
	m.registerLocked(c)
	for _, id := range guildIDs {
		m.subscribeLocked(c.UserID, id)
	}
	m.sendLocked(c, EventReady, ReadyData{
		SessionID:  c.SessionID,
		UserID:     c.UserID,
		Guilds:     guildIDs,
		ReadStates: readStates,
		Presences:  snapshot,
	})

	m.mu.Unlock()
	// Broadcast presence online to guild members.
	m.broadcastPresence(c.UserID, "online")
}

// handleResume preserves the session sequence or issues fresh READY on any gap.
func (m *Manager) handleResume(c *Connection, data json.RawMessage) {
	var r ResumeData
	if json.Unmarshal(data, &r) != nil {
		c.Close()
		return
	}
	claims, err := m.tokens.ValidateAccessToken(r.Token)
	if err != nil {
		c.Close()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	guilds, guildErr := m.guilds.GetByUserID(ctx, claims.UserID)
	cancel()
	if guildErr != nil {
		c.Close()
		return
	}
	m.mu.Lock()
	m.pruneLocked()
	old := m.sessions[r.SessionID]
	membershipChanged := false
	if old != nil {
		if len(guilds) != len(old.guildIDs) {
			membershipChanged = true
		}
		for _, g := range guilds {
			if !old.guildIDs[g.ID] {
				membershipChanged = true
			}
		}
	}
	if membershipChanged || old == nil || old.UserID != claims.UserID || r.Sequence < old.replayFloor || r.Sequence > old.sequence.Load() || m.connections[old.UserID][old] {
		m.mu.Unlock()
		raw, _ := json.Marshal(IdentifyData{Token: r.Token})
		m.handleIdentify(c, raw)
		return
	}
	c.presenceStatus = old.presenceStatus
	c.UserID = old.UserID
	c.SessionID = old.SessionID
	c.guildIDs = old.guildIDs
	c.replay = old.replay
	c.replayFloor = old.replayFloor
	c.sequence.Store(old.sequence.Load())
	m.registerLocked(c)
	for id := range c.guildIDs {
		m.subscribeLocked(c.UserID, id)
	}
	for _, p := range c.replay {
		if *p.Sequence > r.Sequence {
			c.SendPayload(p)
		}
	}
	status := c.presenceStatus
 m.mu.Unlock()
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	if status == "" {
		status = "online"
	}
	_ = m.redis.SetPresence(ctx, c.UserID, status)
	cancel()
	m.broadcastPresence(c.UserID, status)
}
func (m *Manager) refreshPresence(c *Connection) {
	if c.UserID == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.redis.RefreshPresence(ctx, c.UserID); err != nil {
		slog.Error("refresh presence", "error", err)
	}
}
func (m *Manager) presenceSnapshot(guildIDs []int64) []PresenceUpdateData {
	m.mu.RLock()
	users := make(map[int64]bool)
	for _, id := range guildIDs {
		for _, c := range m.sessions {
			if c.guildIDs[id] {
				users[c.UserID] = true
			}
		}
	}
	m.mu.RUnlock()
	result := make([]PresenceUpdateData, 0, len(users))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for uid := range users {
		status, err := m.redis.GetPresence(ctx, uid)
		if err == nil && status != "" && status != "offline" {
			result = append(result, PresenceUpdateData{UserID: uid, Status: status})
		}
	}
	return result
}

// handlePresenceUpdate processes a client presence update.
func (m *Manager) handlePresenceUpdate(c *Connection, data json.RawMessage) {
	var update ClientPresenceUpdate
	if err := json.Unmarshal(data, &update); err != nil {
		return
	}

	switch update.Status {
	case "online", "idle", "dnd", "invisible":
		// valid
	default:
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	redisStatus := update.Status
	if redisStatus == "invisible" {
		redisStatus = "offline"
	}
	if err := m.redis.SetPresence(ctx, c.UserID, redisStatus); err != nil {
		slog.Error("failed to update presence", "userID", c.UserID, "error", err)
		return
	}

	broadcastStatus := update.Status
	if broadcastStatus == "invisible" {
		broadcastStatus = "offline"
	}
	m.mu.Lock()
	for _, session := range m.sessions {
		if session.UserID == c.UserID {
			session.presenceStatus = broadcastStatus
		}
	}
	m.mu.Unlock()
	m.broadcastPresence(c.UserID, broadcastStatus)
}

// broadcastPresence sends a PRESENCE_UPDATE event to all guilds the user is in.
func (m *Manager) broadcastPresence(userID int64, status string) {
	event := Event{
		Name: EventPresenceUpdate,
		Data: PresenceUpdateData{
			UserID: userID,
			Status: status,
		},
	}

	m.mu.RLock()
	var guildIDs []int64
	for guildID, members := range m.subscriptions {
		if members[userID] {
			guildIDs = append(guildIDs, guildID)
		}
	}
	m.mu.RUnlock()

	for _, guildID := range guildIDs {
		m.sendToGuildInternal(guildID, event)
	}
}

// PrepareChannelDeletion snapshots authorization before overrides cascade away.
func (m *Manager) PrepareChannelDeletion(guildID, channelID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.checker == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, c := range m.sessions {
		if c.guildIDs[guildID] {
			key := [3]int64{guildID, channelID, c.UserID}
			m.visibility[key] = m.checker.RequireChannelPermission(ctx, guildID, channelID, c.UserID, permissions.PermViewChannel) == nil
		}
	}
}

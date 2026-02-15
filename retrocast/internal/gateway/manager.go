package gateway

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/redis"
)

const (
	replayBufferSize = 100
)

// Manager manages all active WebSocket connections and event routing.
type Manager struct {
	mu            sync.RWMutex
	connections   map[int64]*Connection            // userID → connection
	subscriptions map[int64]map[int64]bool          // guildID → set of userIDs
	sessions      map[string]*Connection            // sessionID → connection

	// Ring buffer per guild for session resume replay.
	replayMu     sync.RWMutex
	replayBuffer map[int64]*ringBuffer // guildID → ring buffer of events

	tokens   *auth.TokenService
	guilds   database.GuildRepository
	redis    *redis.Client
}

// NewManager creates a new gateway Manager.
func NewManager(
	tokens *auth.TokenService,
	guilds database.GuildRepository,
	redisClient *redis.Client,
) *Manager {
	return &Manager{
		connections:   make(map[int64]*Connection),
		subscriptions: make(map[int64]map[int64]bool),
		sessions:      make(map[string]*Connection),
		replayBuffer:  make(map[int64]*ringBuffer),
		tokens:        tokens,
		guilds:        guilds,
		redis:         redisClient,
	}
}

// register adds a connection to the manager.
func (m *Manager) register(c *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Disconnect existing connection for this user.
	if old, ok := m.connections[c.UserID]; ok {
		old.SendPayload(GatewayPayload{Op: OpReconnect})
		old.Close()
		delete(m.sessions, old.SessionID)
	}

	m.connections[c.UserID] = c
	m.sessions[c.SessionID] = c
}

// unregister removes a connection from the manager and cleans up subscriptions.
func (m *Manager) unregister(c *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.connections[c.UserID]; ok && existing == c {
		delete(m.connections, c.UserID)

		// Remove from all guild subscriptions.
		for guildID, members := range m.subscriptions {
			delete(members, c.UserID)
			if len(members) == 0 {
				delete(m.subscriptions, guildID)
			}
		}

		// Clear presence with grace period.
		go m.clearPresenceWithGrace(c.UserID)
	}

	delete(m.sessions, c.SessionID)
}

// clearPresenceWithGrace waits before setting offline, allowing reconnection.
func (m *Manager) clearPresenceWithGrace(userID int64) {
	time.Sleep(10 * time.Second)

	m.mu.RLock()
	_, stillConnected := m.connections[userID]
	m.mu.RUnlock()

	if stillConnected {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := m.redis.SetPresence(ctx, userID, "offline"); err != nil {
		log.Printf("gateway: failed to clear presence for user %d: %v", userID, err)
	}

	m.broadcastPresence(userID, "offline")
}

// subscribe adds a user to a guild's event subscription.
func (m *Manager) subscribe(userID, guildID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.subscriptions[guildID] == nil {
		m.subscriptions[guildID] = make(map[int64]bool)
	}
	m.subscriptions[guildID][userID] = true
}

// SubscribeToGuild adds a user to a guild's event subscription.
func (m *Manager) SubscribeToGuild(userID, guildID int64) {
	m.subscribe(userID, guildID)
}

// UnsubscribeFromGuild removes a user from a guild's event subscription.
func (m *Manager) UnsubscribeFromGuild(userID, guildID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if members, ok := m.subscriptions[guildID]; ok {
		delete(members, userID)
		if len(members) == 0 {
			delete(m.subscriptions, guildID)
		}
	}
}

// DispatchToUser sends a dispatch event to a specific connected user.
func (m *Manager) DispatchToUser(userID int64, event string, data interface{}) {
	m.mu.RLock()
	c, ok := m.connections[userID]
	m.mu.RUnlock()

	if ok {
		c.SendEvent(event, data)
	}
}

// DispatchToGuild sends a dispatch event to all users subscribed to a guild.
func (m *Manager) DispatchToGuild(guildID int64, event string, data interface{}) {
	m.mu.RLock()
	members := m.subscriptions[guildID]
	conns := make([]*Connection, 0, len(members))
	for userID := range members {
		if c, ok := m.connections[userID]; ok {
			conns = append(conns, c)
		}
	}
	m.mu.RUnlock()

	for _, c := range conns {
		c.SendEvent(event, data)
	}

	// Store in replay buffer.
	m.storeReplayEvent(guildID, Event{Name: event, Data: data})
}

// DispatchToGuildExcept sends a dispatch event to all guild subscribers except one user.
func (m *Manager) DispatchToGuildExcept(guildID int64, exceptUserID int64, event string, data interface{}) {
	m.mu.RLock()
	members := m.subscriptions[guildID]
	conns := make([]*Connection, 0, len(members))
	for userID := range members {
		if userID == exceptUserID {
			continue
		}
		if c, ok := m.connections[userID]; ok {
			conns = append(conns, c)
		}
	}
	m.mu.RUnlock()

	for _, c := range conns {
		c.SendEvent(event, data)
	}

	// Store in replay buffer.
	m.storeReplayEvent(guildID, Event{Name: event, Data: data})
}

// sendToGuildInternal sends an Event to all guild subscribers (internal use).
func (m *Manager) sendToGuildInternal(guildID int64, event Event) {
	m.mu.RLock()
	members := m.subscriptions[guildID]
	conns := make([]*Connection, 0, len(members))
	for userID := range members {
		if c, ok := m.connections[userID]; ok {
			conns = append(conns, c)
		}
	}
	m.mu.RUnlock()

	for _, c := range conns {
		c.SendEvent(event.Name, event.Data)
	}

	m.storeReplayEvent(guildID, event)
}

// handleIdentify processes an IDENTIFY payload from a client.
func (m *Manager) handleIdentify(c *Connection, data json.RawMessage) {
	var identify IdentifyData
	if err := json.Unmarshal(data, &identify); err != nil {
		log.Printf("gateway: invalid identify data: %v", err)
		c.Close()
		return
	}

	claims, err := m.tokens.ValidateAccessToken(identify.Token)
	if err != nil {
		log.Printf("gateway: invalid token in identify: %v", err)
		c.Close()
		return
	}

	c.UserID = claims.UserID
	c.SessionID = uuid.NewString()

	// Get user's guilds and subscribe.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	guilds, err := m.guilds.GetByUserID(ctx, c.UserID)
	if err != nil {
		log.Printf("gateway: failed to get guilds for user %d: %v", c.UserID, err)
		c.Close()
		return
	}

	m.register(c)

	guildIDs := make([]int64, len(guilds))
	for i, g := range guilds {
		guildIDs[i] = g.ID
		m.subscribe(c.UserID, g.ID)
	}

	// Set presence to online.
	if err := m.redis.SetPresence(ctx, c.UserID, "online"); err != nil {
		log.Printf("gateway: failed to set presence for user %d: %v", c.UserID, err)
	}

	// Send READY.
	c.SendEvent(EventReady, ReadyData{
		SessionID: c.SessionID,
		UserID:    c.UserID,
		Guilds:    guildIDs,
	})

	// Broadcast presence online to guild members.
	m.broadcastPresence(c.UserID, "online")
}

// handleResume processes a RESUME payload to replay missed events.
func (m *Manager) handleResume(c *Connection, data json.RawMessage) {
	var resume ResumeData
	if err := json.Unmarshal(data, &resume); err != nil {
		log.Printf("gateway: invalid resume data: %v", err)
		c.SendPayload(GatewayPayload{Op: OpReconnect})
		c.Close()
		return
	}

	claims, err := m.tokens.ValidateAccessToken(resume.Token)
	if err != nil {
		log.Printf("gateway: invalid token in resume: %v", err)
		c.Close()
		return
	}

	c.UserID = claims.UserID
	c.SessionID = resume.SessionID

	// Get user's guilds.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	guilds, err := m.guilds.GetByUserID(ctx, c.UserID)
	if err != nil {
		log.Printf("gateway: failed to get guilds on resume for user %d: %v", c.UserID, err)
		c.SendPayload(GatewayPayload{Op: OpReconnect})
		c.Close()
		return
	}

	m.register(c)

	for _, g := range guilds {
		m.subscribe(c.UserID, g.ID)

		// Replay missed events from ring buffer.
		m.replayMu.RLock()
		rb, ok := m.replayBuffer[g.ID]
		m.replayMu.RUnlock()

		if ok {
			events := rb.since(resume.Sequence)
			for _, ev := range events {
				c.SendEvent(ev.Name, ev.Data)
			}
		}
	}
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
		log.Printf("gateway: failed to update presence for user %d: %v", c.UserID, err)
		return
	}

	broadcastStatus := update.Status
	if broadcastStatus == "invisible" {
		broadcastStatus = "offline"
	}
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

// storeReplayEvent adds an event to the guild's replay ring buffer.
func (m *Manager) storeReplayEvent(guildID int64, event Event) {
	m.replayMu.Lock()
	defer m.replayMu.Unlock()

	rb, ok := m.replayBuffer[guildID]
	if !ok {
		rb = newRingBuffer(replayBufferSize)
		m.replayBuffer[guildID] = rb
	}
	rb.add(event)
}

// sequencedEvent pairs an event with its sequence number for replay.
type sequencedEvent struct {
	Sequence int64
	Event
}

// ringBuffer is a fixed-size circular buffer for replay events.
type ringBuffer struct {
	events []sequencedEvent
	size   int
	pos    int
	seq    int64
	full   bool
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{
		events: make([]sequencedEvent, size),
		size:   size,
	}
}

func (rb *ringBuffer) add(event Event) {
	rb.seq++
	rb.events[rb.pos] = sequencedEvent{Sequence: rb.seq, Event: event}
	rb.pos = (rb.pos + 1) % rb.size
	if rb.pos == 0 {
		rb.full = true
	}
}

// since returns all events with sequence > afterSeq.
func (rb *ringBuffer) since(afterSeq int64) []Event {
	var result []Event
	count := rb.size
	if !rb.full {
		count = rb.pos
	}

	start := 0
	if rb.full {
		start = rb.pos
	}

	for i := 0; i < count; i++ {
		idx := (start + i) % rb.size
		if rb.events[idx].Sequence > afterSeq {
			result = append(result, rb.events[idx].Event)
		}
	}
	return result
}

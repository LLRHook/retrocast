package api

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/service"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// pilotContractServer seeds a fresh database for each operation. No handler or
// response writer is stubbed; only object storage is an in-memory fixture.
func pilotContractServer(t *testing.T) (*echo.Echo, string, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL required")
	}
	admin, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("l40_contract_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close(); _, _ = admin.Exec(ctx, "DROP DATABASE "+name); admin.Close() })
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	files, _ := filepath.Glob("../../migrations/*.up.sql")
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		exec(string(b))
	}
	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	exec("INSERT INTO users(id,username,display_name,password_hash) VALUES(100,'owner','Owner',$1),(200,'target','Target',$1),(300,'outsider','Other',$1)", hash)
	exec("INSERT INTO guilds(id,name,owner_id) VALUES(1,'Pilot',100),(2,'Other',200)")
	exec("INSERT INTO members(guild_id,user_id) VALUES(1,100),(1,200),(2,100),(2,200)")
	exec("INSERT INTO channels(id,guild_id,name,type,position) VALUES(10,1,'general',0,0),(11,1,'Voice',2,1)")
	exec("INSERT INTO roles(id,guild_id,name,permissions,position,is_default) VALUES(30,1,'@everyone',0,0,true),(31,1,'Helper',0,1,false)")
	exec("INSERT INTO messages(id,channel_id,author_id,content) VALUES(20,10,100,'hello')")
	exec("INSERT INTO invites(code,guild_id,creator_id) VALUES('pilot',1,100)")
	exec("INSERT INTO dm_channels(id,type,owner_id) VALUES(12,3,100)")
	exec("INSERT INTO dm_recipients(channel_id,user_id) VALUES(12,100),(12,200)")
	exec("INSERT INTO voice_states(guild_id,channel_id,user_id,session_id) VALUES(1,11,100,'pilot')")
	rdb := newTestRedis(t)
	tokenSvc := auth.NewTokenService("pilot-contract-fixture")
	sf := testSnowflake()
	if err := rdb.StoreRefreshToken(ctx, "pilot-refresh", 100, time.Hour); err != nil {
		t.Fatal(err)
	}
	users := database.NewUserRepository(pool)
	guilds := database.NewGuildRepository(pool)
	channels := database.NewChannelRepository(pool)
	roles := database.NewRoleRepository(pool)
	members := database.NewMemberRepository(pool)
	messages := database.NewMessageRepository(pool)
	invites := database.NewInviteRepository(pool)
	overrides := database.NewChannelOverrideRepository(pool)
	attachments := database.NewAttachmentRepository(pool)
	bans := database.NewBanRepository(pool)
	dmChannels := database.NewDMChannelRepository(pool)
	readStates := database.NewReadStateRepository(pool)
	reactions := database.NewReactionRepository(pool)
	voiceStates := database.NewVoiceStateRepository(pool)

	// --- Storage ---

	minioClient := &mockStorage{}

	// --- Gateway ---

	gwManager := gateway.NewManager(tokenSvc, guilds, readStates, rdb)

	// --- Services ---

	permChecker := service.NewPermissionChecker(guilds, members, roles, overrides)

	authSvc := service.NewAuthService(users, tokenSvc, rdb, sf)
	userSvc := service.NewUserService(users)
	guildSvc := service.NewGuildService(guilds, channels, members, roles, sf, gwManager, permChecker)
	channelSvc := service.NewChannelService(channels, members, sf, gwManager, permChecker)
	memberSvc := service.NewMemberService(members, guilds, roles, gwManager, permChecker)
	roleSvc := service.NewRoleService(guilds, roles, members, channels, overrides, sf, gwManager, permChecker)
	messageSvc := service.NewMessageService(messages, channels, dmChannels, sf, gwManager, permChecker)
	messageSvc.SetTypingStore(rdb)
	inviteSvc := service.NewInviteService(invites, guilds, members, bans, gwManager, permChecker)
	authSvc.ConfigureRegistration("open", inviteSvc)
	banSvc := service.NewBanService(guilds, members, roles, bans, gwManager, permChecker)
	dmSvc := service.NewDMService(dmChannels, users, sf, gwManager)
	uploadSvc := service.NewUploadService(attachments, channels, sf, minioClient, permChecker)
	readStateSvc := service.NewReadStateService(readStates, channels, dmChannels, permChecker)
	reactionSvc := service.NewReactionService(reactions, messages, channels, dmChannels, gwManager, permChecker)
	searchSvc := service.NewSearchService(messages, members, permChecker)
	messageSvc.SetFileStorage(minioClient)
	searchSvc.SetFileStorage(minioClient)
	voiceSvc := service.NewVoiceService(voiceStates, channels, users, gwManager, permChecker, "fixture-key", "fixture-secret")

	// --- Handlers ---

	authHandler := NewAuthHandler(authSvc)
	userHandler := NewUserHandler(userSvc)
	guildHandler := NewGuildHandler(guildSvc)
	channelHandler := NewChannelHandler(channelSvc)
	memberHandler := NewMemberHandler(memberSvc)
	roleHandler := NewRoleHandler(roleSvc)
	messageHandler := NewMessageHandler(messageSvc)
	inviteHandler := NewInviteHandler(inviteSvc)
	banHandler := NewBanHandler(banSvc)
	dmHandler := NewDMHandler(dmSvc)
	uploadHandler := NewUploadHandler(uploadSvc)
	readStateHandler := NewReadStateHandler(readStateSvc)
	reactionHandler := NewReactionHandler(reactionSvc)
	searchHandler := NewSearchHandler(searchSvc)
	voiceHandler := NewVoiceHandler(voiceSvc)

	deps := &Dependencies{
		Auth:         authHandler,
		Guilds:       guildHandler,
		Channels:     channelHandler,
		Members:      memberHandler,
		Users:        userHandler,
		Messages:     messageHandler,
		Invites:      inviteHandler,
		Roles:        roleHandler,
		Uploads:      uploadHandler,
		Bans:         banHandler,
		DMs:          dmHandler,
		ReadStates:   readStateHandler,
		Reactions:    reactionHandler,
		Search:       searchHandler,
		Voice:        voiceHandler,
		Gateway:      gwManager,
		TokenService: tokenSvc,
		Pool:         pool,
		Redis:        rdb,
	}

	old := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = old })
	e := echo.New()
	SetupRouter(e, deps)
	token, err := tokenSvc.GenerateAccessToken(100)
	if err != nil {
		t.Fatal(err)
	}
	return e, token, pool
}

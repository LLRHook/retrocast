package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/victorivanov/retrocast/internal/api"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/config"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	redisclient "github.com/victorivanov/retrocast/internal/redis"
	"github.com/victorivanov/retrocast/internal/snowflake"
	"github.com/victorivanov/retrocast/internal/storage"
)

func main() {
	cfg := config.Load()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})))
	ctx := context.Background()

	// --- Infrastructure ---

	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Run migrations
	m, err := migrate.New("file://migrations", cfg.DatabaseURL)
	if err != nil {
		slog.Error("migration init failed", "error", err)
		os.Exit(1)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")

	rdb, err := redisclient.NewClient(cfg.RedisURL)
	if err != nil {
		slog.Error("redis connection failed", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	sf, err := snowflake.NewGenerator(1, 1)
	if err != nil {
		slog.Error("snowflake generator failed", "error", err)
		os.Exit(1)
	}
	tokenSvc := auth.NewTokenService(cfg.JWTSecret)

	// --- Repositories ---

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

	// --- Storage ---

	minioClient, err := storage.NewMinIOClient(
		cfg.MinIOEndpoint, cfg.MinIOAccessKey, cfg.MinIOSecretKey, "retrocast",
	)
	if err != nil {
		slog.Error("minio connection failed", "error", err)
		os.Exit(1)
	}

	// --- Gateway ---

	gwManager := gateway.NewManager(tokenSvc, guilds, rdb)

	// --- Handlers ---

	guildHandler := api.NewGuildHandler(guilds, channels, members, roles, sf, gwManager)
	channelHandler := api.NewChannelHandler(channels, guilds, members, roles, sf, guildHandler.RequirePermission(), gwManager)
	memberHandler := api.NewMemberHandler(members, guilds, roles, guildHandler.RequirePermission(), gwManager)
	userHandler := api.NewUserHandler(users)
	authHandler := api.NewAuthHandler(users, tokenSvc, rdb, sf)
	messageHandler := api.NewMessageHandler(messages, channels, dmChannels, members, roles, guilds, overrides, sf, gwManager)
	dmHandler := api.NewDMHandler(dmChannels, users, sf, gwManager)
	inviteHandler := api.NewInviteHandler(invites, guilds, members, roles, bans, gwManager)
	banHandler := api.NewBanHandler(guilds, members, roles, bans, gwManager)
	roleHandler := api.NewRoleHandler(guilds, roles, members, channels, overrides, sf, gwManager)
	uploadHandler := api.NewUploadHandler(attachments, channels, members, roles, guilds, overrides, sf, minioClient)
	typingHandler := gateway.NewTypingHandler(channels, rdb, gwManager)

	deps := &api.Dependencies{
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
		Typing:       typingHandler,
		Gateway:      gwManager,
		TokenService: tokenSvc,
		Redis:        rdb,
	}

	// --- Echo ---

	e := echo.New()
	e.HidePort = true
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	api.SetupRouter(e, deps)

	// --- Start ---

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("retrocast starting", "addr", cfg.ServerAddr)
		if err := e.Start(cfg.ServerAddr); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-sigCtx.Done()
	slog.Info("shutting down")
	if err := e.Shutdown(context.Background()); err != nil {
		slog.Error("shutdown error", "error", err)
		os.Exit(1)
	}
}

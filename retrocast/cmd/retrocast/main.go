package main

import (
	"context"
	"errors"
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
	"github.com/victorivanov/retrocast/internal/permissions"
	redisclient "github.com/victorivanov/retrocast/internal/redis"
	"github.com/victorivanov/retrocast/internal/service"
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
	defer func() { _ = rdb.Close() }()

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
	readStates := database.NewReadStateRepository(pool)
	reactions := database.NewReactionRepository(pool)
	voiceStates := database.NewVoiceStateRepository(pool)

	// --- Storage ---

	minioClient, err := storage.NewMinIOClient(
		cfg.MinIOEndpoint, cfg.MinIOAccessKey, cfg.MinIOSecretKey, "retrocast",
	)
	if err != nil {
		slog.Error("minio connection failed", "error", err)
		os.Exit(1)
	}

	// --- Gateway ---

	gwManager := gateway.NewManager(tokenSvc, guilds, readStates, rdb)

	// --- Services ---

	permChecker := service.NewPermissionChecker(guilds, members, roles, overrides)
	gwManager.SetPermissionChecker(gateway.PermissionCheckerFunc(func(ctx context.Context, guildID, channelID, userID int64, perm permissions.Permission) error {
		err := permChecker.RequireChannelPermission(ctx, guildID, channelID, userID, perm)
		return gatewayPermissionResult(err)
	}))

	authSvc := service.NewAuthService(users, tokenSvc, rdb, sf)
	userSvc := service.NewUserService(users)
	guildSvc := service.NewGuildService(guilds, channels, members, roles, sf, gwManager, permChecker)
	channelSvc := service.NewChannelService(channels, members, sf, gwManager, permChecker)
	memberSvc := service.NewMemberService(members, guilds, roles, gwManager, permChecker)
	roleSvc := service.NewRoleService(guilds, roles, members, channels, overrides, sf, gwManager, permChecker)
	messageSvc := service.NewMessageService(messages, channels, dmChannels, sf, gwManager, permChecker)
	messageSvc.SetTypingStore(rdb)
	inviteSvc := service.NewInviteService(invites, guilds, members, bans, gwManager, permChecker)
	authSvc.ConfigureRegistration(cfg.RegistrationMode, inviteSvc)
	banSvc := service.NewBanService(guilds, members, roles, bans, gwManager, permChecker)
	dmSvc := service.NewDMService(dmChannels, users, sf, gwManager)
	uploadSvc := service.NewUploadService(attachments, channels, sf, minioClient, permChecker)
	readStateSvc := service.NewReadStateService(readStates, channels, dmChannels, permChecker)
	reactionSvc := service.NewReactionService(reactions, messages, channels, dmChannels, gwManager, permChecker)
	searchSvc := service.NewSearchService(messages, members, permChecker)
	messageSvc.SetFileStorage(minioClient)
	searchSvc.SetFileStorage(minioClient)
	voiceSvc := service.NewVoiceService(voiceStates, channels, users, gwManager, permChecker, cfg.LiveKitAPIKey, cfg.LiveKitAPISecret)

	// --- Handlers ---

	authHandler := api.NewAuthHandler(authSvc)
	userHandler := api.NewUserHandler(userSvc)
	guildHandler := api.NewGuildHandler(guildSvc)
	channelHandler := api.NewChannelHandler(channelSvc)
	memberHandler := api.NewMemberHandler(memberSvc)
	roleHandler := api.NewRoleHandler(roleSvc)
	messageHandler := api.NewMessageHandler(messageSvc)
	inviteHandler := api.NewInviteHandler(inviteSvc)
	banHandler := api.NewBanHandler(banSvc)
	dmHandler := api.NewDMHandler(dmSvc)
	uploadHandler := api.NewUploadHandler(uploadSvc)
	readStateHandler := api.NewReadStateHandler(readStateSvc)
	reactionHandler := api.NewReactionHandler(reactionSvc)
	searchHandler := api.NewSearchHandler(searchSvc)
	voiceHandler := api.NewVoiceHandler(voiceSvc)

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
		ReadStates:   readStateHandler,
		Reactions:    reactionHandler,
		Search:       searchHandler,
		Voice:        voiceHandler,
		Gateway:      gwManager,
		TokenService: tokenSvc,
		Pool:         pool,
		Redis:        rdb,
	}

	// --- Echo ---

	e := echo.New()
	e.HidePort = true
	e.HideBanner = true
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:     true,
		LogStatus:  true,
		LogMethod:  true,
		LogLatency: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			slog.Info("request",
				"method", v.Method,
				"uri", v.URI,
				"status", v.Status,
				"latency", v.Latency.String(),
			)
			return nil
		},
	}))
	e.Use(middleware.Recover())

	deps.AllowedOrigins = cfg.AllowedOrigins
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

// gatewayPermissionResult distinguishes authorization denials from lookup failures.
func gatewayPermissionResult(err error) error {
	if errors.Is(err, service.ErrForbidden) || errors.Is(err, service.ErrNotFound) {
		return gateway.ErrChannelHidden
	}
	return err
}

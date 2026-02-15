package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/victorivanov/retrocast/internal/api"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/config"
	"github.com/victorivanov/retrocast/internal/database"
	"github.com/victorivanov/retrocast/internal/gateway"
	redisclient "github.com/victorivanov/retrocast/internal/redis"
	"github.com/victorivanov/retrocast/internal/snowflake"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// --- Infrastructure ---

	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()

	rdb, err := redisclient.NewClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer rdb.Close()

	sf, err := snowflake.NewGenerator(1, 1)
	if err != nil {
		log.Fatalf("snowflake: %v", err)
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

	// --- Gateway ---

	gwManager := gateway.NewManager(tokenSvc, guilds, rdb)

	// --- Handlers ---

	guildHandler := api.NewGuildHandler(guilds, channels, members, roles, sf)
	channelHandler := api.NewChannelHandler(channels, guilds, members, roles, sf, guildHandler.RequirePermission())
	memberHandler := api.NewMemberHandler(members, guilds, roles, guildHandler.RequirePermission())
	userHandler := api.NewUserHandler(users)
	authHandler := api.NewAuthHandler(users, tokenSvc, rdb, sf)
	messageHandler := api.NewMessageHandler(messages, channels, members, roles, guilds, sf, gwManager)
	inviteHandler := api.NewInviteHandler(invites, guilds, members, roles, gwManager)
	roleHandler := api.NewRoleHandler(guilds, roles, members, channels, overrides, sf)
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
		log.Printf("retrocast starting on %s", cfg.ServerAddr)
		if err := e.Start(cfg.ServerAddr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-sigCtx.Done()
	log.Println("shutting down...")
	if err := e.Shutdown(context.Background()); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
}

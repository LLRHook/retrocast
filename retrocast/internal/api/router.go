package api

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/redis"
)

// Dependencies holds all handler instances and middleware for route wiring.
type Dependencies struct {
	Auth     *AuthHandler
	Guilds   *GuildHandler
	Channels *ChannelHandler
	Members  *MemberHandler
	Users    *UserHandler
	Messages *MessageHandler
	Invites  *InviteHandler
	Roles    *RoleHandler
	Typing   *gateway.TypingHandler
	Gateway  *gateway.Manager

	TokenService *auth.TokenService
	Redis        *redis.Client
}

// SetupRouter registers all API routes on the Echo instance.
func SetupRouter(e *echo.Echo, deps *Dependencies) {
	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// WebSocket gateway
	e.GET("/gateway", deps.Gateway.HandleWebSocket)

	v1 := e.Group("/api/v1")

	// Auth routes — no auth middleware, stricter rate limit
	authGroup := v1.Group("/auth",
		RateLimitMiddleware(deps.Redis, 5, time.Minute),
	)
	authGroup.POST("/register", deps.Auth.Register)
	authGroup.POST("/login", deps.Auth.Login)
	authGroup.POST("/refresh", deps.Auth.Refresh)

	// Public invite info — no auth required
	v1.GET("/invites/:code", deps.Invites.GetInvite)

	// Protected routes — require JWT auth + general rate limit
	authMw := deps.TokenService.Middleware()
	protected := v1.Group("", authMw,
		RateLimitMiddleware(deps.Redis, 50, time.Minute),
	)

	// Auth (protected)
	protected.POST("/auth/logout", deps.Auth.Logout)

	// Users
	protected.GET("/users/@me", deps.Users.GetMe)
	protected.PATCH("/users/@me", deps.Users.UpdateMe)
	protected.GET("/users/@me/guilds", deps.Guilds.ListMyGuilds)

	// Guilds
	protected.POST("/guilds", deps.Guilds.CreateGuild)
	protected.GET("/guilds/:id", deps.Guilds.GetGuild)
	protected.PATCH("/guilds/:id", deps.Guilds.UpdateGuild)
	protected.DELETE("/guilds/:id", deps.Guilds.DeleteGuild)

	// Channels
	protected.POST("/guilds/:id/channels", deps.Channels.CreateChannel)
	protected.GET("/guilds/:id/channels", deps.Channels.ListChannels)
	protected.GET("/channels/:id", deps.Channels.GetChannel)
	protected.PATCH("/channels/:id", deps.Channels.UpdateChannel)
	protected.DELETE("/channels/:id", deps.Channels.DeleteChannel)

	// Members
	protected.GET("/guilds/:id/members", deps.Members.ListMembers)
	protected.GET("/guilds/:id/members/:user_id", deps.Members.GetMember)
	protected.PATCH("/guilds/:id/members/:user_id", deps.Members.UpdateMember)
	protected.PATCH("/guilds/:id/members/@me", deps.Members.UpdateSelf)
	protected.DELETE("/guilds/:id/members/:user_id", deps.Members.KickMember)
	protected.DELETE("/guilds/:id/members/@me", deps.Members.LeaveGuild)

	// Roles
	protected.POST("/guilds/:id/roles", deps.Roles.CreateRole)
	protected.GET("/guilds/:id/roles", deps.Roles.ListRoles)
	protected.PATCH("/guilds/:id/roles/:role_id", deps.Roles.UpdateRole)
	protected.DELETE("/guilds/:id/roles/:role_id", deps.Roles.DeleteRole)
	protected.PUT("/guilds/:id/members/:user_id/roles/:role_id", deps.Roles.AssignRole)
	protected.DELETE("/guilds/:id/members/:user_id/roles/:role_id", deps.Roles.RemoveRole)

	// Channel permission overrides
	protected.PUT("/channels/:id/permissions/:role_id", deps.Roles.SetChannelOverride)
	protected.DELETE("/channels/:id/permissions/:role_id", deps.Roles.DeleteChannelOverride)

	// Messages
	protected.POST("/channels/:id/messages", deps.Messages.SendMessage)
	protected.GET("/channels/:id/messages", deps.Messages.GetMessages)
	protected.GET("/channels/:id/messages/:message_id", deps.Messages.GetMessage)
	protected.PATCH("/channels/:id/messages/:message_id", deps.Messages.EditMessage)
	protected.DELETE("/channels/:id/messages/:message_id", deps.Messages.DeleteMessage)

	// Typing
	protected.POST("/channels/:id/typing", deps.Typing.Handle)

	// Invites (protected)
	protected.POST("/guilds/:id/invites", deps.Invites.CreateInvite)
	protected.GET("/guilds/:id/invites", deps.Invites.ListInvites)
	protected.POST("/invites/:code", deps.Invites.AcceptInvite)
	protected.DELETE("/invites/:code", deps.Invites.RevokeInvite)
}

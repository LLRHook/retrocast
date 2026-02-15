package api

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/docs"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/gateway"
	"github.com/victorivanov/retrocast/internal/redis"
)

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Retrocast API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>html{box-sizing:border-box;overflow-y:scroll}*,*::before,*::after{box-sizing:inherit}body{margin:0;background:#fafafa}</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/docs/openapi.yaml",
      dom_id: "#swagger-ui",
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: "BaseLayout",
    });
  </script>
</body>
</html>`

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
	Uploads  *UploadHandler
	Bans     *BanHandler
	DMs      *DMHandler
	Typing   *gateway.TypingHandler
	Gateway  *gateway.Manager

	TokenService *auth.TokenService
	Pool         *pgxpool.Pool
	Redis        *redis.Client
}

// SetupRouter registers all API routes on the Echo instance.
func SetupRouter(e *echo.Echo, deps *Dependencies) {
	// Health check — deep: pings Postgres and Redis
	e.GET("/health", func(c echo.Context) error {
		ctx := c.Request().Context()

		if err := deps.Pool.Ping(ctx); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"status":    "error",
				"component": "postgres",
			})
		}
		if err := deps.Redis.Ping(ctx); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"status":    "error",
				"component": "redis",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Prometheus metrics
	e.Use(echoprometheus.NewMiddleware("retrocast"))
	e.GET("/metrics", echoprometheus.NewHandler())

	// Swagger UI — serves the embedded OpenAPI spec
	e.GET("/docs", func(c echo.Context) error {
		return c.HTML(http.StatusOK, swaggerUIHTML)
	})
	e.GET("/docs/openapi.yaml", func(c echo.Context) error {
		return c.Blob(http.StatusOK, "application/yaml", docs.OpenAPISpec)
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

	// DM channels
	protected.POST("/users/@me/channels", deps.DMs.CreateDM)
	protected.GET("/users/@me/channels", deps.DMs.ListDMs)

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

	// Attachments
	protected.POST("/channels/:id/attachments", deps.Uploads.Upload)

	// Typing
	protected.POST("/channels/:id/typing", deps.Typing.Handle)

	// Bans
	protected.PUT("/guilds/:id/bans/:user_id", deps.Bans.BanMember)
	protected.DELETE("/guilds/:id/bans/:user_id", deps.Bans.UnbanMember)
	protected.GET("/guilds/:id/bans", deps.Bans.ListBans)

	// Invites (protected)
	protected.POST("/guilds/:id/invites", deps.Invites.CreateInvite)
	protected.GET("/guilds/:id/invites", deps.Invites.ListInvites)
	protected.POST("/invites/:code", deps.Invites.AcceptInvite)
	protected.DELETE("/invites/:code", deps.Invites.RevokeInvite)
}

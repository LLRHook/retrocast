package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/redis"
)

// RateLimitMiddleware creates per-IP (unauthenticated) or per-user (authenticated)
// rate limiting using Redis.
func RateLimitMiddleware(redisClient *redis.Client, limit int, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var key string
			if uid, ok := c.Get("user_id").(int64); ok {
				key = fmt.Sprintf("rl:user:%d:%s", uid, c.Path())
			} else {
				key = fmt.Sprintf("rl:ip:%s:%s", c.RealIP(), c.Path())
			}

			allowed, err := redisClient.CheckRateLimit(c.Request().Context(), key, limit, window)
			if err != nil {
				// On Redis failure, allow the request through rather than blocking users.
				return next(c)
			}
			if !allowed {
				return errorJSON(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests, please try again later")
			}

			return next(c)
		}
	}
}

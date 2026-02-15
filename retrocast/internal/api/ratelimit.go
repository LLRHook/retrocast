package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/redis"
)

// RateLimitMiddleware creates per-IP (unauthenticated) or per-user (authenticated)
// rate limiting using Redis. Sets standard rate limit response headers.
func RateLimitMiddleware(redisClient *redis.Client, limit int, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var key string
			if uid, ok := c.Get("user_id").(int64); ok {
				key = fmt.Sprintf("rl:user:%d:%s", uid, c.Path())
			} else {
				key = fmt.Sprintf("rl:ip:%s:%s", c.RealIP(), c.Path())
			}

			allowed, count, ttlMs, err := redisClient.CheckRateLimit(c.Request().Context(), key, limit, window)
			if err != nil {
				// On Redis failure, allow the request through rather than blocking users.
				return next(c)
			}

			remaining := int64(limit) - count
			if remaining < 0 {
				remaining = 0
			}
			resetAt := time.Now().Add(time.Duration(ttlMs) * time.Millisecond).Unix()

			c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			c.Response().Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
			c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))

			if !allowed {
				retryAfterSec := (ttlMs + 999) / 1000 // round up to next second
				c.Response().Header().Set("Retry-After", strconv.FormatInt(retryAfterSec, 10))
				return errorJSON(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests, please try again later")
			}

			return next(c)
		}
	}
}

package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/victorivanov/retrocast/internal/redis"
)

// ClientIPExtractor returns the extractor Echo uses for Context.RealIP.
//
// Since Echo v4.16.3, RealIP returns the socket peer when no extractor is set, so
// every client behind the bundled Caddy reverse proxy would share one rate-limit
// key. This extractor reads X-Forwarded-For but trusts a hop only when it is a
// loopback, link-local or private address (Echo's default trust set, which covers
// the Docker network Caddy runs on). A client connecting from a public address
// cannot choose its own rate-limit key by sending forwarding headers.
func ClientIPExtractor() echo.IPExtractor {
	return echo.ExtractIPFromXFFHeader()
}

// RateLimitMiddleware creates per-IP (unauthenticated) or per-user (authenticated)
// rate limiting using Redis. Sets standard rate limit response headers.
func RateLimitMiddleware(redisClient *redis.Client, limit int, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			_, authenticated := c.Get("user_id").(int64)
			var key string
			if uid, ok := c.Get("user_id").(int64); ok {
				key = fmt.Sprintf("rl:user:%d:%s", uid, c.Path())
			} else {
				key = fmt.Sprintf("rl:ip:%s:%s", c.RealIP(), c.Path())
			}

			allowed, count, ttlMs, err := redisClient.CheckRateLimit(c.Request().Context(), key, limit, window)
			if err != nil {
				// Keep authenticated members chatting; prevent unthrottled anonymous abuse.
				if !authenticated {
					return errorJSON(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "rate limiting unavailable")
				}
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

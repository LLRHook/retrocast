package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Client wraps a Redis connection for session and rate-limiting operations.
type Client struct {
	rdb *goredis.Client
}

// NewClient creates a Redis client from a URL and verifies the connection.
func NewClient(redisURL string) (*Client, error) {
	opts, err := goredis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parsing redis URL: %w", err)
	}
	rdb := goredis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connecting to redis: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

// Ping checks the Redis connection.
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Close closes the Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}

const (
	refreshTokenPrefix = "refresh:"
	presencePrefix     = "presence:"
	typingPrefix       = "typing:"
	presenceTTL        = 5 * time.Minute
	typingTTL          = 10 * time.Second
)

// StoreRefreshToken stores a refresh token mapped to a user ID with an expiry.
func (c *Client) StoreRefreshToken(ctx context.Context, token string, userID int64, expiry time.Duration) error {
	return c.rdb.Set(ctx, refreshTokenPrefix+token, userID, expiry).Err()
}

// GetRefreshTokenUserID returns the user ID associated with a refresh token.
func (c *Client) GetRefreshTokenUserID(ctx context.Context, token string) (int64, error) {
	val, err := c.rdb.Get(ctx, refreshTokenPrefix+token).Result()
	if err == goredis.Nil {
		return 0, fmt.Errorf("refresh token not found")
	}
	if err != nil {
		return 0, fmt.Errorf("getting refresh token: %w", err)
	}

	userID, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing user ID: %w", err)
	}
	return userID, nil
}

// DeleteRefreshToken removes a refresh token from Redis.
func (c *Client) DeleteRefreshToken(ctx context.Context, token string) error {
	return c.rdb.Del(ctx, refreshTokenPrefix+token).Err()
}

// rateLimitScript atomically increments a counter and sets its TTL on first use.
var rateLimitScript = goredis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
    redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return count
`)

// CheckRateLimit returns true if the request is allowed, false if rate limited.
// Uses an atomic INCR + PEXPIRE Lua script for a fixed-window counter.
func (c *Client) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	count, err := rateLimitScript.Run(ctx, c.rdb, []string{key}, window.Milliseconds()).Int64()
	if err != nil {
		return false, fmt.Errorf("checking rate limit: %w", err)
	}
	return count <= int64(limit), nil
}

// SetPresence sets a user's presence status with a TTL.
func (c *Client) SetPresence(ctx context.Context, userID int64, status string) error {
	key := presencePrefix + strconv.FormatInt(userID, 10)
	return c.rdb.Set(ctx, key, status, presenceTTL).Err()
}

// GetPresence returns a user's presence status, or empty string if not set.
func (c *Client) GetPresence(ctx context.Context, userID int64) (string, error) {
	key := presencePrefix + strconv.FormatInt(userID, 10)
	val, err := c.rdb.Get(ctx, key).Result()
	if err == goredis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("getting presence: %w", err)
	}
	return val, nil
}

// DeletePresence removes a user's presence status.
func (c *Client) DeletePresence(ctx context.Context, userID int64) error {
	key := presencePrefix + strconv.FormatInt(userID, 10)
	return c.rdb.Del(ctx, key).Err()
}

// SetTyping marks a user as typing in a channel with a short TTL.
func (c *Client) SetTyping(ctx context.Context, channelID, userID int64) error {
	key := typingPrefix + strconv.FormatInt(channelID, 10) + ":" + strconv.FormatInt(userID, 10)
	return c.rdb.Set(ctx, key, 1, typingTTL).Err()
}

// GetTyping returns the user IDs currently typing in a channel.
func (c *Client) GetTyping(ctx context.Context, channelID int64) ([]int64, error) {
	pattern := typingPrefix + strconv.FormatInt(channelID, 10) + ":*"
	prefix := typingPrefix + strconv.FormatInt(channelID, 10) + ":"

	var userIDs []int64
	var cursor uint64
	for {
		keys, next, err := c.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scanning typing keys: %w", err)
		}
		for _, key := range keys {
			uidStr := key[len(prefix):]
			uid, err := strconv.ParseInt(uidStr, 10, 64)
			if err != nil {
				continue
			}
			userIDs = append(userIDs, uid)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return userIDs, nil
}

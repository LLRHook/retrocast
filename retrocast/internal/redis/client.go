package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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
	opts.ContextTimeoutEnabled = true
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

// RefreshGeneration returns the revocation epoch (legacy accounts start at zero).
func (c *Client) RefreshGeneration(ctx context.Context, userID int64) (int64, error) {
	val, err := c.rdb.Get(ctx, fmt.Sprintf("refresh_gen:%d", userID)).Int64()
	if err == goredis.Nil {
		return 0, nil
	}
	return val, err
}

// AdvanceRefreshGeneration fails closed before any password update.
func (c *Client) AdvanceRefreshGeneration(ctx context.Context, userID int64) error {
	return c.rdb.Incr(ctx, fmt.Sprintf("refresh_gen:%d", userID)).Err()
}

var storeRefreshScript = goredis.NewScript(`
local gen = redis.call("GET", KEYS[2]) or "0"
if ARGV[3] ~= "" and gen ~= ARGV[3] then return 0 end
redis.call("SET", KEYS[1], ARGV[1] .. ":" .. gen, "PX", ARGV[2])
return 1
`)

func (c *Client) StoreRefreshToken(ctx context.Context, token string, userID int64, expiry time.Duration) error {
	return c.storeRefresh(ctx, token, userID, expiry, "")
}
func (c *Client) StoreRefreshTokenAtGeneration(ctx context.Context, token string, userID int64, expiry time.Duration, gen int64) error {
	return c.storeRefresh(ctx, token, userID, expiry, strconv.FormatInt(gen, 10))
}
func (c *Client) storeRefresh(ctx context.Context, token string, userID int64, expiry time.Duration, gen string) error {
	n, err := storeRefreshScript.Run(ctx, c.rdb, []string{refreshTokenPrefix + token, fmt.Sprintf("refresh_gen:%d", userID)}, userID, expiry.Milliseconds(), gen).Int()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("session generation changed")
	}
	return nil
}

// consumeRefreshScript is shared by rotation, consume and logout. Redis executes
// the generation check, deletion and replacement without an invalidation gap.
var consumeRefreshScript = goredis.NewScript(`
local value = redis.call("GET", KEYS[1])
if not value then return redis.error_reply("invalid token") end
local uid, gen = string.match(value, "^(%d+):(%d+)$")
if not uid then uid = value; gen = "0" end
local current = redis.call("GET", "refresh_gen:" .. uid) or "0"
redis.call("DEL", KEYS[1])
if gen ~= current then return redis.error_reply("invalid token") end
if KEYS[2] ~= KEYS[1] then redis.call("SET", KEYS[2], uid .. ":" .. gen, "PX", ARGV[1]) end
return uid
`)

func (c *Client) RotateRefreshToken(ctx context.Context, old, next string, expiry time.Duration) (int64, error) {
	val, err := consumeRefreshScript.Run(ctx, c.rdb, []string{refreshTokenPrefix + old, refreshTokenPrefix + next}, expiry.Milliseconds()).Text()
	if err != nil {
		return 0, fmt.Errorf("consuming refresh token: %w", err)
	}
	return strconv.ParseInt(val, 10, 64)
}
func (c *Client) ConsumeRefreshTokenUserID(ctx context.Context, token string) (int64, error) {
	return c.RotateRefreshToken(ctx, token, token, time.Second)
}
func (c *Client) DeleteRefreshToken(ctx context.Context, token string) error {
	_, err := c.ConsumeRefreshTokenUserID(ctx, token)
	return err
}
func (c *Client) GetRefreshTokenUserID(ctx context.Context, token string) (int64, error) {
	val, err := c.rdb.Get(ctx, refreshTokenPrefix+token).Result()
	if err != nil {
		return 0, err
	}
	parts := strings.Split(val, ":")
	uid, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	gen := int64(0)
	if len(parts) == 2 {
		gen, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, err
		}
	}
	current, err := c.RefreshGeneration(ctx, uid)
	if err != nil {
		return 0, err
	}
	if gen != current {
		return 0, fmt.Errorf("invalid token")
	}
	return uid, nil
}

// rateLimitScript atomically increments a counter, sets its TTL on first use,
// and returns {count, pttl_ms} for rate limit headers.
var rateLimitScript = goredis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
    redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
local ttl = redis.call("PTTL", KEYS[1])
return {count, ttl}
`)

// CheckRateLimit returns whether the request is allowed, the current count,
// and the remaining TTL in milliseconds for the rate limit window.
func (c *Client) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, count int64, ttlMs int64, err error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	result, err := rateLimitScript.Run(ctx, c.rdb, []string{key}, window.Milliseconds()).Result()
	if err != nil {
		return false, 0, 0, fmt.Errorf("checking rate limit: %w", err)
	}
	vals, ok := result.([]interface{})
	if !ok || len(vals) < 2 {
		return false, 0, 0, fmt.Errorf("unexpected rate limit result")
	}
	count, _ = vals[0].(int64)
	ttlMs, _ = vals[1].(int64)
	return count <= int64(limit), count, ttlMs, nil
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

// RefreshPresence extends the existing status without changing idle/dnd/invisible.
func (c *Client) RefreshPresence(ctx context.Context, userID int64) error {
	return c.rdb.Expire(ctx, presencePrefix+strconv.FormatInt(userID, 10), presenceTTL).Err()
}

package redis

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestLegacyRefreshGenerationLive(t *testing.T) {
	if os.Getenv("REDIS_URL") == "" {
		t.Skip("REDIS_URL required")
	}
	c, err := NewClient(os.Getenv("REDIS_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	ctx := context.Background()
	uid := time.Now().UnixNano()
	genkey := fmt.Sprintf("refresh_gen:%d", uid)
	defer c.rdb.Del(ctx, genkey)
	token := fmt.Sprintf("l25-legacy-%d", uid)
	next := token + "next"
	defer c.rdb.Del(ctx, refreshTokenPrefix+token, refreshTokenPrefix+next)
	if err := c.rdb.Set(ctx, refreshTokenPrefix+token, uid, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if got, err := c.RotateRefreshToken(ctx, token, next, time.Minute); err != nil || got != uid {
		t.Fatal("legacy token refused before revocation")
	}
	if err := c.rdb.Set(ctx, refreshTokenPrefix+token, uid, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if err := c.AdvanceRefreshGeneration(ctx, uid); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{token, next} {
		if _, err := c.RotateRefreshToken(ctx, v, v+"invalid", time.Minute); err == nil {
			t.Fatal("old generation survived")
		}
	}
}

func TestMalformedRefreshValueLive(t *testing.T) {
	if os.Getenv("REDIS_URL") == "" {
		t.Skip("REDIS_URL required")
	}
	c, err := NewClient(os.Getenv("REDIS_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	ctx := context.Background()
	for i, value := range []string{"abc", "abc:0", "+123", "123:invalid", "12:3:4", "", "00123", "9223372036854775808", "10000000000000000000"} {
		token := fmt.Sprintf("l25-malformed-%d-%d", time.Now().UnixNano(), i)
		next := token + "next"
		if err := c.rdb.Set(ctx, refreshTokenPrefix+token, value, time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
		_, err := c.RotateRefreshToken(ctx, token, next, time.Minute)
		count, e := c.rdb.Exists(ctx, refreshTokenPrefix+token, refreshTokenPrefix+next).Result()
		_ = c.rdb.Del(ctx, refreshTokenPrefix+token, refreshTokenPrefix+next).Err()
		if err == nil || e != nil || count != 0 {
			t.Fatalf("malformed case %d wrote or retained a token", i)
		}
	}
}

func TestMaxInt64RefreshLive(t *testing.T) {
	if os.Getenv("REDIS_URL") == "" {
		t.Skip("REDIS_URL required")
	}
	c, err := NewClient(os.Getenv("REDIS_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	ctx := context.Background()
	token := fmt.Sprintf("l28-max-%d", time.Now().UnixNano())
	defer c.rdb.Del(ctx, refreshTokenPrefix+token, refreshTokenPrefix+token+"next")
	if err := c.rdb.Set(ctx, refreshTokenPrefix+token, "9223372036854775807:0", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if id, err := c.RotateRefreshToken(ctx, token, token+"next", time.Minute); err != nil || id != 9223372036854775807 {
		t.Fatal("max int64 must be accepted")
	}
}

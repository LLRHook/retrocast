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

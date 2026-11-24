package database

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/models"
	"os"
	"strings"
	"testing"
)

type memberIdentityTracer struct{ queries int }

func (m *memberIdentityTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if strings.Contains(data.SQL, "FROM users WHERE id = ANY") {
		m.queries++
	}
	return ctx
}
func (*memberIdentityTracer) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}
func TestMemberPublicIdentityBatch(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL required")
	}
	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	tracer := &memberIdentityTracer{}
	cfg.ConnConfig.Tracer = tracer
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	users := NewUserRepository(pool)
	u := createTestUserSimple(t, users)
	u2 := createTestUserSimple(t, users)
	members := []*models.Member{{UserID: u.ID}, {UserID: u2.ID}, {UserID: -1}}
	if err := (&memberRepo{pool: pool}).HydrateUsers(context.Background(), members); err != nil {
		t.Fatal(err)
	}
	if tracer.queries != 1 {
		t.Fatalf("identity queries = %d", tracer.queries)
	}
	for i, m := range members {
		if i == 2 {
			if m.User != nil {
				t.Fatal("missing user must be absent")
			}
			continue
		}
		if m.User == nil || m.User.ID != m.UserID || m.User.Username == "" {
			t.Fatalf("identity absent: %+v", m)
		}
	}
	b, err := json.Marshal(members[0])
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err = json.Unmarshal(b, &body); err != nil {
		t.Fatal(err)
	}
	user := body["user"].(map[string]any)
	for key := range user {
		if key != "id" && key != "username" && key != "display_name" && key != "avatar_hash" {
			t.Fatalf("nonpublic field %s", key)
		}
	}
	if user["id"] == nil || user["username"] == nil {
		t.Fatal("missing public identity")
	}
}

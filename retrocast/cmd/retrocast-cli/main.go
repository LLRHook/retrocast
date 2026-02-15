package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/victorivanov/retrocast/internal/auth"
	"github.com/victorivanov/retrocast/internal/snowflake"
)

// Set via -ldflags at build time.
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "migrate":
		if hasFlag("--help", os.Args[2:]) {
			fmt.Println("Usage: retrocast-cli migrate")
			fmt.Println()
			fmt.Println("Run database migrations from the migrations/ directory.")
			fmt.Println()
			fmt.Println("Environment:")
			fmt.Println("  DATABASE_URL  PostgreSQL connection string (required)")
			return
		}
		os.Exit(runMigrate())
	case "seed":
		if hasFlag("--help", os.Args[2:]) {
			fmt.Println("Usage: retrocast-cli seed")
			fmt.Println()
			fmt.Println("Seed the database with demo data: 2 users, a guild, channels, and messages.")
			fmt.Println()
			fmt.Println("Environment:")
			fmt.Println("  DATABASE_URL  PostgreSQL connection string (required)")
			return
		}
		os.Exit(runSeed())
	case "health":
		if hasFlag("--help", os.Args[2:]) {
			fmt.Println("Usage: retrocast-cli health")
			fmt.Println()
			fmt.Println("Check if the Retrocast server is running.")
			fmt.Println()
			fmt.Println("Environment:")
			fmt.Println("  SERVER_URL  Server base URL (default: http://localhost:8080)")
			return
		}
		os.Exit(runHealth())
	case "version":
		fmt.Printf("retrocast-cli %s\n", version)
	case "--help", "-h", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: retrocast-cli <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  migrate  Run database migrations")
	fmt.Println("  seed     Seed demo data (users, guild, channels, messages)")
	fmt.Println("  health   Check if the server is running")
	fmt.Println("  version  Print version info")
	fmt.Println()
	fmt.Println("Run 'retrocast-cli <command> --help' for details on a command.")
}

func hasFlag(flag string, args []string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintf(os.Stderr, "error: %s environment variable is required\n", key)
		os.Exit(1)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// --- migrate ---

func runMigrate() int {
	dbURL := requireEnv("DATABASE_URL")

	fmt.Println("connecting to database...")
	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: migration init failed: %v\n", err)
		return 1
	}
	defer m.Close()

	fmt.Println("running migrations...")
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		fmt.Fprintf(os.Stderr, "error: migration failed: %v\n", err)
		return 1
	}

	v, dirty, _ := m.Version()
	if err == migrate.ErrNoChange {
		fmt.Printf("no new migrations (current version: %d)\n", v)
	} else {
		fmt.Printf("migrations applied (version: %d, dirty: %v)\n", v, dirty)
	}
	return 0
}

// --- seed ---

func runSeed() int {
	dbURL := requireEnv("DATABASE_URL")
	ctx := context.Background()

	fmt.Println("connecting to database...")
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: database connection failed: %v\n", err)
		return 1
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: database ping failed: %v\n", err)
		return 1
	}

	sf, err := snowflake.NewGenerator(0, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: snowflake init failed: %v\n", err)
		return 1
	}

	// Hash passwords for demo users.
	fmt.Println("hashing passwords...")
	aliceHash, err := auth.HashPassword("password123")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: hashing password: %v\n", err)
		return 1
	}
	bobHash, err := auth.HashPassword("password456")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: hashing password: %v\n", err)
		return 1
	}

	// Generate IDs.
	aliceID := sf.Generate()
	bobID := sf.Generate()
	guildID := sf.Generate()
	generalChanID := sf.Generate()
	randomChanID := sf.Generate()
	everyoneRoleID := sf.Generate()
	msg1ID := sf.Generate()
	msg2ID := sf.Generate()
	msg3ID := sf.Generate()

	now := time.Now()

	tx, err := pool.Begin(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: starting transaction: %v\n", err)
		return 1
	}
	defer tx.Rollback(ctx)

	// Users.
	fmt.Println("creating users...")
	_, err = tx.Exec(ctx,
		`INSERT INTO users (id, username, display_name, password_hash, created_at) VALUES ($1,$2,$3,$4,$5), ($6,$7,$8,$9,$10)
		 ON CONFLICT (id) DO NOTHING`,
		aliceID.Int64(), "alice", "Alice", aliceHash, now,
		bobID.Int64(), "bob", "Bob", bobHash, now,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: creating users: %v\n", err)
		return 1
	}

	// Guild.
	fmt.Println("creating guild...")
	_, err = tx.Exec(ctx,
		`INSERT INTO guilds (id, name, owner_id, created_at) VALUES ($1,$2,$3,$4)
		 ON CONFLICT (id) DO NOTHING`,
		guildID.Int64(), "Demo Server", aliceID.Int64(), now,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: creating guild: %v\n", err)
		return 1
	}

	// Channels.
	fmt.Println("creating channels...")
	_, err = tx.Exec(ctx,
		`INSERT INTO channels (id, guild_id, name, type, position) VALUES ($1,$2,$3,0,0), ($4,$5,$6,0,1)
		 ON CONFLICT (id) DO NOTHING`,
		generalChanID.Int64(), guildID.Int64(), "general",
		randomChanID.Int64(), guildID.Int64(), "random",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: creating channels: %v\n", err)
		return 1
	}

	// Default @everyone role.
	fmt.Println("creating default role...")
	_, err = tx.Exec(ctx,
		`INSERT INTO roles (id, guild_id, name, permissions, position, is_default) VALUES ($1,$2,$3,$4,0,true)
		 ON CONFLICT (id) DO NOTHING`,
		everyoneRoleID.Int64(), guildID.Int64(), "@everyone", int64(0x00000437),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: creating role: %v\n", err)
		return 1
	}

	// Members.
	fmt.Println("creating members...")
	_, err = tx.Exec(ctx,
		`INSERT INTO members (guild_id, user_id, joined_at) VALUES ($1,$2,$3), ($4,$5,$6)
		 ON CONFLICT (guild_id, user_id) DO NOTHING`,
		guildID.Int64(), aliceID.Int64(), now,
		guildID.Int64(), bobID.Int64(), now,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: creating members: %v\n", err)
		return 1
	}

	// Member roles.
	_, err = tx.Exec(ctx,
		`INSERT INTO member_roles (guild_id, user_id, role_id) VALUES ($1,$2,$3), ($4,$5,$6)
		 ON CONFLICT (guild_id, user_id, role_id) DO NOTHING`,
		guildID.Int64(), aliceID.Int64(), everyoneRoleID.Int64(),
		guildID.Int64(), bobID.Int64(), everyoneRoleID.Int64(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: creating member roles: %v\n", err)
		return 1
	}

	// Messages.
	fmt.Println("creating messages...")
	_, err = tx.Exec(ctx,
		`INSERT INTO messages (id, channel_id, author_id, content, created_at) VALUES ($1,$2,$3,$4,$5), ($6,$7,$8,$9,$10), ($11,$12,$13,$14,$15)
		 ON CONFLICT (id) DO NOTHING`,
		msg1ID.Int64(), generalChanID.Int64(), aliceID.Int64(), "Welcome to the Demo Server!", now,
		msg2ID.Int64(), generalChanID.Int64(), bobID.Int64(), "Hey Alice, glad to be here!", now,
		msg3ID.Int64(), randomChanID.Int64(), aliceID.Int64(), "This is the random channel.", now,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: creating messages: %v\n", err)
		return 1
	}

	if err := tx.Commit(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: committing transaction: %v\n", err)
		return 1
	}

	fmt.Println()
	fmt.Println("seed complete:")
	fmt.Printf("  users:    alice (password: password123), bob (password: password456)\n")
	fmt.Printf("  guild:    Demo Server (owner: alice)\n")
	fmt.Printf("  channels: #general, #random\n")
	fmt.Printf("  messages: 3 messages in #general and #random\n")
	return 0
}

// --- health ---

func runHealth() int {
	serverURL := envOr("SERVER_URL", "http://localhost:8080")
	url := serverURL + "/health"

	fmt.Printf("checking %s ...\n", url)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("status: %d\n", resp.StatusCode)
	if len(body) > 0 {
		fmt.Printf("body:   %s\n", string(body))
	}

	if resp.StatusCode == http.StatusOK {
		fmt.Println("server is healthy")
		return 0
	}
	fmt.Fprintln(os.Stderr, "server returned non-200 status")
	return 1
}

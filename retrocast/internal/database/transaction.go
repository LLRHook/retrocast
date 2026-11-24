package database

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type transactionKey struct{}
type querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func queryDB(ctx context.Context, pool *pgxpool.Pool) querier {
	if tx, ok := ctx.Value(transactionKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

// InTransaction joins an existing transaction, or starts one on the repository's
// pool. Repository doubles without a pool execute the callback directly.
func InTransaction(ctx context.Context, repository any, fn func(context.Context) error) error {
	if _, ok := ctx.Value(transactionKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	source, ok := repository.(interface{ transactionPool() *pgxpool.Pool })
	if !ok {
		return fn(ctx)
	}
	tx, err := source.transactionPool().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err := fn(context.WithValue(ctx, transactionKey{}, tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// LockRegistration serializes the bootstrap check with every account insertion.
func LockRegistration(ctx context.Context, users UserRepository) (bool, error) {
	source, ok := users.(interface{ transactionPool() *pgxpool.Pool })
	if !ok {
		return false, nil
	}
	db := queryDB(ctx, source.transactionPool())
	if _, err := db.Exec(ctx, "SELECT pg_advisory_xact_lock(72463001)"); err != nil {
		return false, err
	}
	var exists bool
	err := db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users)").Scan(&exists)
	return !exists, err
}

// LockGuildForModeration takes the parent before member or ban rows, matching
// guild deletion and invite acceptance. The transaction serializes those writes.
func LockGuildForModeration(ctx context.Context, repository any, guildID int64) error {
	source, ok := repository.(interface{ transactionPool() *pgxpool.Pool })
	if !ok {
		return nil
	}
	var id int64
	return queryDB(ctx, source.transactionPool()).QueryRow(ctx, "SELECT id FROM guilds WHERE id=$1 FOR UPDATE", guildID).Scan(&id)
}

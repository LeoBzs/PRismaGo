// Package postgres owns the shared database plumbing: opening the connection
// pool and applying schema migrations. Feature packages own their own queries.
package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// Migrations are compiled into the binary, so deploys need no extra files.
//
//go:embed migrations/*.sql
var embedded embed.FS

// Open creates a connection pool and verifies the database is reachable.
// The pool is safe for concurrent use by all request goroutines.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

// Migrate applies any pending migrations. Goose records applied versions in
// the goose_db_version table, so running it on every startup is a no-op once
// the schema is current.
func Migrate(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	migrations, err := fs.Sub(embedded, "migrations")
	if err != nil {
		return err
	}

	// goose speaks database/sql; wrap the pgx pool rather than opening a second one.
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	// A Postgres advisory lock makes concurrent callers (several API instances
	// starting at once, parallel test packages) take turns: the first applies
	// pending migrations, the rest wait and then find nothing to do.
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("create migration lock: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations, goose.WithSessionLocker(locker))
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	for _, r := range results {
		logger.Info("migration applied", "version", r.Source.Version, "file", r.Source.Path, "duration", r.Duration)
	}
	return nil
}

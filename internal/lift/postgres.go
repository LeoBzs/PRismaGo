package lift

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// foreignKeyViolation is the Postgres SQLSTATE for a REFERENCES failure.
const foreignKeyViolation = "23503"

// PostgresRepository implements Repository. Concurrency is handled by the
// pool and the database, so unlike MemoryRepository it needs no mutex.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const selectLifts = `SELECT id, user_id, exercise, weight_kg, reps, performed_at, created_at FROM lifts`

func (r *PostgresRepository) Create(ctx context.Context, l Lift) error {
	if !validUUID(l.UserID) {
		return ErrUserNotFound
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO lifts (id, user_id, exercise, weight_kg, reps, performed_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		l.ID, l.UserID, l.Exercise, l.WeightKg, l.Reps, l.PerformedAt, l.CreatedAt,
	)
	// The users table is the source of truth for "does this user exist";
	// the foreign key rejects the insert if not.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
		return ErrUserNotFound
	}
	return err
}

func (r *PostgresRepository) Get(ctx context.Context, userID, id string) (Lift, error) {
	// A malformed id can't exist; without this Postgres rejects the cast and
	// the client would get a 500 instead of a 404.
	if !validUUID(userID) || !validUUID(id) {
		return Lift{}, ErrNotFound
	}
	l, err := scanLift(r.pool.QueryRow(ctx, selectLifts+` WHERE user_id = $1 AND id = $2`, userID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Lift{}, ErrNotFound
	}
	return l, err
}

func (r *PostgresRepository) List(ctx context.Context, f ListFilter) ([]Lift, error) {
	if !validUUID(f.UserID) {
		return nil, ErrUserNotFound
	}
	query, args := selectLifts+` WHERE user_id = $1`, []any{f.UserID}
	if f.Exercise != "" {
		query += ` AND exercise = $2`
		args = append(args, f.Exercise)
	}
	query += ` ORDER BY performed_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	lifts, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Lift, error) {
		return scanLift(row)
	})
	if err != nil || len(lifts) > 0 {
		return lifts, err
	}

	// Empty result: a real user with no lifts gets [], an unknown user gets
	// ErrUserNotFound. Only checked here so the common case stays one query.
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, f.UserID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}
	return lifts, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, userID, id string) error {
	if !validUUID(userID) || !validUUID(id) {
		return ErrNotFound
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM lifts WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func validUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// scanLift maps one row to a Lift. Column order must match selectLifts.
func scanLift(row pgx.Row) (Lift, error) {
	var l Lift
	err := row.Scan(&l.ID, &l.UserID, &l.Exercise, &l.WeightKg, &l.Reps, &l.PerformedAt, &l.CreatedAt)
	// pgx returns timestamptz in the Go process's local zone; keep the API in UTC.
	l.PerformedAt, l.CreatedAt = l.PerformedAt.UTC(), l.CreatedAt.UTC()
	return l, err
}

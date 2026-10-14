package session

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, tokenHash []byte, userID string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt,
	)
	return err
}

func (r *PostgresRepository) UserID(ctx context.Context, tokenHash []byte, now time.Time) (string, error) {
	var userID string
	err := r.pool.QueryRow(ctx,
		`SELECT user_id FROM sessions WHERE token_hash = $1 AND expires_at > $2`,
		tokenHash, now,
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUnauthenticated
	}
	return userID, err
}

func (r *PostgresRepository) Delete(ctx context.Context, tokenHash []byte) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

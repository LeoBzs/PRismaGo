package user

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// uniqueViolation is the Postgres SQLSTATE for a UNIQUE constraint failure.
const uniqueViolation = "23505"

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const selectUsers = `SELECT id, email, name, sex, units, height_cm, body_weight_kg, created_at FROM users`

func (r *PostgresRepository) Create(ctx context.Context, u User, passwordHash string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, name, sex, units, height_cm, body_weight_kg, created_at, password_hash)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		u.ID, u.Email, u.Name, u.Sex, u.Units, u.HeightCm, u.BodyWeightKg, u.CreatedAt, passwordHash,
	)
	return mapError(err)
}

// Credentials is the only query that reads password_hash; selectUsers never does.
func (r *PostgresRepository) Credentials(ctx context.Context, email string) (string, string, error) {
	var id, hash string
	err := r.pool.QueryRow(ctx, `SELECT id, password_hash FROM users WHERE email = $1`, email).Scan(&id, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return id, hash, err
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (User, error) {
	if _, err := uuid.Parse(id); err != nil {
		return User{}, ErrNotFound
	}
	u, err := scanUser(r.pool.QueryRow(ctx, selectUsers+` WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// Update replaces every editable column. created_at is never changed.
func (r *PostgresRepository) Update(ctx context.Context, u User) error {
	if _, err := uuid.Parse(u.ID); err != nil {
		return ErrNotFound
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE users
		 SET email = $2, name = $3, sex = $4, units = $5, height_cm = $6, body_weight_kg = $7
		 WHERE id = $1`,
		u.ID, u.Email, u.Name, u.Sex, u.Units, u.HeightCm, u.BodyWeightKg,
	)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrNotFound
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// mapError turns database errors the service cares about into domain errors.
func mapError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return ErrEmailTaken
	}
	return err
}

// scanUser maps one row to a User. Column order must match selectUsers.
func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Sex, &u.Units, &u.HeightCm, &u.BodyWeightKg, &u.CreatedAt)
	u.CreatedAt = u.CreatedAt.UTC()
	return u, err
}

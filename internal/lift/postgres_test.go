package lift

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/LeoBzs/prismago/internal/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Runs only when TEST_DATABASE_URL is set, e.g. against docker compose:
//
//	TEST_DATABASE_URL=postgres://prismago:prismago@localhost:5432/prismago?sslmode=disable go test ./...
func TestPostgresRepository(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()

	pool, err := postgres.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := postgres.Migrate(ctx, pool, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatal(err)
	}

	repo := NewPostgresRepository(pool)
	alice, bob := insertUser(t, pool), insertUser(t, pool)

	// Postgres stores microseconds; truncate so round-tripped values compare equal.
	now := time.Now().UTC().Truncate(time.Microsecond)
	older := Lift{ID: newID(), UserID: alice, Exercise: "squat", WeightKg: 142.5, Reps: 3, PerformedAt: now.Add(-time.Hour), CreatedAt: now}
	newer := Lift{ID: newID(), UserID: alice, Exercise: "squat", WeightKg: 145, Reps: 1, PerformedAt: now, CreatedAt: now}
	for _, l := range []Lift{older, newer} {
		if err := repo.Create(ctx, l); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	got, err := repo.Get(ctx, alice, older.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != older {
		t.Errorf("get = %+v, want %+v", got, older)
	}

	list, err := repo.List(ctx, ListFilter{UserID: alice, Exercise: "squat"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 || list[0].ID != newer.ID {
		t.Errorf("list = %+v, want 2 lifts newest first", list)
	}

	// Scoping: bob can't see or delete alice's lifts.
	if _, err := repo.Get(ctx, bob, older.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("other user's get: err = %v, want ErrNotFound", err)
	}
	if err := repo.Delete(ctx, bob, older.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("other user's delete: err = %v, want ErrNotFound", err)
	}
	if list, err := repo.List(ctx, ListFilter{UserID: bob}); err != nil || list == nil || len(list) != 0 {
		t.Errorf("user with no lifts: list = %#v, err = %v, want non-nil empty slice", list, err)
	}

	// Unknown user (well-formed or not) is a 404, not an empty list.
	for _, id := range []string{uuid.New().String(), "not-a-uuid"} {
		if _, err := repo.List(ctx, ListFilter{UserID: id}); !errors.Is(err, ErrUserNotFound) {
			t.Errorf("list for unknown user %q: err = %v, want ErrUserNotFound", id, err)
		}
	}

	// Foreign key: lifts can't be created for users that don't exist.
	orphan := older
	orphan.ID, orphan.UserID = newID(), uuid.New().String()
	if err := repo.Create(ctx, orphan); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("create for unknown user: err = %v, want ErrUserNotFound", err)
	}

	if _, err := repo.Get(ctx, alice, "not-a-uuid"); !errors.Is(err, ErrNotFound) {
		t.Errorf("get malformed id: err = %v, want ErrNotFound", err)
	}

	if err := repo.Delete(ctx, alice, older.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Get(ctx, alice, older.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("get deleted: err = %v, want ErrNotFound", err)
	}

	// ON DELETE CASCADE: removing the user removes their remaining lifts.
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, alice); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(ctx, alice, newer.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("lift after owner deleted: err = %v, want ErrNotFound", err)
	}
}

// insertUser creates a throwaway user row with plain SQL, so lift tests don't
// depend on the user package, and removes it (and, via cascade, its lifts) after.
func insertUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	id := uuid.NewV7().String()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, name, sex, units, height_cm, body_weight_kg, password_hash)
		 VALUES ($1, $2, 'Test', 'male', 'metric', 180, 80, 'x')`,
		id, "test-"+id+"@example.com",
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

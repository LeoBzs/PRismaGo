package user

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"
	"uuid"

	"github.com/LeoBzs/prismago/internal/postgres"
)

// Runs only when TEST_DATABASE_URL is set; see the README.
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

	svc := NewService(NewPostgresRepository(pool))
	in := validInput()
	in.Email = "test-" + uuid.New().String() + "@example.com" // isolates this run's row
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, in.Email)
	})

	created, err := svc.Create(ctx, in, "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != created {
		t.Errorf("get = %+v, want %+v", got, created)
	}

	if _, err := svc.Create(ctx, in, "hash"); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("duplicate email: err = %v, want ErrEmailTaken", err)
	}

	if id, hash, err := svc.Credentials(ctx, " "+strings.ToUpper(in.Email)); err != nil || id != created.ID || hash != "hash" {
		t.Errorf("credentials = %q, %q, %v; want %q, \"hash\"", id, hash, err, created.ID)
	}
	if _, _, err := svc.Credentials(ctx, "nobody@example.com"); !errors.Is(err, ErrNotFound) {
		t.Errorf("credentials unknown email: err = %v, want ErrNotFound", err)
	}

	in.Name = "Renamed"
	if _, err := svc.Update(ctx, created.ID, in); err != nil {
		t.Fatalf("update: %v", err)
	}
	if got, _ := svc.Get(ctx, created.ID); got.Name != "Renamed" || !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("after update = %+v", got)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Get(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("get deleted: err = %v, want ErrNotFound", err)
	}
	if _, err := svc.Get(ctx, "not-a-uuid"); !errors.Is(err, ErrNotFound) {
		t.Errorf("get malformed id: err = %v, want ErrNotFound", err)
	}
}

package user

import (
	"context"
	"errors"
	"testing"
)

func validInput() Input {
	return Input{
		Email:        "  Lifter@Example.com ",
		Name:         " Sam ",
		Sex:          Female,
		Measurements: Measurements{Units: Imperial, Height: 66, Weight: 150},
	}
}

func TestServiceCreate(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	ctx := context.Background()

	u, err := svc.Create(ctx, validInput(), "hash")
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "lifter@example.com" || u.Name != "Sam" {
		t.Errorf("not normalized: email=%q name=%q", u.Email, u.Name)
	}
	// 66 in = 167.64 cm; 150 lb = 68.0388555 kg, rounded to 68.04.
	if u.HeightCm != 167.64 || u.BodyWeightKg != 68.04 {
		t.Errorf("metric = %v cm / %v kg, want 167.64 / 68.04", u.HeightCm, u.BodyWeightKg)
	}

	if _, err := svc.Create(ctx, validInput(), "hash"); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("duplicate email: err = %v, want ErrEmailTaken", err)
	}

	bad := validInput()
	bad.Email = "other@example.com"
	bad.Height = 180 // cm sent as imperial
	if _, err := svc.Create(ctx, bad, "hash"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("bad height: err = %v, want ErrInvalidInput", err)
	}
}

func TestServiceUpdate(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	ctx := context.Background()

	created, err := svc.Create(ctx, validInput(), "hash")
	if err != nil {
		t.Fatal(err)
	}

	in := validInput()
	in.Name = "Samantha"
	in.Measurements = Measurements{Units: Metric, Height: 168, Weight: 70}
	updated, err := svc.Update(ctx, created.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Samantha" || updated.Units != Metric || updated.BodyWeightKg != 70 {
		t.Errorf("update not applied: %+v", updated)
	}
	if updated.ID != created.ID || !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("id/created_at changed: %+v", updated)
	}

	if _, err := svc.Update(ctx, "missing", in); !errors.Is(err, ErrNotFound) {
		t.Errorf("update missing: err = %v, want ErrNotFound", err)
	}
}

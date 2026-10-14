package lift

import (
	"context"
	"errors"
	"testing"
)

func TestServiceCreate(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	ctx := context.Background()

	tests := []struct {
		name    string
		in      CreateInput
		wantErr error
	}{
		{"valid", CreateInput{Exercise: " Squat ", WeightKg: 140, Reps: 3}, nil},
		{"missing exercise", CreateInput{WeightKg: 100, Reps: 1}, ErrInvalidInput},
		{"zero weight", CreateInput{Exercise: "bench", Reps: 1}, ErrInvalidInput},
		{"weight rounds to zero", CreateInput{Exercise: "bench", WeightKg: 0.004, Reps: 1}, ErrInvalidInput},
		{"weight over max", CreateInput{Exercise: "bench", WeightKg: 1000.01, Reps: 1}, ErrInvalidInput},
		{"zero reps", CreateInput{Exercise: "bench", WeightKg: 100}, ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.Create(ctx, "user-1", tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && (got.Exercise != "squat" || got.UserID != "user-1") {
				t.Errorf("got exercise %q user %q, want normalized %q for user-1", got.Exercise, got.UserID, "squat")
			}
		})
	}
}

func TestServiceScopesLiftsToUser(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	ctx := context.Background()

	mine, err := svc.Create(ctx, "alice", CreateInput{Exercise: "squat", WeightKg: 100, Reps: 5})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Get(ctx, "bob", mine.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("other user's get: err = %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, "bob", mine.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("other user's delete: err = %v, want ErrNotFound", err)
	}
	if list, _ := svc.List(ctx, ListFilter{UserID: "bob"}); len(list) != 0 {
		t.Errorf("other user's list = %+v, want empty", list)
	}
	if list, _ := svc.List(ctx, ListFilter{UserID: "alice"}); len(list) != 1 {
		t.Errorf("owner's list has %d lifts, want 1", len(list))
	}
}

func TestEstimated1RM(t *testing.T) {
	if got := (Lift{WeightKg: 100, Reps: 1}).Estimated1RM(); got != 100 {
		t.Errorf("single = %v, want 100", got)
	}
	if got := (Lift{WeightKg: 90, Reps: 30}).Estimated1RM(); got != 180 {
		t.Errorf("30 reps = %v, want 180", got)
	}
}

package lift

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
	"uuid"
)

const maxWeightKg = 1000

type Service struct {
	repo Repository
	now  func() time.Time // injectable for tests
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

type CreateInput struct {
	Exercise    string     `json:"exercise"`
	WeightKg    float64    `json:"weight_kg"`
	Reps        int        `json:"reps"`
	PerformedAt *time.Time `json:"performed_at"` // optional, defaults to now
}

// Create records a lift for userID. Who the user is comes from the caller
// (today the URL, later the auth token), never from the request body.
func (s *Service) Create(ctx context.Context, userID string, in CreateInput) (Lift, error) {
	exercise := strings.ToLower(strings.TrimSpace(in.Exercise))
	if exercise == "" {
		return Lift{}, fmt.Errorf("%w: exercise is required", ErrInvalidInput)
	}
	// Round to the precision the database stores (numeric(6,2), microsecond
	// timestamps) so the returned Lift is exactly what a later GET returns.
	weightKg := math.Round(in.WeightKg*100) / 100
	if weightKg <= 0 || weightKg > maxWeightKg {
		return Lift{}, fmt.Errorf("%w: weight_kg must be > 0 and <= %d", ErrInvalidInput, maxWeightKg)
	}
	if in.Reps <= 0 {
		return Lift{}, fmt.Errorf("%w: reps must be > 0", ErrInvalidInput)
	}

	now := s.now().UTC().Truncate(time.Microsecond)
	performedAt := now
	if in.PerformedAt != nil {
		performedAt = in.PerformedAt.UTC().Truncate(time.Microsecond)
	}

	l := Lift{
		ID:          newID(),
		UserID:      userID,
		Exercise:    exercise,
		WeightKg:    weightKg,
		Reps:        in.Reps,
		PerformedAt: performedAt,
		CreatedAt:   now,
	}
	if err := s.repo.Create(ctx, l); err != nil {
		return Lift{}, err
	}
	return l, nil
}

func (s *Service) Get(ctx context.Context, userID, id string) (Lift, error) {
	return s.repo.Get(ctx, userID, id)
}

func (s *Service) List(ctx context.Context, f ListFilter) ([]Lift, error) {
	f.Exercise = strings.ToLower(strings.TrimSpace(f.Exercise))
	return s.repo.List(ctx, f)
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	return s.repo.Delete(ctx, userID, id)
}

// newID returns a UUIDv7: time-ordered, so inserts append to the primary-key
// index instead of landing at random positions.
func newID() string {
	return uuid.NewV7().String()
}

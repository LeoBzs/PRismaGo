package user

import (
	"context"
	"math"
	"strings"
	"time"
	"uuid"
)

type Service struct {
	repo Repository
	now  func() time.Time // injectable for tests
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Input is the profile as the client sends it, with height and weight in
// the user's chosen units. Measurements is embedded, so its fields appear
// at the top level of the JSON: {"email", "name", "sex", "units", "height", "weight"}.
type Input struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Sex   Sex    `json:"sex"`
	Measurements
}

// Create registers a user. Password policy and hashing belong to the auth
// package; this one only stores the hash it is given.
func (s *Service) Create(ctx context.Context, in Input, passwordHash string) (User, error) {
	u, err := fromInput(in)
	if err != nil {
		return User{}, err
	}
	u.ID = uuid.NewV7().String()
	u.CreatedAt = s.now().UTC().Truncate(time.Microsecond)
	if err := s.repo.Create(ctx, u, passwordHash); err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Service) Get(ctx context.Context, id string) (User, error) {
	return s.repo.Get(ctx, id)
}

// Credentials returns the id and password hash for a login email.
func (s *Service) Credentials(ctx context.Context, email string) (userID, passwordHash string, err error) {
	return s.repo.Credentials(ctx, strings.ToLower(strings.TrimSpace(email)))
}

// Update replaces the whole profile (PUT semantics).
func (s *Service) Update(ctx context.Context, id string, in Input) (User, error) {
	existing, err := s.repo.Get(ctx, id)
	if err != nil {
		return User{}, err
	}
	u, err := fromInput(in)
	if err != nil {
		return User{}, err
	}
	u.ID, u.CreatedAt = existing.ID, existing.CreatedAt
	if err := s.repo.Update(ctx, u); err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// fromInput normalizes and validates client input into a User (no ID yet).
func fromInput(in Input) (User, error) {
	heightCm, weightKg, err := in.ToMetric()
	if err != nil {
		return User{}, err
	}
	u := User{
		Email: strings.ToLower(strings.TrimSpace(in.Email)),
		Name:  strings.TrimSpace(in.Name),
		Sex:   in.Sex,
		Units: in.Units,
		// Round to numeric(5,2) so the response matches what a later GET returns.
		HeightCm:     round2(heightCm),
		BodyWeightKg: round2(weightKg),
	}
	if err := u.Validate(); err != nil {
		return User{}, err
	}
	return u, nil
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

package session

import (
	"context"
	"crypto/rand"
	"time"
)

type Service struct {
	repo Repository
	now  func() time.Time // injectable for tests
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Start logs userID in and returns the token to hand to the client. The
// plaintext token is returned once and never stored.
func (s *Service) Start(ctx context.Context, userID string) (Token, error) {
	t := Token{
		Value:     rand.Text(), // 128+ bits from a CSPRNG
		ExpiresAt: s.now().UTC().Add(TTL).Truncate(time.Microsecond),
	}
	if err := s.repo.Create(ctx, hash(t.Value), userID, t.ExpiresAt); err != nil {
		return Token{}, err
	}
	return t, nil
}

// Authenticate returns the user id for a valid, unexpired token.
func (s *Service) Authenticate(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", ErrUnauthenticated
	}
	return s.repo.UserID(ctx, hash(token), s.now())
}

// End logs the token out. Unknown or empty tokens are a no-op.
func (s *Service) End(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.repo.Delete(ctx, hash(token))
}

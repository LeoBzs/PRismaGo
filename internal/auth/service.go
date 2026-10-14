package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/LeoBzs/prismago/internal/session"
	"github.com/LeoBzs/prismago/internal/user"
	"golang.org/x/crypto/bcrypt"
)

// Users is what auth needs from the user feature. *user.Service satisfies it.
type Users interface {
	Create(ctx context.Context, in user.Input, passwordHash string) (user.User, error)
	Get(ctx context.Context, id string) (user.User, error)
	Credentials(ctx context.Context, email string) (userID, passwordHash string, err error)
}

// Sessions is what auth needs from the session feature. *session.Service satisfies it.
type Sessions interface {
	Start(ctx context.Context, userID string) (session.Token, error)
	End(ctx context.Context, token string) error
}

type Service struct {
	users    Users
	sessions Sessions
	cost     int // bcrypt work factor; tests lower it for speed
}

func NewService(users Users, sessions Sessions) *Service {
	// Build the dummy hash now, so the first unknown-email login isn't slower
	// than every later one (that one slow response would itself be a signal).
	dummyHash()
	return &Service{users: users, sessions: sessions, cost: bcrypt.DefaultCost}
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (Result, error) {
	if n := len(in.Password); n < minPasswordBytes || n > maxPasswordBytes {
		return Result{}, fmt.Errorf("%w: password must be %d-%d bytes", ErrInvalidInput, minPasswordBytes, maxPasswordBytes)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), s.cost)
	if err != nil {
		return Result{}, err
	}
	u, err := s.users.Create(ctx, in.Input, string(hash))
	if err != nil {
		return Result{}, err
	}
	return s.startSession(ctx, u)
}

func (s *Service) Login(ctx context.Context, in LoginInput) (Result, error) {
	userID, hash, err := s.users.Credentials(ctx, in.Email)
	if errors.Is(err, user.ErrNotFound) {
		// Do the same bcrypt work as a real check, so the response time doesn't
		// reveal whether the email is registered.
		_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(in.Password))
		return Result{}, ErrInvalidCredentials
	}
	if err != nil {
		return Result{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		return Result{}, ErrInvalidCredentials
	}
	u, err := s.users.Get(ctx, userID)
	if err != nil {
		return Result{}, err
	}
	return s.startSession(ctx, u)
}

// Logout ends the session for token. Unknown tokens are not an error.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.sessions.End(ctx, token)
}

func (s *Service) startSession(ctx context.Context, u user.User) (Result, error) {
	tok, err := s.sessions.Start(ctx, u.ID)
	if err != nil {
		return Result{}, err
	}
	return Result{Token: tok, User: u}, nil
}

// dummyHash is computed once (NewService triggers it) and reused.
var dummyHash = sync.OnceValue(func() []byte {
	h, err := bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)
	if err != nil {
		panic(err) // only fails for invalid cost or >72-byte input; both are constants
	}
	return h
})

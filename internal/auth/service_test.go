package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/LeoBzs/prismago/internal/session"
	"github.com/LeoBzs/prismago/internal/user"
	"golang.org/x/crypto/bcrypt"
)

func newTestService() (*Service, *session.Service) {
	sessions := session.NewService(session.NewMemoryRepository())
	svc := NewService(user.NewService(user.NewMemoryRepository()), sessions)
	svc.cost = bcrypt.MinCost // real cost makes each hash ~100ms
	return svc, sessions
}

func registerInput(password string) RegisterInput {
	return RegisterInput{
		Input: user.Input{
			Email:        "sam@example.com",
			Name:         "Sam",
			Sex:          user.Male,
			Measurements: user.Measurements{Units: user.Metric, Height: 180, Weight: 80},
		},
		Password: password,
	}
}

func TestRegisterAndLogin(t *testing.T) {
	svc, sessions := newTestService()
	ctx := context.Background()

	reg, err := svc.Register(ctx, registerInput("correct horse"))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := sessions.Authenticate(ctx, reg.Token.Value); err != nil || got != reg.User.ID {
		t.Fatalf("register token authenticates as %q, %v; want %q", got, err, reg.User.ID)
	}

	login, err := svc.Login(ctx, LoginInput{Email: "SAM@example.com", Password: "correct horse"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if login.User.ID != reg.User.ID || login.Token.Value == reg.Token.Value {
		t.Errorf("login = %+v; want same user, new token", login)
	}

	// Wrong password and unknown email must be indistinguishable.
	for _, in := range []LoginInput{
		{Email: "sam@example.com", Password: "wrong password"},
		{Email: "nobody@example.com", Password: "correct horse"},
	} {
		if _, err := svc.Login(ctx, in); !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("login(%q): err = %v, want ErrInvalidCredentials", in.Email, err)
		}
	}

	if err := svc.Logout(ctx, login.Token.Value); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Authenticate(ctx, login.Token.Value); !errors.Is(err, session.ErrUnauthenticated) {
		t.Errorf("after logout: err = %v, want ErrUnauthenticated", err)
	}
	if _, err := sessions.Authenticate(ctx, reg.Token.Value); err != nil {
		t.Errorf("other session should survive logout: %v", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	for _, pw := range []string{"short", string(make([]byte, 73))} {
		if _, err := svc.Register(ctx, registerInput(pw)); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("password len %d: err = %v, want ErrInvalidInput", len(pw), err)
		}
	}

	if _, err := svc.Register(ctx, registerInput("long enough")); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(ctx, registerInput("long enough")); !errors.Is(err, user.ErrEmailTaken) {
		t.Errorf("duplicate email: err = %v, want user.ErrEmailTaken", err)
	}
}

// Package auth proves who a caller is: registration, login and logout.
// Passwords are hashed here (bcrypt) and nowhere else; the user package only
// stores the resulting hash, and the session package issues the token.
//
// auth has no table of its own, so there is no repository.go.
package auth

import (
	"errors"

	"github.com/LeoBzs/prismago/internal/session"
	"github.com/LeoBzs/prismago/internal/user"
)

var (
	// ErrInvalidCredentials is deliberately vague: it never says whether the
	// email or the password was wrong, so login can't be used to probe accounts.
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidInput       = errors.New("invalid input")
)

// bcrypt only hashes the first 72 bytes; reject longer instead of silently truncating.
const (
	minPasswordBytes = 8
	maxPasswordBytes = 72
)

// RegisterInput is a user profile plus a password. Both embeds flatten, so the
// JSON is {"email", "name", "sex", "units", "height", "weight", "password"}.
type RegisterInput struct {
	user.Input
	Password string `json:"password"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Result is returned by register and login:
// {"token", "expires_at", "user": {...}}.
type Result struct {
	session.Token
	User user.User `json:"user"`
}

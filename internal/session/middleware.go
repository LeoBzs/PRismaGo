package session

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/LeoBzs/prismago/internal/httpx"
)

// ctxKey is unexported so no other package can read or overwrite the value;
// UserID is the only way in.
type ctxKey struct{}

// RequireUser rejects requests without a valid bearer token (401) and passes
// the rest through with the user id in the context.
func (s *Service) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := s.Authenticate(r.Context(), BearerToken(r))
		if errors.Is(err, ErrUnauthenticated) {
			w.Header().Set("WWW-Authenticate", "Bearer") // RFC 6750 §3
			httpx.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserID returns the authenticated user's id. It is only set inside handlers
// wrapped by RequireUser; elsewhere it returns "".
func UserID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

// BearerToken extracts the token from "Authorization: Bearer <token>".
func BearerToken(r *http.Request) string {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

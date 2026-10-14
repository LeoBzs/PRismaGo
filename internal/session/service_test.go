package session

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionLifecycle(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	ctx := context.Background()

	tok, err := svc.Start(ctx, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := svc.Authenticate(ctx, tok.Value); err != nil || got != "user-1" {
		t.Fatalf("authenticate = %q, %v; want user-1", got, err)
	}
	for _, bad := range []string{"", "not-a-real-token"} {
		if _, err := svc.Authenticate(ctx, bad); !errors.Is(err, ErrUnauthenticated) {
			t.Errorf("authenticate(%q): err = %v, want ErrUnauthenticated", bad, err)
		}
	}

	// Expiry: move the clock past the TTL.
	realNow := svc.now
	svc.now = func() time.Time { return realNow().Add(TTL + time.Second) }
	if _, err := svc.Authenticate(ctx, tok.Value); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("expired: err = %v, want ErrUnauthenticated", err)
	}
	svc.now = realNow

	if err := svc.End(ctx, tok.Value); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(ctx, tok.Value); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("after logout: err = %v, want ErrUnauthenticated", err)
	}
}

func TestRequireUser(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	tok, err := svc.Start(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}

	var seen string
	h := svc.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = UserID(r.Context())
	}))

	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantUser   string
	}{
		{"no header", "", http.StatusUnauthorized, ""},
		{"wrong scheme", "Basic " + tok.Value, http.StatusUnauthorized, ""},
		{"unknown token", "Bearer nope", http.StatusUnauthorized, ""},
		{"valid", "Bearer " + tok.Value, http.StatusOK, "user-1"},
		{"scheme is case-insensitive", "bearer " + tok.Value, http.StatusOK, "user-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seen = ""
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus || seen != tt.wantUser {
				t.Errorf("status = %d, user = %q; want %d, %q", rec.Code, seen, tt.wantStatus, tt.wantUser)
			}
			if rec.Code == http.StatusUnauthorized && rec.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Errorf("401 without WWW-Authenticate: Bearer")
			}
		})
	}
}

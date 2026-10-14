package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	reached := false
	h := cors([]string{"http://localhost:4300"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
	}))

	tests := []struct {
		name        string
		method      string
		origin      string
		preflight   bool
		wantStatus  int
		wantAllow   string
		wantReached bool
	}{
		{"preflight from allowed origin", http.MethodOptions, "http://localhost:4300", true, http.StatusNoContent, "http://localhost:4300", false},
		{"real request from allowed origin", http.MethodPost, "http://localhost:4300", false, http.StatusOK, "http://localhost:4300", true},
		{"preflight from other origin", http.MethodOptions, "http://evil.example", true, http.StatusOK, "", true},
		{"no origin (curl, server-to-server)", http.MethodGet, "", false, http.StatusOK, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reached = false
			req := httptest.NewRequest(tt.method, "/api/v1/auth/register", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.preflight {
				req.Header.Set("Access-Control-Request-Method", http.MethodPost)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus || reached != tt.wantReached {
				t.Errorf("status = %d, reached = %v; want %d, %v", rec.Code, reached, tt.wantStatus, tt.wantReached)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.wantAllow {
				t.Errorf("Allow-Origin = %q, want %q", got, tt.wantAllow)
			}
		})
	}
}

package user

import (
	"errors"
	"net/http"

	"github.com/LeoBzs/prismago/internal/httpx"
	"github.com/LeoBzs/prismago/internal/session"
)

type Handler struct {
	svc         *Service
	requireUser func(http.Handler) http.Handler
}

// requireUser is the auth middleware (session.Service.RequireUser in main.go).
// Passing it in keeps this package testable without a real session store.
func NewHandler(svc *Service, requireUser func(http.Handler) http.Handler) *Handler {
	return &Handler{svc: svc, requireUser: requireUser}
}

// Every route acts on the logged-in user; there is no way to address another
// user by id. Registration lives in the auth package.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/me", h.requireUser(http.HandlerFunc(h.get)))
	mux.Handle("PUT /api/v1/me", h.requireUser(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/me", h.requireUser(http.HandlerFunc(h.delete)))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	u, err := h.svc.Get(r.Context(), session.UserID(r.Context()))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var in Input
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	u, err := h.svc.Update(r.Context(), session.UserID(r.Context()), in)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, u)
}

// delete removes the account; its lifts and sessions go with it (ON DELETE CASCADE).
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), session.UserID(r.Context())); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeError maps domain errors to HTTP status codes.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrEmailTaken):
		httpx.Error(w, http.StatusConflict, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

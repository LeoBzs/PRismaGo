package lift

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
func NewHandler(svc *Service, requireUser func(http.Handler) http.Handler) *Handler {
	return &Handler{svc: svc, requireUser: requireUser}
}

// Lifts always belong to the logged-in user; the owner comes from the session,
// never from the URL or body.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/me/lifts", h.requireUser(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/me/lifts", h.requireUser(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/me/lifts/{id}", h.requireUser(http.HandlerFunc(h.get)))
	mux.Handle("DELETE /api/v1/me/lifts/{id}", h.requireUser(http.HandlerFunc(h.delete)))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	lifts, err := h.svc.List(r.Context(), ListFilter{
		UserID:   session.UserID(r.Context()),
		Exercise: r.URL.Query().Get("exercise"),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, lifts)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in CreateInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	l, err := h.svc.Create(r.Context(), session.UserID(r.Context()), in)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, l)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	l, err := h.svc.Get(r.Context(), session.UserID(r.Context()), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, l)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), session.UserID(r.Context()), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeError maps domain errors to HTTP status codes. This is the only place
// in the feature that knows about both.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrUserNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

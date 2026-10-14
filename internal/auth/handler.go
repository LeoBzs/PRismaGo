package auth

import (
	"errors"
	"net/http"

	"github.com/LeoBzs/prismago/internal/httpx"
	"github.com/LeoBzs/prismago/internal/session"
	"github.com/LeoBzs/prismago/internal/user"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// These are the only unauthenticated API routes besides /healthz.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", h.register)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var in RegisterInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.Register(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in LoginInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := h.svc.Login(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

// logout is idempotent: 204 whether or not the token was valid, so a client
// can always clear its state.
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Logout(r.Context(), session.BearerToken(r)); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, ErrInvalidInput), errors.Is(err, user.ErrInvalidInput):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, user.ErrEmailTaken):
		httpx.Error(w, http.StatusConflict, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

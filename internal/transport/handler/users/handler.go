package users

import (
	"net/http"

	"avitohvk/internal/transport/utilhttp"

	"github.com/go-chi/chi/v5"
)

type UsersHandler struct{}

func New() *UsersHandler {
	return &UsersHandler{}
}

func (h *UsersHandler) RegisterPublicRoutes(r chi.Router) {
	r.Get("/users/{user_id}/items", h.Items)
}

func (h *UsersHandler) RegisterProtectedRoutes(r chi.Router) {
	r.Get("/users/{user_id}/deals", h.Deals)
}

func (h *UsersHandler) Items(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "users hello world"})
}

func (h *UsersHandler) Deals(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "users hello world"})
}

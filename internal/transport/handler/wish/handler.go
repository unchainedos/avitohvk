package wish

import (
	"net/http"

	"avitohvk/internal/transport/utilhttp"

	"github.com/go-chi/chi/v5"
)

type WishHandler struct{}

func New() *WishHandler {
	return &WishHandler{}
}

func (h *WishHandler) RegisterPublicRoutes(r chi.Router) {
	r.Get("/wishes/{wish_id}", h.Get)
}

func (h *WishHandler) RegisterProtectedRoutes(r chi.Router) {
	r.Post("/wishes/{wish_id}", h.Create)
	r.Patch("/wishes/{wish_id}", h.Update)
	r.Delete("/wishes/{wish_id}", h.Delete)
}

func (h *WishHandler) Create(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "wish hello world"})
}

func (h *WishHandler) Get(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "wish hello world"})
}

func (h *WishHandler) Update(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "wish hello world"})
}

func (h *WishHandler) Delete(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "wish hello world"})
}

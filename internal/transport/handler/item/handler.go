package item

import (
	"net/http"

	"avitohvk/internal/transport/utilhttp"

	"github.com/go-chi/chi/v5"
)

type ItemHandler struct{}

func New() *ItemHandler {
	return &ItemHandler{}
}

func (h *ItemHandler) RegisterPublicRoutes(r chi.Router) {
	r.Get("/items/{item_id}", h.Get)
}

func (h *ItemHandler) RegisterProtectedRoutes(r chi.Router) {
	r.Post("/items/{item_id}", h.Create)
	r.Patch("/items/{item_id}", h.Update)
	r.Delete("/items/{item_id}", h.Delete)
}

func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "item hello world"})
}

func (h *ItemHandler) Get(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "item hello world"})
}

func (h *ItemHandler) Update(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "item hello world"})
}

func (h *ItemHandler) Delete(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "item hello world"})
}

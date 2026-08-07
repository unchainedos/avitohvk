package user

import (
	"net/http"

	"avitohvk/internal/transport/utilhttp"

	"github.com/go-chi/chi/v5"
)

type UserHandler struct{}

func New() *UserHandler {
	return &UserHandler{}
}

func (h *UserHandler) RegisterRoutes(r chi.Router) {
	r.Post("/user", h.Create)
	r.Get("/user", h.Get)
	r.Patch("/user", h.Update)
	r.Delete("/user", h.Delete)
	r.Post("/user/add", h.AddItem)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "user hello world"})
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "user hello world"})
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "user hello world"})
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "user hello world"})
}

func (h *UserHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "user hello world"})
}

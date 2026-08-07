package chown

import (
	"net/http"

	"avitohvk/internal/transport/utilhttp"

	"github.com/go-chi/chi/v5"
)

type ChownHandler struct{}

func New() *ChownHandler {
	return &ChownHandler{}
}

func (h *ChownHandler) RegisterRoutes(r chi.Router) {
	r.Post("/chown", h.Chown)
}

func (h *ChownHandler) Chown(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "chown hello world"})
}

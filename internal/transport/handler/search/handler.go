package search

import (
	"net/http"

	"avitohvk/internal/transport/utilhttp"

	"github.com/go-chi/chi/v5"
)

type SearchHandler struct{}

func New() *SearchHandler {
	return &SearchHandler{}
}

func (h *SearchHandler) RegisterRoutes(r chi.Router) {
	r.Post("/search/{query}", h.Search)
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "search hello world"})
}

package deal

import (
	"net/http"

	"avitohvk/internal/transport/utilhttp"

	"github.com/go-chi/chi/v5"
)

type DealHandler struct{}

func New() *DealHandler {
	return &DealHandler{}
}

func (h *DealHandler) RegisterRoutes(r chi.Router) {
	r.Get("/deal/{deal_id}", h.Get)
}

func (h *DealHandler) Get(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "deal hello world"})
}

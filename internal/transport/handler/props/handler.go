package props

import (
	"net/http"

	"avitohvk/internal/transport/utilhttp"

	"github.com/go-chi/chi/v5"
)

type PropsHandler struct{}

func New() *PropsHandler {
	return &PropsHandler{}
}

func (h *PropsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/props/{deal_id}", h.Get)
	r.Patch("/props/{deal_id}", h.Update)
	r.Post("/props/{deal_id}", h.Create)
	r.Delete("/props/{deal_id}", h.Delete)
	r.Get("/props/users/{user_id}", h.GetByUser)
	r.Post("/props/approve/{deal_id}", h.Approve)
}

func (h *PropsHandler) Get(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "props hello world"})
}

func (h *PropsHandler) Update(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "props hello world"})
}

func (h *PropsHandler) Create(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "props hello world"})
}

func (h *PropsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "props hello world"})
}

func (h *PropsHandler) GetByUser(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "props hello world"})
}

func (h *PropsHandler) Approve(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "props hello world"})
}

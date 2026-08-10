package users

import (
	"avitohvk/internal/domain"
	"avitohvk/internal/dto"
	"avitohvk/internal/transport/utilhttp"
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type ItemLister interface {
	ListByUser(ctx context.Context, userID string) ([]domain.Item, error)
}
type UsersHandler struct {
	items ItemLister
}

func New(items ItemLister) *UsersHandler {
	return &UsersHandler{items: items}
}
func (h *UsersHandler) RegisterPublicRoutes(r chi.Router) {
	r.Get("/users/{user_id}/items", h.Items)
}
func (h *UsersHandler) RegisterProtectedRoutes(r chi.Router) {
	r.Get("/users/{user_id}/deals", h.Deals)
}
func (h *UsersHandler) Items(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	list, err := h.items.ListByUser(r.Context(), userID)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	resp := make([]dto.ItemResponse, 0, len(list))
	for _, it := range list {
		resp = append(resp, dto.ItemResponse{
			ID:          it.ID,
			AuthorID:    it.AuthorID,
			HolderID:    it.HolderID,
			Title:       it.Title,
			Description: it.Description,
			ImageURL:    it.ImageURL,
			Category:    it.Category,
			Unit:        it.Unit,
			Quantity:    it.Quantity,
			IsLocked:    it.IsLocked,
			CreatedAt:   it.CreatedAt,
		})
	}
	_ = utilhttp.WriteJSON(w, http.StatusOK, resp)
}
func (h *UsersHandler) Deals(w http.ResponseWriter, r *http.Request) {
	utilhttp.WriteJSON(w, http.StatusOK, map[string]string{"content": "users hello world"})
}

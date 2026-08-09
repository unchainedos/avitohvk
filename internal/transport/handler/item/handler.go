package item

import (
	"context"
	"net/http"

	"avitohvk/internal/domain"
	"avitohvk/internal/dto"
	statusErrors "avitohvk/internal/errors"
	"avitohvk/internal/transport/middleware"
	"avitohvk/internal/transport/utilhttp"

	"github.com/go-chi/chi/v5"
)

type ItemService interface {
	Create(ctx context.Context, userID string, in dto.CreateItemRequest) (string, error)
	Get(ctx context.Context, id string) (domain.Item, error)
	Update(ctx context.Context, actorID, itemID string, in dto.UpdateItemRequest) (domain.Item, error)
	Delete(ctx context.Context, actorID, itemID string) error
}
type ItemHandler struct {
	service ItemService
}

func New(service ItemService) *ItemHandler {
	return &ItemHandler{service: service}
}

func (h *ItemHandler) RegisterPublicRoutes(r chi.Router) {
	r.Get("/items/{item_id}", h.Get)
}

func (h *ItemHandler) RegisterProtectedRoutes(r chi.Router) {
	r.Post("/items", h.Create)
	r.Patch("/items/{item_id}", h.Update)
	r.Delete("/items/{item_id}", h.Delete)
}

func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	req, err := utilhttp.ReadFromJSON[dto.CreateItemRequest](r)
	if err != nil {
		utilhttp.WriteError(w, statusErrors.ErrBadRequest)
		return
	}
	id, err := h.service.Create(r.Context(), userID, *req)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusCreated, dto.CreateItemResponse{ID: id})
}

func (h *ItemHandler) Get(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "item_id")
	item, err := h.service.Get(r.Context(), itemID)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusOK, toItemResponse(item))
}

func (h *ItemHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	itemID := chi.URLParam(r, "item_id")
	req, err := utilhttp.ReadFromJSON[dto.UpdateItemRequest](r)
	if err != nil {
		utilhttp.WriteError(w, statusErrors.ErrBadRequest)
		return
	}
	item, err := h.service.Update(r.Context(), userID, itemID, *req)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusOK, toItemResponse(item))
}

func (h *ItemHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	itemID := chi.URLParam(r, "item_id")
	if err := h.service.Delete(r.Context(), userID, itemID); err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toItemResponse(item domain.Item) dto.ItemResponse {
	return dto.ItemResponse{
		ID:          item.ID,
		AuthorID:    item.AuthorID,
		HolderID:    item.HolderID,
		Title:       item.Title,
		Description: item.Description,
		ImageURL:    item.ImageURL,
		Category:    item.Category,
		Unit:        item.Unit,
		Quantity:    item.Quantity,
		IsLocked:    item.IsLocked,
		CreatedAt:   item.CreatedAt,
	}
}

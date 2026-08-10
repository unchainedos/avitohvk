package wish

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

type WishService interface {
	Create(ctx context.Context, userID, id, title string, description *string) (string, error)
	Get(ctx context.Context, id string) (domain.Wish, error)
	ListItemsByUser(ctx context.Context, userID string) ([]domain.Item, error)
	Update(ctx context.Context, actorID, wishID string, title, description *string) (domain.Wish, error)
	Delete(ctx context.Context, actorID, wishID string) error
}

type WishHandler struct {
	service WishService
}

func New(service WishService) *WishHandler {
	return &WishHandler{service: service}
}

func (h *WishHandler) RegisterPublicRoutes(r chi.Router) {
	r.Get("/wishes/{wish_id}", h.Get)
	r.Get("/wishes/{user_id}/items", h.ListItemsByUser)
}

func (h *WishHandler) RegisterProtectedRoutes(r chi.Router) {
	r.Post("/wishes/{wish_id}", h.Create)
	r.Patch("/wishes/{wish_id}", h.Update)
	r.Delete("/wishes/{wish_id}", h.Delete)
}

func (h *WishHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	wishID := chi.URLParam(r, "wish_id")
	req, err := utilhttp.ReadFromJSON[dto.CreateWishRequest](r)
	if err != nil {
		utilhttp.WriteError(w, statusErrors.ErrBadRequest)
		return
	}
	id, err := h.service.Create(r.Context(), userID, wishID, req.Title, req.Description)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusCreated, dto.CreateWishResponse{ID: id})
}

func (h *WishHandler) Get(w http.ResponseWriter, r *http.Request) {
	wishID := chi.URLParam(r, "wish_id")
	wish, err := h.service.Get(r.Context(), wishID)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusOK, toWishResponse(wish))
}

func (h *WishHandler) ListItemsByUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	items, err := h.service.ListItemsByUser(r.Context(), userID)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	resp := make([]dto.ItemResponse, 0, len(items))
	for _, it := range items {
		resp = append(resp, dto.ItemResponse{
			ID: it.ID, AuthorID: it.AuthorID, HolderID: it.HolderID,
			Title: it.Title, Description: it.Description, ImageURL: it.ImageURL,
			Category: it.Category, Unit: it.Unit, Quantity: it.Quantity,
			IsLocked: it.IsLocked, CreatedAt: it.CreatedAt,
		})
	}
	_ = utilhttp.WriteJSON(w, http.StatusOK, resp)
}

func (h *WishHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	wishID := chi.URLParam(r, "wish_id")
	req, err := utilhttp.ReadFromJSON[dto.UpdateWishRequest](r)
	if err != nil {
		utilhttp.WriteError(w, statusErrors.ErrBadRequest)
		return
	}
	wish, err := h.service.Update(r.Context(), userID, wishID, req.Title, req.Description)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusOK, toWishResponse(wish))
}

func (h *WishHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	wishID := chi.URLParam(r, "wish_id")
	if err := h.service.Delete(r.Context(), userID, wishID); err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toWishResponse(wish domain.Wish) dto.WishResponse {
	return dto.WishResponse{
		ID:          wish.ID,
		UserID:      wish.UserID,
		Title:       wish.Title,
		Description: wish.Description,
		CreatedAt:   wish.CreatedAt,
	}
}

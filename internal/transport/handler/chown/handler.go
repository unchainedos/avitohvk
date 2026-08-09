package chown

import (
	"context"
	"net/http"

	"avitohvk/internal/dto"
	statusErrors "avitohvk/internal/errors"
	"avitohvk/internal/transport/middleware"
	"avitohvk/internal/transport/utilhttp"

	"github.com/go-chi/chi/v5"
)

type ChownService interface {
	Chown(ctx context.Context, actorID, itemID string, req dto.ChownRequest) (dto.ChownResponse, error)
}

type ChownHandler struct {
	service ChownService
}

func New(service ChownService) *ChownHandler {
	return &ChownHandler{service: service}
}

func (h *ChownHandler) RegisterRoutes(r chi.Router) {
	r.Post("/chown/{item_id}", h.Chown)
}

func (h *ChownHandler) Chown(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	itemID := chi.URLParam(r, "item_id")
	req, err := utilhttp.ReadFromJSON[dto.ChownRequest](r)
	if err != nil {
		utilhttp.WriteError(w, statusErrors.ErrBadRequest)
		return
	}
	resp, err := h.service.Chown(r.Context(), actorID, itemID, *req)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusCreated, resp)
}

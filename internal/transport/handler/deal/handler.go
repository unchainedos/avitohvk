package deal

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

type Service interface {
	GetByID(ctx context.Context, id string) (domain.Deal, error)
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/deal/{deal_id}", h.Get)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if _, ok := middleware.UserIDFromContext(r.Context()); !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	dealID := chi.URLParam(r, "deal_id")
	d, err := h.service.GetByID(r.Context(), dealID)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusOK, toDealResponse(&d))
}

func toDealResponse(d *domain.Deal) dto.DealResponse {
	return dto.DealResponse{
		ID:                       d.ID,
		RootItemID:               d.RootItemID,
		CreatorID:                d.CreatorID,
		Status:                   string(d.Status),
		NegotiationWindowSeconds: int64(d.NegotiationWindow.Seconds()),
		DeadlineAt:               d.DeadlineAt,
		CreatedAt:                d.CreatedAt,
	}
}

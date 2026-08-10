package chown

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
	Chown(ctx context.Context, actorID, itemID string, req dto.CreateProposalRequest) (domain.Proposal, error)
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/chown/{item_id}", h.Chown)
}

func (h *Handler) Chown(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	itemID := chi.URLParam(r, "item_id")
	req, err := utilhttp.ReadFromJSON[dto.CreateProposalRequest](r)
	if err != nil {
		utilhttp.WriteError(w, statusErrors.ErrBadRequest)
		return
	}
	p, err := h.service.Chown(r.Context(), actorID, itemID, *req)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusCreated, toProposalResponse(&p))
}

func toProposalResponse(p *domain.Proposal) dto.ProposalResponse {
	return dto.ProposalResponse{
		DealID:        p.DealID,
		TransactionID: p.TransactionID,
		ParticipantID: p.ParticipantID,
		ItemID:        p.ItemID,
		ToUserID:      p.ToUserID,
		Quantity:      p.Quantity,
		Status:        string(p.Status),
		UpdatedAt:     p.UpdatedAt,
	}
}

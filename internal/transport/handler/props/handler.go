package props

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

type PropsService interface {
	CreateDeal(ctx context.Context, actorID string, req dto.CreateDealRequest) (domain.Proposal, error)
	CreateProposal(ctx context.Context, actorID, dealID string, req dto.CreateProposalRequest) (domain.Proposal, error)
	GetProposal(ctx context.Context, actorID, dealID string) (domain.Proposal, error)
	UpdateProposal(ctx context.Context, actorID, dealID string, upd domain.ProposalUpdate) (domain.Proposal, error)
	WithdrawProposal(ctx context.Context, actorID, dealID string) error
	ListForUser(ctx context.Context, actorID, userID string) ([]domain.Proposal, error)
	Approve(ctx context.Context, actorID, dealID string) (domain.Proposal, error)
}

type PropsHandler struct {
	service PropsService
}

func New(service PropsService) *PropsHandler {
	return &PropsHandler{service: service}
}

func (h *PropsHandler) RegisterRoutes(r chi.Router) {
	r.Post("/props/", h.CreateDeal)
	r.Get("/props/{deal_id}", h.Get)
	r.Patch("/props/{deal_id}", h.Update)
	r.Post("/props/{deal_id}", h.Create)
	r.Delete("/props/{deal_id}", h.Delete)
	r.Get("/props/users/{user_id}", h.GetByUser)
	r.Post("/props/approve/{deal_id}", h.Approve)
}

func (h *PropsHandler) CreateDeal(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	req, err := utilhttp.ReadFromJSON[dto.CreateDealRequest](r)
	if err != nil {
		utilhttp.WriteError(w, statusErrors.ErrBadRequest)
		return
	}
	p, err := h.service.CreateDeal(r.Context(), actorID, *req)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusCreated, toProposalResponse(p))
}

func (h *PropsHandler) Create(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	dealID := chi.URLParam(r, "deal_id")
	req, err := utilhttp.ReadFromJSON[dto.CreateProposalRequest](r)
	if err != nil {
		utilhttp.WriteError(w, statusErrors.ErrBadRequest)
		return
	}
	p, err := h.service.CreateProposal(r.Context(), actorID, dealID, *req)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusCreated, toProposalResponse(p))
}

func (h *PropsHandler) Get(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	dealID := chi.URLParam(r, "deal_id")
	p, err := h.service.GetProposal(r.Context(), actorID, dealID)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusOK, toProposalResponse(p))
}

func (h *PropsHandler) Update(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	dealID := chi.URLParam(r, "deal_id")
	req, err := utilhttp.ReadFromJSON[dto.UpdateProposalRequest](r)
	if err != nil {
		utilhttp.WriteError(w, statusErrors.ErrBadRequest)
		return
	}
	p, err := h.service.UpdateProposal(r.Context(), actorID, dealID, domain.ProposalUpdate{
		ItemID:   req.ItemID,
		Quantity: req.Quantity,
	})
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusOK, toProposalResponse(p))
}

func (h *PropsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	dealID := chi.URLParam(r, "deal_id")
	if err := h.service.WithdrawProposal(r.Context(), actorID, dealID); err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PropsHandler) GetByUser(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	userID := chi.URLParam(r, "user_id")
	list, err := h.service.ListForUser(r.Context(), actorID, userID)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	resp := dto.ProposalListResponse{Proposals: make([]dto.ProposalResponse, 0, len(list))}
	for _, p := range list {
		resp.Proposals = append(resp.Proposals, toProposalResponse(p))
	}
	_ = utilhttp.WriteJSON(w, http.StatusOK, resp)
}

func (h *PropsHandler) Approve(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	dealID := chi.URLParam(r, "deal_id")
	p, err := h.service.Approve(r.Context(), actorID, dealID)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusOK, toProposalResponse(p))
}

func toProposalResponse(p domain.Proposal) dto.ProposalResponse {
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

package chown

import (
	"context"
	"fmt"

	"avitohvk/internal/domain"
	"avitohvk/internal/dto"
	statusErrors "avitohvk/internal/errors"
)

type Repository interface {
	EnsureWish(ctx context.Context, userID, itemID string) error
}

type ProposalService interface {
	CreateDeal(ctx context.Context, actorID string, req dto.CreateDealRequest) (domain.Proposal, error)
	CreateProposal(ctx context.Context, actorID, dealID string, req dto.CreateProposalRequest) (domain.Proposal, error)
	FindOpenDealAsRecipient(ctx context.Context, itemID, participantID string) (dealID string, found bool, err error)
}

type Service struct {
	repo      Repository
	proposals ProposalService
}

func NewService(repo Repository, proposals ProposalService) *Service {
	return &Service{repo: repo, proposals: proposals}
}

func (s *Service) Chown(ctx context.Context, actorID, itemID string, req dto.CreateProposalRequest) (domain.Proposal, error) {
	if itemID == "" {
		return domain.Proposal{}, fmt.Errorf("%w: item_id required", statusErrors.ErrBadRequest)
	}
	if req.ItemID == "" {
		return domain.Proposal{}, fmt.Errorf("%w: item_id required for offered item", statusErrors.ErrBadRequest)
	}
	if req.Quantity <= 0 {
		return domain.Proposal{}, fmt.Errorf("%w: quantity must be positive", statusErrors.ErrBadRequest)
	}
	if req.ItemID == itemID {
		return domain.Proposal{}, fmt.Errorf("%w: cannot offer the item you are chowning", statusErrors.ErrBadRequest)
	}

	dealID, found, err := s.proposals.FindOpenDealAsRecipient(ctx, itemID, actorID)
	if err != nil {
		return domain.Proposal{}, err
	}
	if found {
		return s.proposals.CreateProposal(ctx, actorID, dealID, dto.CreateProposalRequest{
			ItemID: req.ItemID, Quantity: req.Quantity,
		})
	}

	p, err := s.proposals.CreateDeal(ctx, actorID, dto.CreateDealRequest{
		RootItemID: itemID, ItemID: req.ItemID, Quantity: req.Quantity,
	})
	if err != nil {
		return domain.Proposal{}, err
	}

	if err := s.repo.EnsureWish(ctx, actorID, itemID); err != nil {
		return domain.Proposal{}, err
	}

	return p, nil
}

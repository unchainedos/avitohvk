package proposal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"avitohvk/internal/domain"
	"avitohvk/internal/dto"
	statusErrors "avitohvk/internal/errors"
	dealrepo "avitohvk/internal/repository/deal"
	proposalrepo "avitohvk/internal/repository/proposal"
)

const minParticipants = 2

type DealRepository interface {
	Create(ctx context.Context, rootItemID, creatorID string, participants int, deadlineAt time.Time) (domain.Deal, error)
	GetByID(ctx context.Context, id string) (domain.Deal, error)
	UpdateStatus(ctx context.Context, id string, status domain.DealStatus) (domain.Deal, error)
}

type ProposalRepository interface {
	Create(ctx context.Context, dealID, participantID, itemID string, quantity float64) (domain.Proposal, error)
	GetByDealAndParticipant(ctx context.Context, dealID, participantID string) (domain.Proposal, error)
	Update(ctx context.Context, dealID, participantID string, upd domain.ProposalUpdate) (domain.Proposal, error)
	SetStatus(ctx context.Context, dealID, participantID string, status domain.ProposalStatus) (domain.Proposal, error)
	ListForUser(ctx context.Context, userID string) ([]domain.Proposal, error)
	AllAccepted(ctx context.Context, dealID string) (bool, error)
	CountForDeal(ctx context.Context, dealID string) (int, error)
	TryLockChain(ctx context.Context, dealID string) error
	UnlockAllForDeal(ctx context.Context, dealID string) error
}

type Service struct {
	deals     DealRepository
	proposals ProposalRepository
}

func NewService(deals DealRepository, proposals ProposalRepository) *Service {
	return &Service{deals: deals, proposals: proposals}
}

func (s *Service) GetByID(ctx context.Context, id string) (domain.Deal, error) {
	d, err := s.deals.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, dealrepo.ErrNotFound) {
			return domain.Deal{}, statusErrors.ErrNotFound
		}
		return domain.Deal{}, err
	}
	return d, nil
}

func (s *Service) CreateDeal(ctx context.Context, actorID string, req dto.CreateDealRequest) (domain.Proposal, error) {
	if err := validateOffer(req.ItemID, req.Quantity); err != nil {
		return domain.Proposal{}, err
	}
	if req.RootItemID == "" {
		return domain.Proposal{}, fmt.Errorf("%w: root_item_id required", statusErrors.ErrBadRequest)
	}
	if req.Participants < minParticipants {
		return domain.Proposal{}, fmt.Errorf("%w: participants must be at least %d", statusErrors.ErrBadRequest, minParticipants)
	}
	if !req.DeadlineAt.After(time.Now()) {
		return domain.Proposal{}, fmt.Errorf("%w: deadline_at must be in the future", statusErrors.ErrBadRequest)
	}

	d, err := s.deals.Create(ctx, req.RootItemID, actorID, req.Participants, req.DeadlineAt)
	if err != nil {
		if errors.Is(err, dealrepo.ErrRootItemNotFound) {
			return domain.Proposal{}, fmt.Errorf("%w: root item not found", statusErrors.ErrNotFound)
		}
		return domain.Proposal{}, err
	}

	return s.createProposal(ctx, d.ID, actorID, req.ItemID, req.Quantity)
}

func (s *Service) CreateProposal(ctx context.Context, actorID, dealID string, req dto.CreateProposalRequest) (domain.Proposal, error) {
	if err := validateOffer(req.ItemID, req.Quantity); err != nil {
		return domain.Proposal{}, err
	}

	d, err := s.GetByID(ctx, dealID)
	if err != nil {
		return domain.Proposal{}, err
	}
	if err := s.ensureDealOpen(ctx, d); err != nil {
		return domain.Proposal{}, err
	}

	count, err := s.proposals.CountForDeal(ctx, dealID)
	if err != nil {
		return domain.Proposal{}, err
	}
	if count >= d.Participants {
		return domain.Proposal{}, fmt.Errorf("%w: deal already has enough participants", statusErrors.ErrConflict)
	}

	if _, err := s.proposals.GetByDealAndParticipant(ctx, dealID, actorID); err == nil {
		return domain.Proposal{}, fmt.Errorf("%w: proposal already exists for this deal", statusErrors.ErrConflict)
	} else if !errors.Is(err, proposalrepo.ErrNotFound) {
		return domain.Proposal{}, err
	}

	return s.createProposal(ctx, dealID, actorID, req.ItemID, req.Quantity)
}

func (s *Service) createProposal(ctx context.Context, dealID, actorID, itemID string, quantity float64) (domain.Proposal, error) {
	p, err := s.proposals.Create(ctx, dealID, actorID, itemID, quantity)
	if err != nil {
		if errors.Is(err, proposalrepo.ErrItemNotFound) {
			return domain.Proposal{}, fmt.Errorf("%w: item not found", statusErrors.ErrNotFound)
		}
		if errors.Is(err, proposalrepo.ErrRecipientNotFound) {
			return domain.Proposal{}, fmt.Errorf("%w: no one wishes for this item", statusErrors.ErrNotFound)
		}
		if errors.Is(err, proposalrepo.ErrNotItemHolder) {
			return domain.Proposal{}, fmt.Errorf("%w: you do not hold this item", statusErrors.ErrConflict)
		}
		if errors.Is(err, proposalrepo.ErrAlreadyProposed) {
			return domain.Proposal{}, fmt.Errorf("%w: proposal already exists for this deal", statusErrors.ErrConflict)
		}
		if errors.Is(err, proposalrepo.ErrDealFull) {
			return domain.Proposal{}, fmt.Errorf("%w: deal already has enough participants", statusErrors.ErrConflict)
		}
		return domain.Proposal{}, err
	}
	return p, nil
}

func (s *Service) GetProposal(ctx context.Context, actorID, dealID string) (domain.Proposal, error) {
	p, err := s.proposals.GetByDealAndParticipant(ctx, dealID, actorID)
	if err != nil {
		if errors.Is(err, proposalrepo.ErrNotFound) {
			return domain.Proposal{}, statusErrors.ErrNotFound
		}
		return domain.Proposal{}, err
	}
	return p, nil
}

func (s *Service) UpdateProposal(ctx context.Context, actorID, dealID string, upd domain.ProposalUpdate) (domain.Proposal, error) {
	if upd.Quantity != nil && *upd.Quantity <= 0 {
		return domain.Proposal{}, fmt.Errorf("%w: quantity must be positive", statusErrors.ErrBadRequest)
	}

	p, err := s.proposals.Update(ctx, dealID, actorID, upd)
	if err != nil {
		if errors.Is(err, proposalrepo.ErrNotFound) {
			return domain.Proposal{}, statusErrors.ErrNotFound
		}
		if errors.Is(err, proposalrepo.ErrItemNotFound) {
			return domain.Proposal{}, fmt.Errorf("%w: item not found", statusErrors.ErrNotFound)
		}
		if errors.Is(err, proposalrepo.ErrRecipientNotFound) {
			return domain.Proposal{}, fmt.Errorf("%w: no one wishes for this item", statusErrors.ErrNotFound)
		}
		if errors.Is(err, proposalrepo.ErrNotItemHolder) {
			return domain.Proposal{}, fmt.Errorf("%w: you do not hold this item", statusErrors.ErrConflict)
		}
		if errors.Is(err, proposalrepo.ErrNotPending) {
			return domain.Proposal{}, fmt.Errorf("%w: proposal is not pending", statusErrors.ErrConflict)
		}
		return domain.Proposal{}, err
	}
	return p, nil
}

func (s *Service) WithdrawProposal(ctx context.Context, actorID, dealID string) error {
	d, err := s.GetByID(ctx, dealID)
	if err != nil {
		return err
	}
	if d.Status != domain.DealStatusPending {
		return fmt.Errorf("%w: deal is not open", statusErrors.ErrConflict)
	}

	_, err = s.proposals.SetStatus(ctx, dealID, actorID, domain.ProposalStatusDeclined)
	if err != nil {
		if errors.Is(err, proposalrepo.ErrNotFound) {
			return statusErrors.ErrNotFound
		}
		if errors.Is(err, proposalrepo.ErrNotPending) {
			return fmt.Errorf("%w: proposal is not pending", statusErrors.ErrConflict)
		}
		return err
	}
	return nil
}

func (s *Service) ListForUser(ctx context.Context, actorID, userID string) ([]domain.Proposal, error) {
	if actorID != userID {
		return nil, statusErrors.ErrUnauthorized
	}
	list, err := s.proposals.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (s *Service) Approve(ctx context.Context, actorID, dealID string) (domain.Proposal, error) {
	d, err := s.GetByID(ctx, dealID)
	if err != nil {
		return domain.Proposal{}, err
	}
	if err := s.ensureDealOpen(ctx, d); err != nil {
		return domain.Proposal{}, err
	}

	p, err := s.proposals.SetStatus(ctx, dealID, actorID, domain.ProposalStatusAccepted)
	if err != nil {
		if errors.Is(err, proposalrepo.ErrNotFound) {
			return domain.Proposal{}, statusErrors.ErrNotFound
		}
		if errors.Is(err, proposalrepo.ErrNotPending) {
			return domain.Proposal{}, fmt.Errorf("%w: proposal is not pending", statusErrors.ErrConflict)
		}
		return domain.Proposal{}, err
	}

	if err := s.proposals.TryLockChain(ctx, dealID); err != nil {
		return domain.Proposal{}, err
	}

	allAccepted, err := s.proposals.AllAccepted(ctx, dealID)
	if err != nil {
		return domain.Proposal{}, err
	}
	if allAccepted {
		if _, err := s.deals.UpdateStatus(ctx, dealID, domain.DealStatusConfirmed); err != nil {
			return domain.Proposal{}, err
		}
	}

	return p, nil
}

func (s *Service) ensureDealOpen(ctx context.Context, d domain.Deal) error {
	if d.Status != domain.DealStatusPending {
		return fmt.Errorf("%w: deal is not open", statusErrors.ErrConflict)
	}
	if !d.DeadlineAt.After(time.Now()) {
		if _, err := s.deals.UpdateStatus(ctx, d.ID, domain.DealStatusCancelled); err != nil {
			return err
		}
		if err := s.proposals.UnlockAllForDeal(ctx, d.ID); err != nil {
			return err
		}
		return fmt.Errorf("%w: deal deadline has passed", statusErrors.ErrConflict)
	}
	return nil
}

func validateOffer(itemID string, quantity float64) error {
	if itemID == "" {
		return fmt.Errorf("%w: item_id required", statusErrors.ErrBadRequest)
	}
	if quantity <= 0 {
		return fmt.Errorf("%w: quantity must be positive", statusErrors.ErrBadRequest)
	}
	return nil
}

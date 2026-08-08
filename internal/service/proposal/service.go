package proposal

import (
	"context"
	"time"

	"avitohvk/internal/domain"
	"avitohvk/internal/dto"
	// dealrepo "avitohvk/internal/repository/deal"
	// proposalrepo "avitohvk/internal/repository/proposal"
)

type DealRepository interface {
	Create(ctx context.Context, rootItemID string, participants int, deadlineAt time.Time) (domain.Deal, error)
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
}

type Service struct {
	deals     DealRepository
	proposals ProposalRepository
}

func NewService(deals DealRepository, proposals ProposalRepository) *Service {
	return &Service{deals: deals, proposals: proposals}
}

func (s *Service) GetByID(ctx context.Context, id string) (domain.Deal, error) {
	return domain.Deal{}, nil
}

func (s *Service) CreateDeal(ctx context.Context, actorID string, req dto.CreateDealRequest) (domain.Proposal, error) {
	return domain.Proposal{}, nil
}

func (s *Service) CreateProposal(ctx context.Context, actorID, dealID string, req dto.CreateProposalRequest) (domain.Proposal, error) {
	return domain.Proposal{}, nil
}

func (s *Service) createProposal(ctx context.Context, dealID, actorID, itemID string, quantity float64) (domain.Proposal, error) {
	return domain.Proposal{}, nil
}

func (s *Service) GetProposal(ctx context.Context, actorID, dealID string) (domain.Proposal, error) {
	return domain.Proposal{}, nil
}

func (s *Service) UpdateProposal(ctx context.Context, actorID, dealID string, upd domain.ProposalUpdate) (domain.Proposal, error) {
	return domain.Proposal{}, nil
}

func (s *Service) WithdrawProposal(ctx context.Context, actorID, dealID string) error {
	return nil
}

func (s *Service) ListForUser(ctx context.Context, actorID, userID string) ([]domain.Proposal, error) {
	return []domain.Proposal{}, nil
}

func (s *Service) Approve(ctx context.Context, actorID, dealID string) (domain.Proposal, error) {
	return domain.Proposal{}, nil
}

func validateOffer(itemID string, quantity float64) error {
	return nil
}

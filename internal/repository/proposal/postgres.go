package proposal

import (
	"context"
	"errors"

	"avitohvk/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("proposal not found")
	ErrItemNotFound = errors.New("item not found")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, dealID, participantID, itemID string, quantity float64) (domain.Proposal, error) {
	return domain.Proposal{}, nil
}

func (r *Repository) GetByDealAndParticipant(ctx context.Context, dealID, participantID string) (domain.Proposal, error) {
	return domain.Proposal{}, nil
}

func (r *Repository) Update(ctx context.Context, dealID, participantID string, upd domain.ProposalUpdate) (domain.Proposal, error) {
	return domain.Proposal{}, nil
}

func (r *Repository) SetStatus(ctx context.Context, dealID, participantID string, status domain.ProposalStatus) (domain.Proposal, error) {
	return domain.Proposal{}, nil
}

func (r *Repository) ListForUser(ctx context.Context, userID string) ([]domain.Proposal, error) {
	return []domain.Proposal{}, nil
}

func (r *Repository) AllAccepted(ctx context.Context, dealID string) (bool, error) {
	return false, nil
}

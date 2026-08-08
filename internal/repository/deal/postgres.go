package deal

import (
	"context"
	"errors"
	"time"

	"avitohvk/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound         = errors.New("deal not found")
	ErrRootItemNotFound = errors.New("root item not found")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, rootItemID string, participants int, deadlineAt time.Time) (domain.Deal, error) {
	return domain.Deal{}, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (domain.Deal, error) {
	return domain.Deal{}, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status domain.DealStatus) (domain.Deal, error) {
	return domain.Deal{}, nil
}

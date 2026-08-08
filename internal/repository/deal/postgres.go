package deal

import (
	"context"
	"errors"
	"time"

	"avitohvk/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func (r *Repository) Create(ctx context.Context, rootItemID, creatorID string, negotiationWindow time.Duration) (domain.Deal, error) {
	const q = `
		INSERT INTO chain_deals (root_item_id, creator_id, negotiation_window_seconds)
		VALUES ($1::uuid, $2::uuid, $3)
		RETURNING id::text, root_item_id::text, creator_id::text, status,
		          negotiation_window_seconds, deadline_at, created_at, updated_at
	`
	seconds := int64(negotiationWindow.Seconds())
	d, err := r.scanDeal(r.pool.QueryRow(ctx, q, rootItemID, creatorID, seconds))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return domain.Deal{}, ErrRootItemNotFound
		}
		return domain.Deal{}, err
	}
	return d, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (domain.Deal, error) {
	const q = `
		SELECT id::text, root_item_id::text, creator_id::text, status,
		       negotiation_window_seconds, deadline_at, created_at, updated_at
		FROM chain_deals
		WHERE id = $1::uuid
	`
	return r.scanDeal(r.pool.QueryRow(ctx, q, id))
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status domain.DealStatus) (domain.Deal, error) {
	const q = `
		UPDATE chain_deals
		SET status = $2
		WHERE id = $1::uuid
		RETURNING id::text, root_item_id::text, creator_id::text, status,
		          negotiation_window_seconds, deadline_at, created_at, updated_at
	`
	return r.scanDeal(r.pool.QueryRow(ctx, q, id, status))
}

func (r *Repository) scanDeal(row pgx.Row) (domain.Deal, error) {
	var d domain.Deal
	var seconds int64
	err := row.Scan(
		&d.ID, &d.RootItemID, &d.CreatorID, &d.Status,
		&seconds, &d.DeadlineAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Deal{}, ErrNotFound
	}
	if err != nil {
		return domain.Deal{}, err
	}
	d.NegotiationWindow = time.Duration(seconds) * time.Second
	return d, nil
}

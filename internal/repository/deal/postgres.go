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
	ErrCreatorNotFound  = errors.New("creator not found")
)

const (
	fkConstraintRootItem = "fk_chain_deals_root_item"
	fkConstraintCreator  = "chain_deals_creator_id_fkey"
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
			switch pgErr.ConstraintName {
			case fkConstraintRootItem:
				return domain.Deal{}, ErrRootItemNotFound
			case fkConstraintCreator:
				return domain.Deal{}, ErrCreatorNotFound
			}
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

func (r *Repository) LockDeal(ctx context.Context, dealID string) (func(context.Context), error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended($1, 0))`, dealID); err != nil {
		conn.Release()
		return nil, err
	}
	return func(unlockCtx context.Context) {
		_, _ = conn.Exec(unlockCtx, `SELECT pg_advisory_unlock(hashtextextended($1, 0))`, dealID)
		conn.Release()
	}, nil
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

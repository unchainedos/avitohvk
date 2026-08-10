package chown

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) EnsureWish(ctx context.Context, userID, itemID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const qExists = `
		SELECT EXISTS (
			SELECT 1 FROM wish_items wi
			JOIN wishes w ON w.id = wi.wish_id
			WHERE w.user_id = $1::uuid AND wi.item_id = $2::uuid
		)
	`
	var exists bool
	if err := tx.QueryRow(ctx, qExists, userID, itemID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return tx.Commit(ctx)
	}

	const qCreateWish = `
		INSERT INTO wishes (user_id, title)
		VALUES ($1::uuid, 'chown claim')
		RETURNING id::text
	`
	var wishID string
	if err := tx.QueryRow(ctx, qCreateWish, userID).Scan(&wishID); err != nil {
		return err
	}

	const qLinkWish = `INSERT INTO wish_items (wish_id, item_id) VALUES ($1::uuid, $2::uuid)`
	if _, err := tx.Exec(ctx, qLinkWish, wishID, itemID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) CompleteTransfer(ctx context.Context, itemID, toUserID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const qTransfer = `
		UPDATE items
		SET holder_id = $2::uuid, is_locked = false, locked_by_deal_id = NULL
		WHERE id = $1::uuid
	`
	if _, err := tx.Exec(ctx, qTransfer, itemID, toUserID); err != nil {
		return err
	}

	const qWish = `
		SELECT wi.wish_id::text
		FROM wish_items wi
		JOIN wishes w ON w.id = wi.wish_id
		WHERE wi.item_id = $1::uuid AND w.user_id = $2::uuid
		ORDER BY w.created_at
		LIMIT 1
	`
	var wishID string
	err = tx.QueryRow(ctx, qWish, itemID, toUserID).Scan(&wishID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err == nil {
		const qUnlink = `DELETE FROM wish_items WHERE wish_id = $1::uuid AND item_id = $2::uuid`
		if _, err := tx.Exec(ctx, qUnlink, wishID, itemID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

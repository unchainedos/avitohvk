package chown

import (
	"context"
	"errors"
	"time"

	"avitohvk/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrItemNotFound      = errors.New("item not found")
	ErrNotItemHolder     = errors.New("participant does not hold this item")
	ErrRecipientNotFound = errors.New("no one wishes for this item")
	ErrOwnItem           = errors.New("cannot chown your own item")
	ErrItemLocked        = errors.New("someone else already has exclusive rights to this item")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Chown(ctx context.Context, actorID, itemID string, offers []domain.OfferedItem) (domain.Chown, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Chown{}, err
	}
	defer tx.Rollback(ctx)

	if err := claimExclusiveRights(ctx, tx, itemID, actorID); err != nil {
		return domain.Chown{}, err
	}

	hops := make([]domain.ChownHop, 0, len(offers))
	for _, offer := range offers {
		toUserID, err := tossItem(ctx, tx, actorID, offer.ItemID, offer.Quantity)
		if err != nil {
			return domain.Chown{}, err
		}
		hops = append(hops, domain.ChownHop{ItemID: offer.ItemID, Quantity: offer.Quantity, ToUserID: toUserID})
	}

	var createdAt time.Time
	if err := tx.QueryRow(ctx, `SELECT now()`).Scan(&createdAt); err != nil {
		return domain.Chown{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Chown{}, err
	}

	return domain.Chown{
		ItemID:     itemID,
		FromUserID: actorID,
		Hops:       hops,
		CreatedAt:  createdAt,
	}, nil
}

func claimExclusiveRights(ctx context.Context, tx pgx.Tx, itemID, actorID string) error {
	const q = `SELECT holder_id::text, is_locked FROM items WHERE id = $1::uuid FOR UPDATE`
	var holderID string
	var isLocked bool
	if err := tx.QueryRow(ctx, q, itemID).Scan(&holderID, &isLocked); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrItemNotFound
		}
		return err
	}
	if holderID == actorID {
		return ErrOwnItem
	}
	if isLocked {
		return ErrItemLocked
	}

	const qLock = `UPDATE items SET is_locked = true WHERE id = $1::uuid`
	if _, err := tx.Exec(ctx, qLock, itemID); err != nil {
		return err
	}

	return linkWish(ctx, tx, actorID, itemID)
}

func linkWish(ctx context.Context, tx pgx.Tx, actorID, itemID string) error {
	const qExists = `
		SELECT EXISTS (
			SELECT 1 FROM wish_items wi
			JOIN wishes w ON w.id = wi.wish_id
			WHERE w.user_id = $1::uuid AND wi.item_id = $2::uuid
		)
	`
	var exists bool
	if err := tx.QueryRow(ctx, qExists, actorID, itemID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}

	const qCreateWish = `
		INSERT INTO wishes (user_id, title)
		VALUES ($1::uuid, 'chown claim')
		RETURNING id::text
	`
	var wishID string
	if err := tx.QueryRow(ctx, qCreateWish, actorID).Scan(&wishID); err != nil {
		return err
	}

	const qLinkWish = `INSERT INTO wish_items (wish_id, item_id) VALUES ($1::uuid, $2::uuid)`
	_, err := tx.Exec(ctx, qLinkWish, wishID, itemID)
	return err
}

func tossItem(ctx context.Context, tx pgx.Tx, actorID, itemID string, quantity float64) (string, error) {
	const qHolder = `SELECT holder_id::text, is_locked FROM items WHERE id = $1::uuid FOR UPDATE`
	var holderID string
	var isLocked bool
	if err := tx.QueryRow(ctx, qHolder, itemID).Scan(&holderID, &isLocked); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrItemNotFound
		}
		return "", err
	}
	if holderID != actorID {
		return "", ErrNotItemHolder
	}
	if isLocked {
		return "", ErrItemLocked
	}

	const qRecipient = `
		SELECT wi.wish_id::text, w.user_id::text
		FROM wish_items wi
		JOIN wishes w ON w.id = wi.wish_id
		WHERE wi.item_id = $1::uuid AND w.user_id <> $2::uuid
		ORDER BY w.created_at
		LIMIT 1
	`
	var wishID, toUserID string
	if err := tx.QueryRow(ctx, qRecipient, itemID, actorID).Scan(&wishID, &toUserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrRecipientNotFound
		}
		return "", err
	}

	const qTransfer = `UPDATE items SET holder_id = $2::uuid WHERE id = $1::uuid`
	if _, err := tx.Exec(ctx, qTransfer, itemID, toUserID); err != nil {
		return "", err
	}

	const qUnlink = `DELETE FROM wish_items WHERE wish_id = $1::uuid AND item_id = $2::uuid`
	if _, err := tx.Exec(ctx, qUnlink, wishID, itemID); err != nil {
		return "", err
	}

	const qTx = `
		INSERT INTO transactions (item_id, from_user, to_user, quantity)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4)
	`
	if _, err := tx.Exec(ctx, qTx, itemID, actorID, toUserID, quantity); err != nil {
		return "", err
	}

	return toUserID, nil
}

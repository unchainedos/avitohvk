package proposal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"avitohvk/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound          = errors.New("proposal not found")
	ErrItemNotFound      = errors.New("item not found")
	ErrRecipientNotFound = errors.New("no one wishes for this item")
	ErrNotItemHolder     = errors.New("participant does not hold this item")
	ErrAlreadyProposed   = errors.New("proposal already exists for this deal")
	ErrChainClosed       = errors.New("chain is already closed to new participants")
	ErrNotPending        = errors.New("proposal is not pending")
	ErrOutOfOrder        = errors.New("recipient has not accepted yet")
	ErrItemLocked        = errors.New("item is locked by another confirmed deal")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, dealID, participantID, itemID string, quantity float64) (domain.Proposal, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Proposal{}, err
	}
	defer tx.Rollback(ctx)

	const qDeal = `
		SELECT (root_item_id = $2::uuid), negotiation_window_seconds, deadline_at
		FROM chain_deals
		WHERE id = $1::uuid
		FOR UPDATE
	`
	var isRootItem bool
	var negotiationWindowSeconds int64
	var deadlineAt *time.Time
	if err := tx.QueryRow(ctx, qDeal, dealID, itemID).Scan(&isRootItem, &negotiationWindowSeconds, &deadlineAt); err != nil {
		return domain.Proposal{}, err
	}
	if deadlineAt != nil {
		return domain.Proposal{}, ErrChainClosed
	}

	if isRootItem {
		deadline := time.Now().Add(time.Duration(negotiationWindowSeconds) * time.Second)
		const qStart = `UPDATE chain_deals SET deadline_at = $2 WHERE id = $1::uuid`
		if _, err := tx.Exec(ctx, qStart, dealID, deadline); err != nil {
			return domain.Proposal{}, err
		}
	}

	if err := checkItemHolder(ctx, tx, itemID, participantID); err != nil {
		return domain.Proposal{}, err
	}

	toUserID, err := resolveRecipient(ctx, tx, itemID, participantID)
	if err != nil {
		return domain.Proposal{}, err
	}

	const qTx = `
		INSERT INTO transactions (item_id, from_user, to_user, quantity)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4)
		RETURNING id::text
	`
	var transactionID string
	if err := tx.QueryRow(ctx, qTx, itemID, participantID, toUserID, quantity).Scan(&transactionID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return domain.Proposal{}, ErrItemNotFound
		}
		return domain.Proposal{}, err
	}

	const qLink = `
		INSERT INTO chain_deal_transactions (deal_id, transaction_id, participant_id)
		VALUES ($1::uuid, $2::uuid, $3::uuid)
		RETURNING status, updated_at
	`
	var p domain.Proposal
	if err := tx.QueryRow(ctx, qLink, dealID, transactionID, participantID).Scan(&p.Status, &p.UpdatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.Proposal{}, ErrAlreadyProposed
		}
		return domain.Proposal{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Proposal{}, err
	}

	p.DealID = dealID
	p.TransactionID = transactionID
	p.ParticipantID = participantID
	p.ItemID = itemID
	p.ToUserID = toUserID
	p.Quantity = quantity
	return p, nil
}

func (r *Repository) GetByDealAndParticipant(ctx context.Context, dealID, participantID string) (domain.Proposal, error) {
	const q = `
		SELECT cdt.deal_id::text, cdt.transaction_id::text, cdt.participant_id::text,
		       t.item_id::text, t.to_user::text, t.quantity, cdt.status, cdt.updated_at
		FROM chain_deal_transactions cdt
		JOIN transactions t ON t.id = cdt.transaction_id
		WHERE cdt.deal_id = $1::uuid AND cdt.participant_id = $2::uuid
	`
	return r.scanProposal(r.pool.QueryRow(ctx, q, dealID, participantID))
}

func (r *Repository) Update(ctx context.Context, dealID, participantID string, upd domain.ProposalUpdate) (domain.Proposal, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Proposal{}, err
	}
	defer tx.Rollback(ctx)

	if err := lockPending(ctx, tx, dealID, participantID); err != nil {
		return domain.Proposal{}, err
	}

	sets := make([]string, 0, 3)
	args := []any{dealID, participantID}
	n := 3
	if upd.ItemID != nil {
		if err := checkItemHolder(ctx, tx, *upd.ItemID, participantID); err != nil {
			return domain.Proposal{}, err
		}
		toUserID, err := resolveRecipient(ctx, tx, *upd.ItemID, participantID)
		if err != nil {
			return domain.Proposal{}, err
		}
		sets = append(sets, fmt.Sprintf("item_id = $%d::uuid", n))
		args = append(args, *upd.ItemID)
		n++
		sets = append(sets, fmt.Sprintf("to_user = $%d::uuid", n))
		args = append(args, toUserID)
		n++
	}
	if upd.Quantity != nil {
		sets = append(sets, fmt.Sprintf("quantity = $%d", n))
		args = append(args, *upd.Quantity)
		n++
	}
	if len(sets) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return domain.Proposal{}, err
		}
		return r.GetByDealAndParticipant(ctx, dealID, participantID)
	}

	q := fmt.Sprintf(`
		UPDATE transactions t
		SET %s
		FROM chain_deal_transactions cdt
		WHERE cdt.transaction_id = t.id
		  AND cdt.deal_id = $1::uuid
		  AND cdt.participant_id = $2::uuid
		RETURNING cdt.deal_id::text, cdt.transaction_id::text, cdt.participant_id::text,
		          t.item_id::text, t.to_user::text, t.quantity, cdt.status, cdt.updated_at
	`, strings.Join(sets, ", "))

	p, err := r.scanProposal(tx.QueryRow(ctx, q, args...))
	if err != nil {
		return domain.Proposal{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Proposal{}, err
	}
	return p, nil
}

func (r *Repository) SetStatus(ctx context.Context, dealID, participantID string, status domain.ProposalStatus) (domain.Proposal, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Proposal{}, err
	}
	defer tx.Rollback(ctx)

	if err := lockPending(ctx, tx, dealID, participantID); err != nil {
		return domain.Proposal{}, err
	}

	if status == domain.ProposalStatusAccepted {
		allowed, err := isApprovalAllowed(ctx, tx, dealID, participantID)
		if err != nil {
			return domain.Proposal{}, err
		}
		if !allowed {
			return domain.Proposal{}, ErrOutOfOrder
		}
	}

	const q = `
		UPDATE chain_deal_transactions cdt
		SET status = $3
		FROM transactions t
		WHERE cdt.transaction_id = t.id
		  AND cdt.deal_id = $1::uuid
		  AND cdt.participant_id = $2::uuid
		RETURNING cdt.deal_id::text, cdt.transaction_id::text, cdt.participant_id::text,
		          t.item_id::text, t.to_user::text, t.quantity, cdt.status, cdt.updated_at
	`
	p, err := r.scanProposal(tx.QueryRow(ctx, q, dealID, participantID, status))
	if err != nil {
		return domain.Proposal{}, err
	}

	if status == domain.ProposalStatusAccepted {
		const qExtend = `
			UPDATE chain_deals
			SET deadline_at = now() + (negotiation_window_seconds || ' seconds')::interval
			WHERE id = $1::uuid
		`
		if _, err := tx.Exec(ctx, qExtend, dealID); err != nil {
			return domain.Proposal{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Proposal{}, err
	}
	return p, nil
}

func (r *Repository) ListForUser(ctx context.Context, userID string) ([]domain.Proposal, error) {
	const q = `
		SELECT cdt.deal_id::text, cdt.transaction_id::text, cdt.participant_id::text,
		       t.item_id::text, t.to_user::text, t.quantity, cdt.status, cdt.updated_at
		FROM chain_deal_transactions cdt
		JOIN transactions t ON t.id = cdt.transaction_id
		WHERE cdt.participant_id = $1::uuid
		ORDER BY cdt.updated_at DESC
	`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.Proposal, 0)
	for rows.Next() {
		var p domain.Proposal
		if err := rows.Scan(
			&p.DealID, &p.TransactionID, &p.ParticipantID,
			&p.ItemID, &p.ToUserID, &p.Quantity, &p.Status, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *Repository) AllAccepted(ctx context.Context, dealID string) (bool, error) {
	const q = `
		SELECT d.deadline_at IS NOT NULL
		   AND COUNT(cdt.transaction_id) FILTER (WHERE cdt.status <> 'ACCEPTED') = 0
		FROM chain_deals d
		LEFT JOIN chain_deal_transactions cdt ON cdt.deal_id = d.id
		WHERE d.id = $1::uuid
		GROUP BY d.id, d.deadline_at
	`
	var allAccepted bool
	if err := r.pool.QueryRow(ctx, q, dealID).Scan(&allAccepted); err != nil {
		return false, err
	}
	return allAccepted, nil
}

func (r *Repository) TryLockChain(ctx context.Context, dealID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	const qDeal = `
		SELECT creator_id::text, root_item_id::text
		FROM chain_deals
		WHERE id = $1::uuid
		FOR UPDATE
	`
	var creatorID, rootItemID string
	if err := tx.QueryRow(ctx, qDeal, dealID).Scan(&creatorID, &rootItemID); err != nil {
		return err
	}

	const qAccepted = `
		SELECT EXISTS (
			SELECT 1 FROM chain_deal_transactions
			WHERE deal_id = $1::uuid AND participant_id = $2::uuid AND status = 'ACCEPTED'
		)
	`
	var creatorAccepted bool
	if err := tx.QueryRow(ctx, qAccepted, dealID, creatorID).Scan(&creatorAccepted); err != nil {
		return err
	}
	if !creatorAccepted {
		return tx.Commit(ctx)
	}

	const qRootAccepted = `
		SELECT EXISTS (
			SELECT 1
			FROM chain_deal_transactions cdt
			JOIN transactions t ON t.id = cdt.transaction_id
			WHERE cdt.deal_id = $1::uuid AND t.item_id = $2::uuid AND cdt.status = 'ACCEPTED'
		)
	`
	var rootAccepted bool
	if err := tx.QueryRow(ctx, qRootAccepted, dealID, rootItemID).Scan(&rootAccepted); err != nil {
		return err
	}
	if !rootAccepted {
		return tx.Commit(ctx)
	}

	const qAlreadyLocked = `SELECT is_locked FROM items WHERE id = $1::uuid`
	var alreadyLocked bool
	if err := tx.QueryRow(ctx, qAlreadyLocked, rootItemID).Scan(&alreadyLocked); err != nil {
		return err
	}
	if alreadyLocked {
		return tx.Commit(ctx)
	}

	const qLockItems = `
		UPDATE items
		SET is_locked = true, locked_by_deal_id = $2::uuid
		WHERE id IN (
			SELECT t.item_id
			FROM chain_deal_transactions cdt
			JOIN transactions t ON t.id = cdt.transaction_id
			WHERE cdt.deal_id = $1::uuid
		)
	`
	if _, err := tx.Exec(ctx, qLockItems, dealID, dealID); err != nil {
		return err
	}

	const qCancelCompeting = `
		UPDATE chain_deals
		SET status = 'CANCELLED'
		WHERE status = 'PENDING'
		  AND id <> $1::uuid
		  AND (
		    root_item_id IN (
		        SELECT t.item_id
		        FROM chain_deal_transactions cdt
		        JOIN transactions t ON t.id = cdt.transaction_id
		        WHERE cdt.deal_id = $1::uuid
		    )
		    OR id IN (
		        SELECT cdt2.deal_id
		        FROM chain_deal_transactions cdt2
		        JOIN transactions t2 ON t2.id = cdt2.transaction_id
		        WHERE t2.item_id IN (
		            SELECT t.item_id
		            FROM chain_deal_transactions cdt
		            JOIN transactions t ON t.id = cdt.transaction_id
		            WHERE cdt.deal_id = $1::uuid
		        )
		    )
		  )
	`
	if _, err := tx.Exec(ctx, qCancelCompeting, dealID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) ListTransfers(ctx context.Context, dealID string) ([]domain.ItemTransfer, error) {
	const q = `
		SELECT t.item_id::text, t.to_user::text
		FROM chain_deal_transactions cdt
		JOIN transactions t ON t.id = cdt.transaction_id
		WHERE cdt.deal_id = $1::uuid
	`
	rows, err := r.pool.Query(ctx, q, dealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.ItemTransfer, 0)
	for rows.Next() {
		var it domain.ItemTransfer
		if err := rows.Scan(&it.ItemID, &it.ToUserID); err != nil {
			return nil, err
		}
		list = append(list, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *Repository) UnlockAllForDeal(ctx context.Context, dealID string) error {
	const q = `
		UPDATE items
		SET is_locked = false, locked_by_deal_id = NULL
		WHERE id IN (
			SELECT t.item_id
			FROM chain_deal_transactions cdt
			JOIN transactions t ON t.id = cdt.transaction_id
			WHERE cdt.deal_id = $1::uuid
		)
	`
	_, err := r.pool.Exec(ctx, q, dealID)
	return err
}

type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func lockPending(ctx context.Context, tx pgx.Tx, dealID, participantID string) error {
	const query = `
		SELECT status
		FROM chain_deal_transactions
		WHERE deal_id = $1::uuid AND participant_id = $2::uuid
		FOR UPDATE
	`
	var current domain.ProposalStatus
	err := tx.QueryRow(ctx, query, dealID, participantID).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if current != domain.ProposalStatusPending {
		return ErrNotPending
	}
	return nil
}

func isApprovalAllowed(ctx context.Context, tx pgx.Tx, dealID, participantID string) (bool, error) {
	const query = `
		SELECT
			cd.creator_id = $2::uuid
			OR t.item_id = cd.root_item_id
			OR (
				ri.is_locked
				AND EXISTS (
					SELECT 1 FROM chain_deal_transactions cdt2
					WHERE cdt2.deal_id = $1::uuid AND cdt2.participant_id = t.to_user AND cdt2.status = 'ACCEPTED'
				)
			)
		FROM chain_deals cd
		JOIN chain_deal_transactions cdt ON cdt.deal_id = cd.id AND cdt.participant_id = $2::uuid
		JOIN transactions t ON t.id = cdt.transaction_id
		JOIN items ri ON ri.id = cd.root_item_id
		WHERE cd.id = $1::uuid
	`
	var allowed bool
	err := tx.QueryRow(ctx, query, dealID, participantID).Scan(&allowed)
	if err != nil {
		return false, err
	}
	return allowed, nil
}

func checkItemHolder(ctx context.Context, tx pgx.Tx, itemID, participantID string) error {
	const query = `
		SELECT holder_id::text, is_locked, locked_by_deal_id IS NOT NULL
		FROM items
		WHERE id = $1::uuid
		FOR UPDATE
	`
	var holderID string
	var isLocked, lockedByDeal bool
	err := tx.QueryRow(ctx, query, itemID).Scan(&holderID, &isLocked, &lockedByDeal)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrItemNotFound
	}
	if err != nil {
		return err
	}
	if holderID != participantID {
		return ErrNotItemHolder
	}
	if isLocked {
		if lockedByDeal {
			return ErrItemLocked
		}
		const qRelease = `UPDATE items SET is_locked = false WHERE id = $1::uuid`
		if _, err := tx.Exec(ctx, qRelease, itemID); err != nil {
			return err
		}
	}
	return nil
}

func resolveRecipient(ctx context.Context, q rowQuerier, itemID, proposerID string) (string, error) {
	const query = `
		SELECT w.user_id::text
		FROM wish_items wi
		JOIN wishes w ON w.id = wi.wish_id
		WHERE wi.item_id = $1::uuid AND w.user_id <> $2::uuid
		ORDER BY w.created_at
		LIMIT 1
	`
	var userID string
	err := q.QueryRow(ctx, query, itemID, proposerID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRecipientNotFound
	}
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (r *Repository) scanProposal(row pgx.Row) (domain.Proposal, error) {
	var p domain.Proposal
	err := row.Scan(
		&p.DealID, &p.TransactionID, &p.ParticipantID,
		&p.ItemID, &p.ToUserID, &p.Quantity, &p.Status, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Proposal{}, ErrNotFound
	}
	if err != nil {
		return domain.Proposal{}, err
	}
	return p, nil
}

package proposal

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	sets := make([]string, 0, 3)
	args := []any{dealID, participantID}
	n := 3
	if upd.ItemID != nil {
		if err := checkItemHolder(ctx, r.pool, *upd.ItemID, participantID); err != nil {
			return domain.Proposal{}, err
		}
		toUserID, err := resolveRecipient(ctx, r.pool, *upd.ItemID, participantID)
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

	p, err := r.scanProposal(r.pool.QueryRow(ctx, q, args...))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return domain.Proposal{}, ErrItemNotFound
		}
		return domain.Proposal{}, err
	}
	return p, nil
}

func (r *Repository) SetStatus(ctx context.Context, dealID, participantID string, status domain.ProposalStatus) (domain.Proposal, error) {
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
	return r.scanProposal(r.pool.QueryRow(ctx, q, dealID, participantID, status))
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
		SELECT COUNT(cdt.transaction_id) = d.participants
		   AND COUNT(cdt.transaction_id) FILTER (WHERE cdt.status <> 'ACCEPTED') = 0
		FROM chain_deals d
		LEFT JOIN chain_deal_transactions cdt ON cdt.deal_id = d.id
		WHERE d.id = $1::uuid
		GROUP BY d.id, d.participants
	`
	var allAccepted bool
	if err := r.pool.QueryRow(ctx, q, dealID).Scan(&allAccepted); err != nil {
		return false, err
	}
	return allAccepted, nil
}

func (r *Repository) CountForDeal(ctx context.Context, dealID string) (int, error) {
	const q = `SELECT COUNT(*) FROM chain_deal_transactions WHERE deal_id = $1::uuid`
	var count int
	if err := r.pool.QueryRow(ctx, q, dealID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func checkItemHolder(ctx context.Context, q rowQuerier, itemID, participantID string) error {
	const query = `SELECT holder_id::text FROM items WHERE id = $1::uuid`
	var holderID string
	err := q.QueryRow(ctx, query, itemID).Scan(&holderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrItemNotFound
	}
	if err != nil {
		return err
	}
	if holderID != participantID {
		return ErrNotItemHolder
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

package proposal

import (
	"context"
	"errors"
	"testing"
	"time"

	"avitohvk/internal/domain"
	"avitohvk/internal/repository/dbtest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func seedUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (username, password_hash) VALUES ('user_' || gen_random_uuid()::text, 'hash') RETURNING id::text`,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id
}

func seedItem(t *testing.T, pool *pgxpool.Pool, authorID, holderID string, locked bool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO items (author_id, holder_id, title, is_locked) VALUES ($1::uuid, $2::uuid, 'item', $3) RETURNING id::text`,
		authorID, holderID, locked,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed item: %v", err)
	}
	return id
}

func lockItemToDeal(t *testing.T, pool *pgxpool.Pool, itemID, dealID string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`UPDATE items SET is_locked = true, locked_by_deal_id = $1::uuid WHERE id = $2::uuid`, dealID, itemID)
	if err != nil {
		t.Fatalf("lock item to deal: %v", err)
	}
}

func seedDeal(t *testing.T, pool *pgxpool.Pool, rootItemID, creatorID string, windowSeconds int, deadlineAt *time.Time) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO chain_deals (root_item_id, creator_id, negotiation_window_seconds, deadline_at)
		 VALUES ($1::uuid, $2::uuid, $3, $4) RETURNING id::text`,
		rootItemID, creatorID, windowSeconds, deadlineAt,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed deal: %v", err)
	}
	return id
}

func seedWish(t *testing.T, pool *pgxpool.Pool, userID, itemID string, createdAt time.Time) string {
	t.Helper()
	var wishID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO wishes (user_id, title, created_at) VALUES ($1::uuid, 'test wish', $2) RETURNING id::text`,
		userID, createdAt,
	).Scan(&wishID)
	if err != nil {
		t.Fatalf("seed wish: %v", err)
	}
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO wish_items (wish_id, item_id) VALUES ($1::uuid, $2::uuid)`, wishID, itemID,
	); err != nil {
		t.Fatalf("seed wish_item: %v", err)
	}
	return wishID
}

func seedProposal(t *testing.T, pool *pgxpool.Pool, dealID, participantID, itemID, toUser string, quantity float64, status domain.ProposalStatus) string {
	t.Helper()
	ctx := context.Background()
	var txID string
	err := pool.QueryRow(ctx,
		`INSERT INTO transactions (item_id, from_user, to_user, quantity) VALUES ($1::uuid, $2::uuid, $3::uuid, $4) RETURNING id::text`,
		itemID, participantID, toUser, quantity,
	).Scan(&txID)
	if err != nil {
		t.Fatalf("seed transaction: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO chain_deal_transactions (deal_id, transaction_id, participant_id, status) VALUES ($1::uuid, $2::uuid, $3::uuid, $4)`,
		dealID, txID, participantID, status,
	); err != nil {
		t.Fatalf("seed chain_deal_transactions: %v", err)
	}
	return txID
}

type itemState struct {
	holderID       string
	isLocked       bool
	lockedByDealID *string
}

func getItemState(t *testing.T, pool *pgxpool.Pool, itemID string) itemState {
	t.Helper()
	var s itemState
	err := pool.QueryRow(context.Background(),
		`SELECT holder_id::text, is_locked, locked_by_deal_id::text FROM items WHERE id = $1::uuid`, itemID,
	).Scan(&s.holderID, &s.isLocked, &s.lockedByDealID)
	if err != nil {
		t.Fatalf("get item state: %v", err)
	}
	return s
}

func getDealState(t *testing.T, pool *pgxpool.Pool, dealID string) (status domain.DealStatus, deadlineAt *time.Time) {
	t.Helper()
	err := pool.QueryRow(context.Background(),
		`SELECT status, deadline_at FROM chain_deals WHERE id = $1::uuid`, dealID,
	).Scan(&status, &deadlineAt)
	if err != nil {
		t.Fatalf("get deal state: %v", err)
	}
	return status, deadlineAt
}

func getProposalStatus(t *testing.T, pool *pgxpool.Pool, dealID, participantID string) domain.ProposalStatus {
	t.Helper()
	var status domain.ProposalStatus
	err := pool.QueryRow(context.Background(),
		`SELECT status FROM chain_deal_transactions WHERE deal_id = $1::uuid AND participant_id = $2::uuid`, dealID, participantID,
	).Scan(&status)
	if err != nil {
		t.Fatalf("get proposal status: %v", err)
	}
	return status
}

func TestRepository_Create_Success_NonRootItemLeavesChainOpen(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	participant := seedUser(t, pool)
	recipient := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	offeredItem := seedItem(t, pool, participant, participant, false)
	seedWish(t, pool, recipient, offeredItem, time.Now())

	p, err := repo.Create(ctx, dealID, participant, offeredItem, 2)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if p.DealID != dealID || p.ParticipantID != participant || p.ItemID != offeredItem || p.ToUserID != recipient || p.Quantity != 2 {
		t.Errorf("proposal = %+v", p)
	}
	if p.TransactionID == "" {
		t.Errorf("TransactionID is empty")
	}
	if p.Status != domain.ProposalStatusPending {
		t.Errorf("Status = %q, want PENDING", p.Status)
	}

	if _, deadline := getDealState(t, pool, dealID); deadline != nil {
		t.Errorf("deadline_at = %v, want nil (root item was never proposed)", *deadline)
	}
}

func TestRepository_Create_Success_RootItemStartsDeadline(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)
	seedWish(t, pool, creator, rootItem, time.Now())

	before := time.Now()
	p, err := repo.Create(ctx, dealID, rootHolder, rootItem, 1)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ToUserID != creator {
		t.Errorf("ToUserID = %q, want creator %q", p.ToUserID, creator)
	}

	_, deadline := getDealState(t, pool, dealID)
	if deadline == nil {
		t.Fatalf("deadline_at is nil, want set to now()+window")
	}
	if deadline.Before(before.Add(3599 * time.Second)) {
		t.Errorf("deadline_at = %v, want roughly %v later", *deadline, 3600*time.Second)
	}
}

func TestRepository_Create_ChainClosedAfterRootItemProposed(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)
	seedWish(t, pool, creator, rootItem, time.Now())

	if _, err := repo.Create(ctx, dealID, rootHolder, rootItem, 1); err != nil {
		t.Fatalf("seed root proposal: %v", err)
	}

	newParticipant := seedUser(t, pool)
	recipient := seedUser(t, pool)
	otherItem := seedItem(t, pool, newParticipant, newParticipant, false)
	seedWish(t, pool, recipient, otherItem, time.Now())

	_, err := repo.Create(ctx, dealID, newParticipant, otherItem, 1)
	if !errors.Is(err, ErrChainClosed) {
		t.Errorf("err = %v, want ErrChainClosed", err)
	}
}

func TestRepository_Create_DealNotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	participant := seedUser(t, pool)
	item := seedItem(t, pool, participant, participant, false)

	_, err := repo.Create(ctx, "00000000-0000-0000-0000-000000000000", participant, item, 1)
	if err == nil {
		t.Fatalf("Create succeeded, want an error")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("err = %v, want pgx.ErrNoRows surfaced unmapped", err)
	}
}

func TestRepository_Create_ItemNotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	_, err := repo.Create(ctx, dealID, participant, "00000000-0000-0000-0000-000000000000", 1)
	if !errors.Is(err, ErrItemNotFound) {
		t.Errorf("err = %v, want ErrItemNotFound", err)
	}
}

func TestRepository_Create_NotItemHolder(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	actualHolder := seedUser(t, pool)
	item := seedItem(t, pool, actualHolder, actualHolder, false)

	_, err := repo.Create(ctx, dealID, participant, item, 1)
	if !errors.Is(err, ErrNotItemHolder) {
		t.Errorf("err = %v, want ErrNotItemHolder", err)
	}
}

func TestRepository_Create_ItemLockedByAnotherDeal(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	item := seedItem(t, pool, participant, participant, true)
	otherDealID := seedDeal(t, pool, item, creator, 3600, nil)
	lockItemToDeal(t, pool, item, otherDealID)

	_, err := repo.Create(ctx, dealID, participant, item, 1)
	if !errors.Is(err, ErrItemLocked) {
		t.Errorf("err = %v, want ErrItemLocked", err)
	}
}

func TestRepository_Create_BareChownLockIsAutoReleased(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	recipient := seedUser(t, pool)
	item := seedItem(t, pool, participant, participant, true)
	seedWish(t, pool, recipient, item, time.Now())

	if _, err := repo.Create(ctx, dealID, participant, item, 1); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if s := getItemState(t, pool, item); s.isLocked {
		t.Errorf("item is_locked = true after Create, want the bare chown lock to be released")
	}
}

func TestRepository_Create_RecipientNotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	item := seedItem(t, pool, participant, participant, false)

	_, err := repo.Create(ctx, dealID, participant, item, 1)
	if !errors.Is(err, ErrRecipientNotFound) {
		t.Errorf("err = %v, want ErrRecipientNotFound", err)
	}
}

func TestRepository_Create_AlreadyProposed(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	recipient1 := seedUser(t, pool)
	item1 := seedItem(t, pool, participant, participant, false)
	seedWish(t, pool, recipient1, item1, time.Now())
	if _, err := repo.Create(ctx, dealID, participant, item1, 1); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	recipient2 := seedUser(t, pool)
	item2 := seedItem(t, pool, participant, participant, false)
	seedWish(t, pool, recipient2, item2, time.Now())

	_, err := repo.Create(ctx, dealID, participant, item2, 1)
	if !errors.Is(err, ErrAlreadyProposed) {
		t.Errorf("err = %v, want ErrAlreadyProposed", err)
	}
}

func TestRepository_Create_EarliestWishWins(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	earlyWisher := seedUser(t, pool)
	lateWisher := seedUser(t, pool)
	item := seedItem(t, pool, participant, participant, false)

	base := time.Now().Add(-time.Hour)
	seedWish(t, pool, lateWisher, item, base.Add(time.Minute))
	seedWish(t, pool, earlyWisher, item, base)

	p, err := repo.Create(ctx, dealID, participant, item, 1)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ToUserID != earlyWisher {
		t.Errorf("ToUserID = %q, want the earlier wisher %q", p.ToUserID, earlyWisher)
	}
}

func TestRepository_GetByDealAndParticipant_Success(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	participant := seedUser(t, pool)
	toUser := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)
	item := seedItem(t, pool, participant, participant, false)
	seedProposal(t, pool, dealID, participant, item, toUser, 2, domain.ProposalStatusPending)

	got, err := repo.GetByDealAndParticipant(ctx, dealID, participant)
	if err != nil {
		t.Fatalf("GetByDealAndParticipant: %v", err)
	}
	if got.DealID != dealID || got.ParticipantID != participant || got.ItemID != item || got.ToUserID != toUser || got.Quantity != 2 {
		t.Errorf("proposal = %+v", got)
	}
}

func TestRepository_GetByDealAndParticipant_NotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)

	_, err := repo.GetByDealAndParticipant(context.Background(), "00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestRepository_Update_Success_ItemAndQuantity(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	oldItem := seedItem(t, pool, participant, participant, false)
	oldRecipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, participant, oldItem, oldRecipient, 1, domain.ProposalStatusPending)

	newItem := seedItem(t, pool, participant, participant, false)
	newRecipient := seedUser(t, pool)
	seedWish(t, pool, newRecipient, newItem, time.Now())

	got, err := repo.Update(ctx, dealID, participant, domain.ProposalUpdate{ItemID: &newItem, Quantity: floatPtr(5)})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.ItemID != newItem || got.ToUserID != newRecipient || got.Quantity != 5 {
		t.Errorf("proposal = %+v", got)
	}
}

func TestRepository_Update_Success_QuantityOnlyBumpsUpdatedAt(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	item := seedItem(t, pool, participant, participant, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, participant, item, recipient, 1, domain.ProposalStatusPending)

	before, err := repo.GetByDealAndParticipant(ctx, dealID, participant)
	if err != nil {
		t.Fatalf("seed read: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	got, err := repo.Update(ctx, dealID, participant, domain.ProposalUpdate{Quantity: floatPtr(9)})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Quantity != 9 {
		t.Errorf("Quantity = %v, want 9", got.Quantity)
	}
	if got.ItemID != item {
		t.Errorf("ItemID = %q, want unchanged %q", got.ItemID, item)
	}
	if !got.UpdatedAt.After(before.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want strictly after %v", got.UpdatedAt, before.UpdatedAt)
	}
}

func TestRepository_Update_EmptyUpdateIsANoOpAndDoesNotBumpUpdatedAt(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	item := seedItem(t, pool, participant, participant, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, participant, item, recipient, 1, domain.ProposalStatusPending)

	before, err := repo.GetByDealAndParticipant(ctx, dealID, participant)
	if err != nil {
		t.Fatalf("seed read: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	got, err := repo.Update(ctx, dealID, participant, domain.ProposalUpdate{})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !got.UpdatedAt.Equal(before.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want unchanged %v (empty update must not touch the row)", got.UpdatedAt, before.UpdatedAt)
	}
}

func TestRepository_Update_NotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)

	_, err := repo.Update(context.Background(), "00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000000", domain.ProposalUpdate{Quantity: floatPtr(1)})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestRepository_Update_NotPending(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	item := seedItem(t, pool, participant, participant, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, participant, item, recipient, 1, domain.ProposalStatusAccepted)

	_, err := repo.Update(ctx, dealID, participant, domain.ProposalUpdate{Quantity: floatPtr(2)})
	if !errors.Is(err, ErrNotPending) {
		t.Errorf("err = %v, want ErrNotPending", err)
	}
}

func TestRepository_Update_NewItemRecipientNotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	oldItem := seedItem(t, pool, participant, participant, false)
	oldRecipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, participant, oldItem, oldRecipient, 1, domain.ProposalStatusPending)

	newItemNoWisher := seedItem(t, pool, participant, participant, false)

	_, err := repo.Update(ctx, dealID, participant, domain.ProposalUpdate{ItemID: &newItemNoWisher})
	if !errors.Is(err, ErrRecipientNotFound) {
		t.Errorf("err = %v, want ErrRecipientNotFound", err)
	}
}

func floatPtr(f float64) *float64 { return &f }

func TestRepository_SetStatus_Success_Declined(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	item := seedItem(t, pool, participant, participant, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, participant, item, recipient, 1, domain.ProposalStatusPending)

	got, err := repo.SetStatus(ctx, dealID, participant, domain.ProposalStatusDeclined)
	if err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	if got.Status != domain.ProposalStatusDeclined {
		t.Errorf("Status = %q, want DECLINED", got.Status)
	}
}

func TestRepository_SetStatus_NotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)

	_, err := repo.SetStatus(context.Background(), "00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000000", domain.ProposalStatusDeclined)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestRepository_SetStatus_NotPending(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	item := seedItem(t, pool, participant, participant, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, participant, item, recipient, 1, domain.ProposalStatusDeclined)

	_, err := repo.SetStatus(ctx, dealID, participant, domain.ProposalStatusAccepted)
	if !errors.Is(err, ErrNotPending) {
		t.Errorf("err = %v, want ErrNotPending", err)
	}
}

func TestRepository_SetStatus_Accepted_NoLongerHoldsItem(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	item := seedItem(t, pool, participant, participant, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, participant, item, recipient, 1, domain.ProposalStatusPending)

	if _, err := pool.Exec(ctx, `UPDATE items SET holder_id = $1::uuid WHERE id = $2::uuid`, recipient, item); err != nil {
		t.Fatalf("seed holder change: %v", err)
	}

	_, err := repo.SetStatus(ctx, dealID, participant, domain.ProposalStatusAccepted)
	if !errors.Is(err, ErrNotItemHolder) {
		t.Errorf("err = %v, want ErrNotItemHolder", err)
	}
}

func TestRepository_SetStatus_Accepted_OutOfOrder(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	item := seedItem(t, pool, participant, participant, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, participant, item, recipient, 1, domain.ProposalStatusPending)
	seedProposal(t, pool, dealID, recipient, seedItem(t, pool, recipient, recipient, false), rootHolder, 1, domain.ProposalStatusPending)

	_, err := repo.SetStatus(ctx, dealID, participant, domain.ProposalStatusAccepted)
	if !errors.Is(err, ErrOutOfOrder) {
		t.Errorf("err = %v, want ErrOutOfOrder", err)
	}
}

func TestRepository_SetStatus_Accepted_AllowedAsCreatorEvenBeforeChainLocks(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	item := seedItem(t, pool, creator, creator, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, creator, item, recipient, 1, domain.ProposalStatusPending)

	got, err := repo.SetStatus(ctx, dealID, creator, domain.ProposalStatusAccepted)
	if err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	if got.Status != domain.ProposalStatusAccepted {
		t.Errorf("Status = %q, want ACCEPTED", got.Status)
	}
	if _, deadline := getDealState(t, pool, dealID); deadline != nil {
		t.Errorf("deadline_at = %v, want still nil (qExtend must no-op when deadline was never started)", *deadline)
	}
}

func TestRepository_SetStatus_Accepted_AllowedAsRootItemHolder(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)
	seedProposal(t, pool, dealID, rootHolder, rootItem, creator, 1, domain.ProposalStatusPending)

	got, err := repo.SetStatus(ctx, dealID, rootHolder, domain.ProposalStatusAccepted)
	if err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	if got.Status != domain.ProposalStatusAccepted {
		t.Errorf("Status = %q, want ACCEPTED", got.Status)
	}
}

func TestRepository_SetStatus_Accepted_LocksOfferedItemToDeal(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)
	seedProposal(t, pool, dealID, rootHolder, rootItem, creator, 1, domain.ProposalStatusPending)

	if _, err := repo.SetStatus(ctx, dealID, rootHolder, domain.ProposalStatusAccepted); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	s := getItemState(t, pool, rootItem)
	if !s.isLocked {
		t.Errorf("is_locked = false, want true")
	}
	if s.lockedByDealID == nil || *s.lockedByDealID != dealID {
		t.Errorf("lockedByDealID = %v, want %q", s.lockedByDealID, dealID)
	}
}

func TestRepository_SetStatus_Accepted_AllowedAfterChainLockedAndRecipientAccepted(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)
	lockItemToDeal(t, pool, rootItem, dealID)

	participant := seedUser(t, pool)
	item := seedItem(t, pool, participant, participant, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, participant, item, recipient, 1, domain.ProposalStatusPending)
	seedProposal(t, pool, dealID, recipient, seedItem(t, pool, recipient, recipient, false), rootHolder, 1, domain.ProposalStatusAccepted)

	got, err := repo.SetStatus(ctx, dealID, participant, domain.ProposalStatusAccepted)
	if err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	if got.Status != domain.ProposalStatusAccepted {
		t.Errorf("Status = %q, want ACCEPTED", got.Status)
	}
}

func TestRepository_SetStatus_Accepted_ExtendsDeadline(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	almostExpired := time.Now().Add(2 * time.Second)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, &almostExpired)

	item := seedItem(t, pool, creator, creator, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, creator, item, recipient, 1, domain.ProposalStatusPending)

	if _, err := repo.SetStatus(ctx, dealID, creator, domain.ProposalStatusAccepted); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	_, deadline := getDealState(t, pool, dealID)
	if deadline == nil || !deadline.After(almostExpired) {
		t.Errorf("deadline_at = %v, want extended past the original near-expiry %v", deadline, almostExpired)
	}
}

func TestRepository_SetStatus_Accepted_CancelsCompetingDeal(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creatorA := seedUser(t, pool)
	rootHolderA := seedUser(t, pool)
	rootItemA := seedItem(t, pool, rootHolderA, rootHolderA, false)
	dealA := seedDeal(t, pool, rootItemA, creatorA, 3600, nil)

	recipient := seedUser(t, pool)
	sharedItem := seedItem(t, pool, creatorA, creatorA, false)
	seedProposal(t, pool, dealA, creatorA, sharedItem, recipient, 1, domain.ProposalStatusPending)

	creatorB := seedUser(t, pool)
	dealBRoot := seedItem(t, pool, creatorB, creatorB, false)
	dealB := seedDeal(t, pool, dealBRoot, creatorB, 3600, nil)
	otherParticipant := seedUser(t, pool)
	otherRecipient := seedUser(t, pool)
	seedProposal(t, pool, dealB, otherParticipant, sharedItem, otherRecipient, 1, domain.ProposalStatusPending)

	dealBLockedItem := seedItem(t, pool, creatorB, creatorB, false)
	lockItemToDeal(t, pool, dealBLockedItem, dealB)

	if _, err := repo.SetStatus(ctx, dealA, creatorA, domain.ProposalStatusAccepted); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	status, _ := getDealState(t, pool, dealB)
	if status != domain.DealStatusCanceled {
		t.Errorf("deal B status = %q, want %q", status, domain.DealStatusCanceled)
	}
	if got := getProposalStatus(t, pool, dealB, otherParticipant); got != domain.ProposalStatusDeclined {
		t.Errorf("deal B proposal status = %q, want DECLINED", got)
	}
	if s := getItemState(t, pool, dealBLockedItem); s.isLocked {
		t.Errorf("deal B's separately locked item is_locked = true, want released")
	}
}

func TestRepository_SetStatus_Accepted_DoesNotCancelCompetingDealMatchedOnlyByRootItemID(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creatorA := seedUser(t, pool)
	rootHolderA := seedUser(t, pool)
	rootItemA := seedItem(t, pool, rootHolderA, rootHolderA, false)
	dealA := seedDeal(t, pool, rootItemA, creatorA, 3600, nil)

	recipient := seedUser(t, pool)
	sharedItem := seedItem(t, pool, creatorA, creatorA, false)
	seedProposal(t, pool, dealA, creatorA, sharedItem, recipient, 1, domain.ProposalStatusPending)

	creatorB := seedUser(t, pool)
	dealB := seedDeal(t, pool, sharedItem, creatorB, 3600, nil)

	if _, err := repo.SetStatus(ctx, dealA, creatorA, domain.ProposalStatusAccepted); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	status, _ := getDealState(t, pool, dealB)
	if status != domain.DealStatusPending {
		t.Errorf("deal B status = %q, want still PENDING (documents the current root_item_id-only gap)", status)
	}
}

func TestRepository_ListForUser_Empty(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)

	list, err := repo.ListForUser(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if list == nil {
		t.Errorf("list is nil, want an empty non-nil slice")
	}
	if len(list) != 0 {
		t.Errorf("list = %+v, want empty", list)
	}
}

func TestRepository_ListForUser_OrderedByUpdatedAtDesc(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	recipient := seedUser(t, pool)

	dealID2 := seedDeal(t, pool, seedItem(t, pool, creator, creator, false), creator, 3600, nil)
	item1 := seedItem(t, pool, participant, participant, false)
	item2 := seedItem(t, pool, participant, participant, false)
	seedProposal(t, pool, dealID, participant, item1, recipient, 1, domain.ProposalStatusPending)
	seedProposal(t, pool, dealID2, participant, item2, recipient, 1, domain.ProposalStatusPending)

	time.Sleep(10 * time.Millisecond)
	if _, err := repo.SetStatus(ctx, dealID2, participant, domain.ProposalStatusDeclined); err != nil {
		t.Fatalf("bump: %v", err)
	}

	list, err := repo.ListForUser(ctx, participant)
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list = %+v, want 2 entries", list)
	}
	if list[0].DealID != dealID2 {
		t.Errorf("list[0].DealID = %q, want the most recently updated deal %q", list[0].DealID, dealID2)
	}
}

func TestRepository_AllAccepted_True(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	deadline := time.Now().Add(time.Hour)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, &deadline)

	participant := seedUser(t, pool)
	item := seedItem(t, pool, participant, participant, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, participant, item, recipient, 1, domain.ProposalStatusAccepted)

	all, err := repo.AllAccepted(ctx, dealID)
	if err != nil {
		t.Fatalf("AllAccepted: %v", err)
	}
	if !all {
		t.Errorf("AllAccepted = false, want true")
	}
}

func TestRepository_AllAccepted_FalseWhenSomePending(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	deadline := time.Now().Add(time.Hour)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, &deadline)

	participant := seedUser(t, pool)
	item := seedItem(t, pool, participant, participant, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, participant, item, recipient, 1, domain.ProposalStatusPending)

	all, err := repo.AllAccepted(ctx, dealID)
	if err != nil {
		t.Fatalf("AllAccepted: %v", err)
	}
	if all {
		t.Errorf("AllAccepted = true, want false")
	}
}

func TestRepository_AllAccepted_FalseWhenDeadlineStillNil(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant := seedUser(t, pool)
	item := seedItem(t, pool, participant, participant, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, participant, item, recipient, 1, domain.ProposalStatusAccepted)

	all, err := repo.AllAccepted(ctx, dealID)
	if err != nil {
		t.Fatalf("AllAccepted: %v", err)
	}
	if all {
		t.Errorf("AllAccepted = true, want false (chain never locked)")
	}
}

func TestRepository_AllAccepted_DealNotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)

	_, err := repo.AllAccepted(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("err = %v, want pgx.ErrNoRows surfaced unmapped", err)
	}
}

func TestRepository_TryLockChain_DealNotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)

	err := repo.TryLockChain(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("err = %v, want pgx.ErrNoRows surfaced unmapped", err)
	}
}

func TestRepository_TryLockChain_NoOpWhenCreatorHasNotAccepted(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)
	seedProposal(t, pool, dealID, rootHolder, rootItem, creator, 1, domain.ProposalStatusAccepted)

	if err := repo.TryLockChain(ctx, dealID); err != nil {
		t.Fatalf("TryLockChain: %v", err)
	}
	if s := getItemState(t, pool, rootItem); s.isLocked {
		t.Errorf("root item is_locked = true, want false (creator has not accepted)")
	}
}

func TestRepository_TryLockChain_NoOpWhenRootProposalNotAccepted(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	item := seedItem(t, pool, creator, creator, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, creator, item, recipient, 1, domain.ProposalStatusAccepted)
	seedProposal(t, pool, dealID, rootHolder, rootItem, creator, 1, domain.ProposalStatusPending)

	if err := repo.TryLockChain(ctx, dealID); err != nil {
		t.Fatalf("TryLockChain: %v", err)
	}
	if s := getItemState(t, pool, rootItem); s.isLocked {
		t.Errorf("root item is_locked = true, want false (root proposal not yet accepted)")
	}
}

func TestRepository_TryLockChain_LocksAllItemsWhenBothAccepted(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	item := seedItem(t, pool, creator, creator, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, creator, item, recipient, 1, domain.ProposalStatusAccepted)
	seedProposal(t, pool, dealID, rootHolder, rootItem, creator, 1, domain.ProposalStatusAccepted)

	if err := repo.TryLockChain(ctx, dealID); err != nil {
		t.Fatalf("TryLockChain: %v", err)
	}

	for _, it := range []string{rootItem, item} {
		s := getItemState(t, pool, it)
		if !s.isLocked || s.lockedByDealID == nil || *s.lockedByDealID != dealID {
			t.Errorf("item %s state = %+v, want locked to deal %q", it, s, dealID)
		}
	}
}

func TestRepository_TryLockChain_IdempotentOnSecondCall(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)
	seedProposal(t, pool, dealID, rootHolder, rootItem, creator, 1, domain.ProposalStatusAccepted)

	item := seedItem(t, pool, creator, creator, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, creator, item, recipient, 1, domain.ProposalStatusAccepted)

	if err := repo.TryLockChain(ctx, dealID); err != nil {
		t.Fatalf("first TryLockChain: %v", err)
	}
	if err := repo.TryLockChain(ctx, dealID); err != nil {
		t.Fatalf("second TryLockChain: %v", err)
	}
}

func TestRepository_TryLockChain_CancelsCompetingDeal(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootHolder := seedUser(t, pool)
	rootItem := seedItem(t, pool, rootHolder, rootHolder, false)
	dealA := seedDeal(t, pool, rootItem, creator, 3600, nil)

	item := seedItem(t, pool, creator, creator, false)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealA, creator, item, recipient, 1, domain.ProposalStatusAccepted)
	seedProposal(t, pool, dealA, rootHolder, rootItem, creator, 1, domain.ProposalStatusAccepted)

	creatorB := seedUser(t, pool)
	dealB := seedDeal(t, pool, item, creatorB, 3600, nil)
	otherParticipant := seedUser(t, pool)
	otherRecipient := seedUser(t, pool)
	otherItem := seedItem(t, pool, otherParticipant, otherParticipant, false)
	seedProposal(t, pool, dealB, otherParticipant, otherItem, otherRecipient, 1, domain.ProposalStatusPending)
	lockItemToDeal(t, pool, otherItem, dealB)

	if err := repo.TryLockChain(ctx, dealA); err != nil {
		t.Fatalf("TryLockChain: %v", err)
	}

	status, _ := getDealState(t, pool, dealB)
	if status != domain.DealStatusCanceled {
		t.Errorf("deal B status = %q, want %q", status, domain.DealStatusCanceled)
	}
	if got := getProposalStatus(t, pool, dealB, otherParticipant); got != domain.ProposalStatusDeclined {
		t.Errorf("deal B proposal status = %q, want DECLINED", got)
	}
	if s := getItemState(t, pool, otherItem); s.isLocked {
		t.Errorf("deal B's held item is_locked = true, want released")
	}
}

func TestRepository_ListTransfers_Empty(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)

	list, err := repo.ListTransfers(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("ListTransfers: %v", err)
	}
	if list == nil {
		t.Errorf("list is nil, want an empty non-nil slice")
	}
}

func TestRepository_ListTransfers_Success(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	participant1 := seedUser(t, pool)
	participant2 := seedUser(t, pool)
	recipient1 := seedUser(t, pool)
	recipient2 := seedUser(t, pool)
	item1 := seedItem(t, pool, participant1, participant1, false)
	item2 := seedItem(t, pool, participant2, participant2, false)
	seedProposal(t, pool, dealID, participant1, item1, recipient1, 1, domain.ProposalStatusPending)
	seedProposal(t, pool, dealID, participant2, item2, recipient2, 1, domain.ProposalStatusPending)

	list, err := repo.ListTransfers(ctx, dealID)
	if err != nil {
		t.Fatalf("ListTransfers: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list = %+v, want 2 entries", list)
	}
	want := map[string]string{item1: recipient1, item2: recipient2}
	for _, transfer := range list {
		if want[transfer.ItemID] != transfer.ToUserID {
			t.Errorf("transfer %+v does not match expected recipient %q", transfer, want[transfer.ItemID])
		}
	}
}

func TestRepository_DeclineAllExcept_LeavesActorUntouched(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	actor := seedUser(t, pool)
	other1 := seedUser(t, pool)
	other2 := seedUser(t, pool)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, actor, seedItem(t, pool, actor, actor, false), recipient, 1, domain.ProposalStatusAccepted)
	seedProposal(t, pool, dealID, other1, seedItem(t, pool, other1, other1, false), recipient, 1, domain.ProposalStatusPending)
	seedProposal(t, pool, dealID, other2, seedItem(t, pool, other2, other2, false), recipient, 1, domain.ProposalStatusAccepted)

	if err := repo.DeclineAllExcept(ctx, dealID, actor); err != nil {
		t.Fatalf("DeclineAllExcept: %v", err)
	}

	if got := getProposalStatus(t, pool, dealID, actor); got != domain.ProposalStatusAccepted {
		t.Errorf("actor's own status = %q, want unchanged ACCEPTED", got)
	}
	if got := getProposalStatus(t, pool, dealID, other1); got != domain.ProposalStatusDeclined {
		t.Errorf("other1 status = %q, want DECLINED", got)
	}
	if got := getProposalStatus(t, pool, dealID, other2); got != domain.ProposalStatusDeclined {
		t.Errorf("other2 status = %q, want DECLINED", got)
	}
}

func TestRepository_DeclineAllForDeal_DeclinesEveryone(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	p1 := seedUser(t, pool)
	p2 := seedUser(t, pool)
	recipient := seedUser(t, pool)
	seedProposal(t, pool, dealID, p1, seedItem(t, pool, p1, p1, false), recipient, 1, domain.ProposalStatusAccepted)
	seedProposal(t, pool, dealID, p2, seedItem(t, pool, p2, p2, false), recipient, 1, domain.ProposalStatusPending)

	if err := repo.DeclineAllForDeal(ctx, dealID); err != nil {
		t.Fatalf("DeclineAllForDeal: %v", err)
	}

	if got := getProposalStatus(t, pool, dealID, p1); got != domain.ProposalStatusDeclined {
		t.Errorf("p1 status = %q, want DECLINED", got)
	}
	if got := getProposalStatus(t, pool, dealID, p2); got != domain.ProposalStatusDeclined {
		t.Errorf("p2 status = %q, want DECLINED", got)
	}
}

func TestRepository_UnlockAllForDeal_OnlyReleasesItemsLockedByThisDeal(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	holder := seedUser(t, pool)
	ownItem := seedItem(t, pool, holder, holder, true)
	otherDealItem := seedItem(t, pool, holder, holder, true)
	bareChownItem := seedItem(t, pool, holder, holder, true)

	dealID := seedDeal(t, pool, ownItem, creator, 3600, nil)
	otherDeal := seedDeal(t, pool, otherDealItem, creator, 3600, nil)

	lockItemToDeal(t, pool, ownItem, dealID)
	lockItemToDeal(t, pool, otherDealItem, otherDeal)

	if err := repo.UnlockAllForDeal(ctx, dealID); err != nil {
		t.Fatalf("UnlockAllForDeal: %v", err)
	}

	if s := getItemState(t, pool, ownItem); s.isLocked || s.lockedByDealID != nil {
		t.Errorf("ownItem state = %+v, want fully unlocked", s)
	}
	if s := getItemState(t, pool, otherDealItem); !s.isLocked || s.lockedByDealID == nil || *s.lockedByDealID != otherDeal {
		t.Errorf("otherDealItem state = %+v, want still locked to %q (must not touch other deals' locks)", s, otherDeal)
	}
	if s := getItemState(t, pool, bareChownItem); !s.isLocked {
		t.Errorf("bareChownItem is_locked = false, want unchanged true (not locked by any deal, out of scope)")
	}
}

func TestRepository_FindOpenDealAsRecipient_Found(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	offerer := seedUser(t, pool)
	recipient := seedUser(t, pool)
	offeredItem := seedItem(t, pool, offerer, offerer, false)
	seedProposal(t, pool, dealID, offerer, offeredItem, recipient, 1, domain.ProposalStatusPending)

	gotDealID, found, err := repo.FindOpenDealAsRecipient(ctx, offeredItem, recipient)
	if err != nil {
		t.Fatalf("FindOpenDealAsRecipient: %v", err)
	}
	if !found {
		t.Fatalf("found = false, want true")
	}
	if gotDealID != dealID {
		t.Errorf("dealID = %q, want %q", gotDealID, dealID)
	}
}

func TestRepository_FindOpenDealAsRecipient_NotFound_WrongParticipant(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	offerer := seedUser(t, pool)
	recipient := seedUser(t, pool)
	someoneElse := seedUser(t, pool)
	offeredItem := seedItem(t, pool, offerer, offerer, false)
	seedProposal(t, pool, dealID, offerer, offeredItem, recipient, 1, domain.ProposalStatusPending)

	_, found, err := repo.FindOpenDealAsRecipient(ctx, offeredItem, someoneElse)
	if err != nil {
		t.Fatalf("FindOpenDealAsRecipient: %v", err)
	}
	if found {
		t.Errorf("found = true, want false (someoneElse is not the designated recipient)")
	}
}

func TestRepository_FindOpenDealAsRecipient_NotFound_WrongItem(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, nil)

	offerer := seedUser(t, pool)
	recipient := seedUser(t, pool)
	offeredItem := seedItem(t, pool, offerer, offerer, false)
	unrelatedItem := seedItem(t, pool, offerer, offerer, false)
	seedProposal(t, pool, dealID, offerer, offeredItem, recipient, 1, domain.ProposalStatusPending)

	_, found, err := repo.FindOpenDealAsRecipient(ctx, unrelatedItem, recipient)
	if err != nil {
		t.Fatalf("FindOpenDealAsRecipient: %v", err)
	}
	if found {
		t.Errorf("found = true, want false (recipient is not promised unrelatedItem)")
	}
}

func TestRepository_FindOpenDealAsRecipient_NotFound_ChainAlreadyClosed(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creator := seedUser(t, pool)
	rootItem := seedItem(t, pool, creator, creator, false)
	deadline := time.Now().Add(time.Hour)
	dealID := seedDeal(t, pool, rootItem, creator, 3600, &deadline)

	offerer := seedUser(t, pool)
	recipient := seedUser(t, pool)
	offeredItem := seedItem(t, pool, offerer, offerer, false)
	seedProposal(t, pool, dealID, offerer, offeredItem, recipient, 1, domain.ProposalStatusPending)

	_, found, err := repo.FindOpenDealAsRecipient(ctx, offeredItem, recipient)
	if err != nil {
		t.Fatalf("FindOpenDealAsRecipient: %v", err)
	}
	if found {
		t.Errorf("found = true, want false (chain already closed to new participants)")
	}
}

func TestRepository_FindOpenDealAsRecipient_NotFound_NoSuchDeal(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)

	_, found, err := repo.FindOpenDealAsRecipient(context.Background(), "00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("FindOpenDealAsRecipient: %v", err)
	}
	if found {
		t.Errorf("found = true, want false")
	}
}

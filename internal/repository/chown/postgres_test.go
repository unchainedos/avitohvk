package chown

import (
	"context"
	"errors"
	"testing"
	"time"

	"avitohvk/internal/domain"
	"avitohvk/internal/repository/dbtest"

	"github.com/jackc/pgx/v5/pgconn"
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

func seedDeal(t *testing.T, pool *pgxpool.Pool, rootItemID, creatorID string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO chain_deals (root_item_id, creator_id, negotiation_window_seconds)
		 VALUES ($1::uuid, $2::uuid, 3600) RETURNING id::text`,
		rootItemID, creatorID,
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

func countWishItems(t *testing.T, pool *pgxpool.Pool, wishID, itemID string) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM wish_items WHERE wish_id = $1::uuid AND item_id = $2::uuid`, wishID, itemID,
	).Scan(&n)
	if err != nil {
		t.Fatalf("count wish_items: %v", err)
	}
	return n
}

func countWishesForUserItem(t *testing.T, pool *pgxpool.Pool, userID, itemID string) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM wishes w JOIN wish_items wi ON wi.wish_id = w.id WHERE w.user_id = $1::uuid AND wi.item_id = $2::uuid`,
		userID, itemID,
	).Scan(&n)
	if err != nil {
		t.Fatalf("count wishes: %v", err)
	}
	return n
}

func countTransactions(t *testing.T, pool *pgxpool.Pool, itemID string) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(), `SELECT count(*) FROM transactions WHERE item_id = $1::uuid`, itemID).Scan(&n)
	if err != nil {
		t.Fatalf("count transactions: %v", err)
	}
	return n
}

func pgErrCode(t *testing.T, err error) string {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("error %v is not a *pgconn.PgError", err)
	}
	return pgErr.Code
}

func TestRepository_Chown_Success_SingleHop(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	actor := seedUser(t, pool)
	targetHolder := seedUser(t, pool)
	recipient := seedUser(t, pool)
	targetItem := seedItem(t, pool, targetHolder, targetHolder, false)
	offeredItem := seedItem(t, pool, actor, actor, false)
	seedWish(t, pool, recipient, offeredItem, time.Now())

	got, err := repo.Chown(ctx, actor, targetItem, []domain.OfferedItem{{ItemID: offeredItem, Quantity: 2}})
	if err != nil {
		t.Fatalf("Chown: %v", err)
	}

	if got.ItemID != targetItem {
		t.Errorf("ItemID = %q, want %q", got.ItemID, targetItem)
	}
	if got.FromUserID != actor {
		t.Errorf("FromUserID = %q, want %q", got.FromUserID, actor)
	}
	if got.CreatedAt.IsZero() {
		t.Errorf("CreatedAt is zero")
	}
	if len(got.Hops) != 1 || got.Hops[0] != (domain.ChownHop{ItemID: offeredItem, Quantity: 2, ToUserID: recipient}) {
		t.Errorf("Hops = %+v, want a single hop to %q", got.Hops, recipient)
	}

	target := getItemState(t, pool, targetItem)
	if target.holderID != targetHolder {
		t.Errorf("target holder = %q, want unchanged %q (chown claims rights, it doesn't transfer the target)", target.holderID, targetHolder)
	}
	if !target.isLocked {
		t.Errorf("target is_locked = false, want true")
	}

	offered := getItemState(t, pool, offeredItem)
	if offered.holderID != recipient {
		t.Errorf("offered item holder = %q, want %q (real transfer)", offered.holderID, recipient)
	}
	if offered.isLocked {
		t.Errorf("offered item is_locked = true, tossing should not lock it")
	}

	if n := countTransactions(t, pool, offeredItem); n != 1 {
		t.Errorf("transactions for offered item = %d, want 1", n)
	}
	if n := countWishesForUserItem(t, pool, actor, targetItem); n != 1 {
		t.Errorf("actor's wish claims on target item = %d, want 1 (linkWish should register the claim)", n)
	}
}

func TestRepository_Chown_Success_DoesNotDuplicateExistingClaimWish(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	actor := seedUser(t, pool)
	targetHolder := seedUser(t, pool)
	targetItem := seedItem(t, pool, targetHolder, targetHolder, false)
	seedWish(t, pool, actor, targetItem, time.Now())

	if _, err := repo.Chown(ctx, actor, targetItem, nil); err != nil {
		t.Fatalf("Chown: %v", err)
	}

	if n := countWishesForUserItem(t, pool, actor, targetItem); n != 1 {
		t.Errorf("actor's wish claims on target item = %d, want 1 (must not create a duplicate)", n)
	}
}

func TestRepository_Chown_Success_EmptyOffersProducesEmptyNotNilHops(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	actor := seedUser(t, pool)
	targetHolder := seedUser(t, pool)
	targetItem := seedItem(t, pool, targetHolder, targetHolder, false)

	got, err := repo.Chown(ctx, actor, targetItem, []domain.OfferedItem{})
	if err != nil {
		t.Fatalf("Chown: %v", err)
	}
	if got.Hops == nil {
		t.Errorf("Hops is nil, want an empty non-nil slice")
	}
	if len(got.Hops) != 0 {
		t.Errorf("Hops = %+v, want empty", got.Hops)
	}
}

func TestRepository_Chown_Success_MultiHop(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	actor := seedUser(t, pool)
	targetHolder := seedUser(t, pool)
	recipient1 := seedUser(t, pool)
	recipient2 := seedUser(t, pool)
	targetItem := seedItem(t, pool, targetHolder, targetHolder, false)
	offered1 := seedItem(t, pool, actor, actor, false)
	offered2 := seedItem(t, pool, actor, actor, false)
	seedWish(t, pool, recipient1, offered1, time.Now())
	seedWish(t, pool, recipient2, offered2, time.Now())

	got, err := repo.Chown(ctx, actor, targetItem, []domain.OfferedItem{
		{ItemID: offered1, Quantity: 1},
		{ItemID: offered2, Quantity: 3},
	})
	if err != nil {
		t.Fatalf("Chown: %v", err)
	}
	if len(got.Hops) != 2 {
		t.Fatalf("Hops = %+v, want 2 entries", got.Hops)
	}
	if got.Hops[0] != (domain.ChownHop{ItemID: offered1, Quantity: 1, ToUserID: recipient1}) {
		t.Errorf("Hops[0] = %+v", got.Hops[0])
	}
	if got.Hops[1] != (domain.ChownHop{ItemID: offered2, Quantity: 3, ToUserID: recipient2}) {
		t.Errorf("Hops[1] = %+v", got.Hops[1])
	}

	if s := getItemState(t, pool, offered1); s.holderID != recipient1 {
		t.Errorf("offered1 holder = %q, want %q", s.holderID, recipient1)
	}
	if s := getItemState(t, pool, offered2); s.holderID != recipient2 {
		t.Errorf("offered2 holder = %q, want %q", s.holderID, recipient2)
	}
}

func TestRepository_Chown_TargetItemNotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	actor := seedUser(t, pool)

	_, err := repo.Chown(ctx, actor, "00000000-0000-0000-0000-000000000000", nil)
	if !errors.Is(err, ErrItemNotFound) {
		t.Errorf("err = %v, want ErrItemNotFound", err)
	}
}

func TestRepository_Chown_OwnItem(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	actor := seedUser(t, pool)
	targetItem := seedItem(t, pool, actor, actor, false)

	_, err := repo.Chown(ctx, actor, targetItem, nil)
	if !errors.Is(err, ErrOwnItem) {
		t.Errorf("err = %v, want ErrOwnItem", err)
	}

	s := getItemState(t, pool, targetItem)
	if s.isLocked {
		t.Errorf("target item is_locked = true after a rejected claim, want false (no partial side effects)")
	}
}

func TestRepository_Chown_TargetItemAlreadyLocked(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	actor := seedUser(t, pool)
	holder := seedUser(t, pool)
	targetItem := seedItem(t, pool, holder, holder, true)

	_, err := repo.Chown(ctx, actor, targetItem, nil)
	if !errors.Is(err, ErrItemLocked) {
		t.Errorf("err = %v, want ErrItemLocked", err)
	}
}

func TestRepository_Chown_OfferedItemErrors_RollBackTargetClaim(t *testing.T) {
	t.Parallel()

	t.Run("offered item not found", func(t *testing.T) {
		t.Parallel()
		pool := dbtest.NewPool(t)
		repo := NewRepository(pool)
		ctx := context.Background()

		actor := seedUser(t, pool)
		targetHolder := seedUser(t, pool)
		targetItem := seedItem(t, pool, targetHolder, targetHolder, false)

		_, err := repo.Chown(ctx, actor, targetItem, []domain.OfferedItem{{ItemID: "00000000-0000-0000-0000-000000000000", Quantity: 1}})
		if !errors.Is(err, ErrItemNotFound) {
			t.Errorf("err = %v, want ErrItemNotFound", err)
		}
		if s := getItemState(t, pool, targetItem); s.isLocked {
			t.Errorf("target item is_locked = true after a rolled-back Chown, want false")
		}
		if n := countWishesForUserItem(t, pool, actor, targetItem); n != 0 {
			t.Errorf("actor claim wishes on target = %d, want 0 (claim must roll back too)", n)
		}
	})

	t.Run("actor does not hold offered item", func(t *testing.T) {
		t.Parallel()
		pool := dbtest.NewPool(t)
		repo := NewRepository(pool)
		ctx := context.Background()

		actor := seedUser(t, pool)
		targetHolder := seedUser(t, pool)
		otherHolder := seedUser(t, pool)
		targetItem := seedItem(t, pool, targetHolder, targetHolder, false)
		offeredItem := seedItem(t, pool, otherHolder, otherHolder, false)

		_, err := repo.Chown(ctx, actor, targetItem, []domain.OfferedItem{{ItemID: offeredItem, Quantity: 1}})
		if !errors.Is(err, ErrNotItemHolder) {
			t.Errorf("err = %v, want ErrNotItemHolder", err)
		}
		if s := getItemState(t, pool, targetItem); s.isLocked {
			t.Errorf("target item is_locked = true after a rolled-back Chown, want false")
		}
	})

	t.Run("offered item is locked", func(t *testing.T) {
		t.Parallel()
		pool := dbtest.NewPool(t)
		repo := NewRepository(pool)
		ctx := context.Background()

		actor := seedUser(t, pool)
		targetHolder := seedUser(t, pool)
		targetItem := seedItem(t, pool, targetHolder, targetHolder, false)
		offeredItem := seedItem(t, pool, actor, actor, true)

		_, err := repo.Chown(ctx, actor, targetItem, []domain.OfferedItem{{ItemID: offeredItem, Quantity: 1}})
		if !errors.Is(err, ErrItemLocked) {
			t.Errorf("err = %v, want ErrItemLocked", err)
		}
		if s := getItemState(t, pool, targetItem); s.isLocked {
			t.Errorf("target item is_locked = true after a rolled-back Chown, want false")
		}
	})

	t.Run("no one wishes for the offered item", func(t *testing.T) {
		t.Parallel()
		pool := dbtest.NewPool(t)
		repo := NewRepository(pool)
		ctx := context.Background()

		actor := seedUser(t, pool)
		targetHolder := seedUser(t, pool)
		targetItem := seedItem(t, pool, targetHolder, targetHolder, false)
		offeredItem := seedItem(t, pool, actor, actor, false)

		_, err := repo.Chown(ctx, actor, targetItem, []domain.OfferedItem{{ItemID: offeredItem, Quantity: 1}})
		if !errors.Is(err, ErrRecipientNotFound) {
			t.Errorf("err = %v, want ErrRecipientNotFound", err)
		}
		if s := getItemState(t, pool, targetItem); s.isLocked {
			t.Errorf("target item is_locked = true after a rolled-back Chown, want false")
		}
	})

	t.Run("only actor's own wish exists for the offered item", func(t *testing.T) {
		t.Parallel()
		pool := dbtest.NewPool(t)
		repo := NewRepository(pool)
		ctx := context.Background()

		actor := seedUser(t, pool)
		targetHolder := seedUser(t, pool)
		targetItem := seedItem(t, pool, targetHolder, targetHolder, false)
		offeredItem := seedItem(t, pool, actor, actor, false)
		seedWish(t, pool, actor, offeredItem, time.Now())

		_, err := repo.Chown(ctx, actor, targetItem, []domain.OfferedItem{{ItemID: offeredItem, Quantity: 1}})
		if !errors.Is(err, ErrRecipientNotFound) {
			t.Errorf("err = %v, want ErrRecipientNotFound (actor's own wish must not count as a recipient)", err)
		}
	})
}

func TestRepository_Chown_PartialMultiHopRollsBackEntirely(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	actor := seedUser(t, pool)
	targetHolder := seedUser(t, pool)
	recipient1 := seedUser(t, pool)
	targetItem := seedItem(t, pool, targetHolder, targetHolder, false)
	offered1 := seedItem(t, pool, actor, actor, false)
	offered2 := seedItem(t, pool, actor, actor, false)
	seedWish(t, pool, recipient1, offered1, time.Now())

	_, err := repo.Chown(ctx, actor, targetItem, []domain.OfferedItem{
		{ItemID: offered1, Quantity: 1},
		{ItemID: offered2, Quantity: 1},
	})
	if !errors.Is(err, ErrRecipientNotFound) {
		t.Fatalf("err = %v, want ErrRecipientNotFound", err)
	}

	if s := getItemState(t, pool, offered1); s.holderID != actor {
		t.Errorf("offered1 holder = %q, want unchanged %q (the whole chown must roll back atomically)", s.holderID, actor)
	}
	if n := countTransactions(t, pool, offered1); n != 0 {
		t.Errorf("transactions for offered1 = %d, want 0 (rolled back)", n)
	}
	if s := getItemState(t, pool, targetItem); s.isLocked {
		t.Errorf("target item is_locked = true, want false (rolled back)")
	}
}

func TestRepository_Chown_EarliestWishWins(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	actor := seedUser(t, pool)
	targetHolder := seedUser(t, pool)
	earlyWisher := seedUser(t, pool)
	lateWisher := seedUser(t, pool)
	targetItem := seedItem(t, pool, targetHolder, targetHolder, false)
	offeredItem := seedItem(t, pool, actor, actor, false)

	base := time.Now().Add(-time.Hour)
	seedWish(t, pool, lateWisher, offeredItem, base.Add(time.Minute))
	seedWish(t, pool, earlyWisher, offeredItem, base)

	got, err := repo.Chown(ctx, actor, targetItem, []domain.OfferedItem{{ItemID: offeredItem, Quantity: 1}})
	if err != nil {
		t.Fatalf("Chown: %v", err)
	}
	if got.Hops[0].ToUserID != earlyWisher {
		t.Errorf("recipient = %q, want the earlier wisher %q", got.Hops[0].ToUserID, earlyWisher)
	}
}

func TestRepository_CompleteTransfer_Success(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	holder := seedUser(t, pool)
	toUser := seedUser(t, pool)
	item := seedItem(t, pool, holder, holder, true)
	dealID := seedDeal(t, pool, item, holder)
	if _, err := pool.Exec(ctx, `UPDATE items SET locked_by_deal_id = $1::uuid WHERE id = $2::uuid`, dealID, item); err != nil {
		t.Fatalf("seed locked_by_deal_id: %v", err)
	}

	if err := repo.CompleteTransfer(ctx, item, toUser); err != nil {
		t.Fatalf("CompleteTransfer: %v", err)
	}

	s := getItemState(t, pool, item)
	if s.holderID != toUser {
		t.Errorf("holder = %q, want %q", s.holderID, toUser)
	}
	if s.isLocked {
		t.Errorf("is_locked = true, want false")
	}
	if s.lockedByDealID != nil {
		t.Errorf("lockedByDealID = %v, want nil", *s.lockedByDealID)
	}
}

func TestRepository_CompleteTransfer_DeletesRecipientsWish(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	holder := seedUser(t, pool)
	toUser := seedUser(t, pool)
	item := seedItem(t, pool, holder, holder, false)
	wishID := seedWish(t, pool, toUser, item, time.Now())

	if err := repo.CompleteTransfer(ctx, item, toUser); err != nil {
		t.Fatalf("CompleteTransfer: %v", err)
	}
	if n := countWishItems(t, pool, wishID, item); n != 0 {
		t.Errorf("wish_items rows for the fulfilled wish = %d, want 0", n)
	}
}

func TestRepository_CompleteTransfer_NoMatchingWishDoesNotError(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	holder := seedUser(t, pool)
	toUser := seedUser(t, pool)
	item := seedItem(t, pool, holder, holder, false)

	if err := repo.CompleteTransfer(ctx, item, toUser); err != nil {
		t.Fatalf("CompleteTransfer: %v", err)
	}
	s := getItemState(t, pool, item)
	if s.holderID != toUser {
		t.Errorf("holder = %q, want %q", s.holderID, toUser)
	}
}

func TestRepository_CompleteTransfer_OnlyDeletesEarliestMatchingWish(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	holder := seedUser(t, pool)
	toUser := seedUser(t, pool)
	item := seedItem(t, pool, holder, holder, false)

	base := time.Now().Add(-time.Hour)
	earlyWishID := seedWish(t, pool, toUser, item, base)
	lateWishID := seedWish(t, pool, toUser, item, base.Add(time.Minute))

	if err := repo.CompleteTransfer(ctx, item, toUser); err != nil {
		t.Fatalf("CompleteTransfer: %v", err)
	}
	if n := countWishItems(t, pool, earlyWishID, item); n != 0 {
		t.Errorf("earliest wish_items row = %d, want 0 (deleted)", n)
	}
	if n := countWishItems(t, pool, lateWishID, item); n != 1 {
		t.Errorf("later wish_items row = %d, want 1 (untouched)", n)
	}
}

func TestRepository_CompleteTransfer_NonExistentItemDoesNotError(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	toUser := seedUser(t, pool)

	err := repo.CompleteTransfer(ctx, "00000000-0000-0000-0000-000000000000", toUser)
	if err != nil {
		t.Errorf("err = %v, want nil (current no-op behavior on a non-existent item)", err)
	}
}

func TestRepository_CompleteTransfer_MalformedIDs(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	holder := seedUser(t, pool)
	item := seedItem(t, pool, holder, holder, false)

	tests := []struct {
		name   string
		itemID string
		toUser string
	}{
		{name: "malformed item id", itemID: "not-a-uuid", toUser: holder},
		{name: "malformed to_user id", itemID: item, toUser: "not-a-uuid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := repo.CompleteTransfer(ctx, tt.itemID, tt.toUser)
			if err == nil {
				t.Fatalf("CompleteTransfer succeeded, want an invalid-uuid error")
			}
			if code := pgErrCode(t, err); code != "22P02" {
				t.Errorf("pg error code = %q, want 22P02 (invalid_text_representation)", code)
			}
		})
	}
}

func TestRepository_Chown_ConcurrentClaimsOnSameTargetItemSerialize(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	targetHolder := seedUser(t, pool)
	actor1 := seedUser(t, pool)
	actor2 := seedUser(t, pool)
	targetItem := seedItem(t, pool, targetHolder, targetHolder, false)

	results := make(chan error, 2)
	for _, actor := range []string{actor1, actor2} {
		go func(actor string) {
			_, err := repo.Chown(ctx, actor, targetItem, nil)
			results <- err
		}(actor)
	}

	err1 := <-results
	err2 := <-results

	successes, lockedErrs := 0, 0
	for _, err := range []error{err1, err2} {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrItemLocked):
			lockedErrs++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if successes != 1 || lockedErrs != 1 {
		t.Errorf("successes=%d lockedErrs=%d, want exactly one of each (no double claim, no double failure)", successes, lockedErrs)
	}

	if n := countWishesForUserItem(t, pool, actor1, targetItem) + countWishesForUserItem(t, pool, actor2, targetItem); n != 1 {
		t.Errorf("total claim wishes on target item = %d, want exactly 1 (only the winner registers a claim)", n)
	}
}

func TestRepository_Chown_TossingSameOfferedItemTwiceFailsOnSecond(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	actor := seedUser(t, pool)
	targetHolder := seedUser(t, pool)
	recipient := seedUser(t, pool)
	targetItem := seedItem(t, pool, targetHolder, targetHolder, false)
	offeredItem := seedItem(t, pool, actor, actor, false)
	seedWish(t, pool, recipient, offeredItem, time.Now())

	_, err := repo.Chown(ctx, actor, targetItem, []domain.OfferedItem{
		{ItemID: offeredItem, Quantity: 1},
		{ItemID: offeredItem, Quantity: 1},
	})
	if !errors.Is(err, ErrNotItemHolder) {
		t.Errorf("err = %v, want ErrNotItemHolder on the second, now-stale offer", err)
	}
	if s := getItemState(t, pool, offeredItem); s.holderID != actor {
		t.Errorf("offered item holder = %q, want unchanged %q (the whole chown rolls back)", s.holderID, actor)
	}
}

func TestRepository_Chown_PreservesFractionalQuantity(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	actor := seedUser(t, pool)
	targetHolder := seedUser(t, pool)
	recipient := seedUser(t, pool)
	targetItem := seedItem(t, pool, targetHolder, targetHolder, false)
	offeredItem := seedItem(t, pool, actor, actor, false)
	seedWish(t, pool, recipient, offeredItem, time.Now())

	got, err := repo.Chown(ctx, actor, targetItem, []domain.OfferedItem{{ItemID: offeredItem, Quantity: 1.5}})
	if err != nil {
		t.Fatalf("Chown: %v", err)
	}
	if got.Hops[0].Quantity != 1.5 {
		t.Errorf("Hops[0].Quantity = %v, want 1.5", got.Hops[0].Quantity)
	}

	var stored float64
	if err := pool.QueryRow(ctx, `SELECT quantity FROM transactions WHERE item_id = $1::uuid`, offeredItem).Scan(&stored); err != nil {
		t.Fatalf("query stored quantity: %v", err)
	}
	if stored != 1.5 {
		t.Errorf("stored transactions.quantity = %v, want 1.5", stored)
	}
}

func TestRepository_Chown_FailsWithAlreadyCanceledContext(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := repo.Chown(ctx, "irrelevant", "irrelevant", nil)
	if err == nil {
		t.Fatalf("Chown succeeded with an already-canceled context, want an error")
	}
}

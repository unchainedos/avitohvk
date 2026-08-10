package chown

import (
	"context"
	"errors"
	"testing"
	"time"

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

func pgErrCode(t *testing.T, err error) string {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("error %v is not a *pgconn.PgError", err)
	}
	return pgErr.Code
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

func TestRepository_EnsureWish_CreatesWishWhenNoneExists(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	user := seedUser(t, pool)
	holder := seedUser(t, pool)
	item := seedItem(t, pool, holder, holder, false)

	if err := repo.EnsureWish(ctx, user, item); err != nil {
		t.Fatalf("EnsureWish: %v", err)
	}
	if n := countWishesForUserItem(t, pool, user, item); n != 1 {
		t.Errorf("wishes for (user, item) = %d, want 1", n)
	}
}

func TestRepository_EnsureWish_NoOpWhenWishAlreadyExists(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	user := seedUser(t, pool)
	holder := seedUser(t, pool)
	item := seedItem(t, pool, holder, holder, false)
	seedWish(t, pool, user, item, time.Now())

	if err := repo.EnsureWish(ctx, user, item); err != nil {
		t.Fatalf("EnsureWish: %v", err)
	}
	if n := countWishesForUserItem(t, pool, user, item); n != 1 {
		t.Errorf("wishes for (user, item) = %d, want 1 (must not create a duplicate)", n)
	}
}

func TestRepository_EnsureWish_MalformedIDs(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	user := seedUser(t, pool)
	holder := seedUser(t, pool)
	item := seedItem(t, pool, holder, holder, false)

	tests := []struct {
		name   string
		userID string
		itemID string
	}{
		{name: "malformed user id", userID: "not-a-uuid", itemID: item},
		{name: "malformed item id", userID: user, itemID: "not-a-uuid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := repo.EnsureWish(ctx, tt.userID, tt.itemID)
			if err == nil {
				t.Fatalf("EnsureWish succeeded, want an invalid-uuid error")
			}
			if code := pgErrCode(t, err); code != "22P02" {
				t.Errorf("pg error code = %q, want 22P02 (invalid_text_representation)", code)
			}
		})
	}
}

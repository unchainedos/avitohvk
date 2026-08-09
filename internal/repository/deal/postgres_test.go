package deal

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

func seedItem(t *testing.T, pool *pgxpool.Pool, authorID, holderID string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO items (author_id, holder_id, title) VALUES ($1::uuid, $2::uuid, 'item') RETURNING id::text`,
		authorID, holderID,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed item: %v", err)
	}
	return id
}

func pgErrCode(t *testing.T, err error) string {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("error %v is not a *pgconn.PgError", err)
	}
	return pgErr.Code
}

func TestRepository_Create_Success(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creatorID := seedUser(t, pool)
	itemID := seedItem(t, pool, creatorID, creatorID)

	d, err := repo.Create(ctx, itemID, creatorID, 24*time.Hour)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if d.ID == "" {
		t.Errorf("ID is empty, want a generated uuid")
	}
	if d.RootItemID != itemID {
		t.Errorf("RootItemID = %q, want %q", d.RootItemID, itemID)
	}
	if d.CreatorID != creatorID {
		t.Errorf("CreatorID = %q, want %q", d.CreatorID, creatorID)
	}
	if d.Status != domain.DealStatusPending {
		t.Errorf("Status = %q, want %q", d.Status, domain.DealStatusPending)
	}
	if d.NegotiationWindow != 24*time.Hour {
		t.Errorf("NegotiationWindow = %v, want %v", d.NegotiationWindow, 24*time.Hour)
	}
	if d.DeadlineAt != nil {
		t.Errorf("DeadlineAt = %v, want nil (Create never sets a deadline)", d.DeadlineAt)
	}
	if d.CreatedAt.IsZero() {
		t.Errorf("CreatedAt is zero")
	}
	if d.UpdatedAt.IsZero() {
		t.Errorf("UpdatedAt is zero")
	}
}

func TestRepository_Create_RootItemNotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creatorID := seedUser(t, pool)

	_, err := repo.Create(ctx, "00000000-0000-0000-0000-000000000000", creatorID, time.Hour)
	if !errors.Is(err, ErrRootItemNotFound) {
		t.Errorf("err = %v, want ErrRootItemNotFound", err)
	}
}

func TestRepository_Create_UnknownCreatorMapsToErrCreatorNotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creatorID := seedUser(t, pool)
	itemID := seedItem(t, pool, creatorID, creatorID)

	_, err := repo.Create(ctx, itemID, "00000000-0000-0000-0000-000000000000", time.Hour)
	if !errors.Is(err, ErrCreatorNotFound) {
		t.Errorf("err = %v, want ErrCreatorNotFound", err)
	}
	if errors.Is(err, ErrRootItemNotFound) {
		t.Errorf("err also matches ErrRootItemNotFound — a bad creator_id must not be mislabeled as a bad root_item_id")
	}
}

func TestRepository_Create_NegotiationWindowMustBePositive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
	}{
		{name: "zero", duration: 0},
		{name: "negative", duration: -time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pool := dbtest.NewPool(t)
			repo := NewRepository(pool)
			ctx := context.Background()

			creatorID := seedUser(t, pool)
			itemID := seedItem(t, pool, creatorID, creatorID)

			_, err := repo.Create(ctx, itemID, creatorID, tt.duration)
			if err == nil {
				t.Fatalf("Create succeeded, want a check-constraint violation")
			}
			if errors.Is(err, ErrRootItemNotFound) || errors.Is(err, ErrNotFound) {
				t.Fatalf("err = %v, want an unmapped check-constraint error, not a repository sentinel", err)
			}
			if code := pgErrCode(t, err); code != "23514" {
				t.Errorf("pg error code = %q, want 23514 (check_violation)", code)
			}
		})
	}
}

func TestRepository_Create_MalformedIDsAreNotMappedToSentinels(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creatorID := seedUser(t, pool)
	itemID := seedItem(t, pool, creatorID, creatorID)

	tests := []struct {
		name       string
		rootItemID string
		creatorID  string
	}{
		{name: "malformed root item id", rootItemID: "not-a-uuid", creatorID: creatorID},
		{name: "malformed creator id", rootItemID: itemID, creatorID: "not-a-uuid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := repo.Create(ctx, tt.rootItemID, tt.creatorID, time.Hour)
			if err == nil {
				t.Fatalf("Create succeeded, want an invalid-uuid error")
			}
			if errors.Is(err, ErrRootItemNotFound) || errors.Is(err, ErrCreatorNotFound) {
				t.Errorf("err = %v, want the raw cast error surfaced unmapped, not a repository sentinel", err)
			}
			if code := pgErrCode(t, err); code != "22P02" {
				t.Errorf("pg error code = %q, want 22P02 (invalid_text_representation)", code)
			}
		})
	}
}

func TestRepository_GetByID_Success(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creatorID := seedUser(t, pool)
	itemID := seedItem(t, pool, creatorID, creatorID)
	created, err := repo.Create(ctx, itemID, creatorID, time.Hour)
	if err != nil {
		t.Fatalf("seed Create: %v", err)
	}

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != created {
		t.Errorf("GetByID = %+v, want %+v", got, created)
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)

	_, err := repo.GetByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestRepository_GetByID_MalformedIDIsNotMappedToNotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)

	_, err := repo.GetByID(context.Background(), "not-a-uuid")
	if err == nil {
		t.Fatalf("GetByID succeeded, want an invalid-uuid error")
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("err wraps ErrNotFound, want the raw cast error surfaced unmapped: %v", err)
	}
	if code := pgErrCode(t, err); code != "22P02" {
		t.Errorf("pg error code = %q, want 22P02 (invalid_text_representation)", code)
	}
}

func TestRepository_GetByID_ScansNonNilDeadline(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creatorID := seedUser(t, pool)
	itemID := seedItem(t, pool, creatorID, creatorID)
	created, err := repo.Create(ctx, itemID, creatorID, time.Hour)
	if err != nil {
		t.Fatalf("seed Create: %v", err)
	}

	deadline := created.CreatedAt.Add(2 * time.Hour).UTC()
	if _, err := pool.Exec(ctx, `UPDATE chain_deals SET deadline_at = $1 WHERE id = $2::uuid`, deadline, created.ID); err != nil {
		t.Fatalf("seed deadline_at: %v", err)
	}

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.DeadlineAt == nil {
		t.Fatalf("DeadlineAt is nil, want %v", deadline)
	}
	if !got.DeadlineAt.Equal(deadline) {
		t.Errorf("DeadlineAt = %v, want %v", *got.DeadlineAt, deadline)
	}
}

func TestRepository_UpdateStatus_Success(t *testing.T) {
	t.Parallel()

	statuses := []domain.DealStatus{
		domain.DealStatusPending,
		domain.DealStatusConfirmed,
		domain.DealStatusCompleted,
		domain.DealStatusCanceled,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			pool := dbtest.NewPool(t)
			repo := NewRepository(pool)
			ctx := context.Background()

			creatorID := seedUser(t, pool)
			itemID := seedItem(t, pool, creatorID, creatorID)
			created, err := repo.Create(ctx, itemID, creatorID, time.Hour)
			if err != nil {
				t.Fatalf("seed Create: %v", err)
			}

			got, err := repo.UpdateStatus(ctx, created.ID, status)
			if err != nil {
				t.Fatalf("UpdateStatus: %v", err)
			}
			if got.Status != status {
				t.Errorf("Status = %q, want %q", got.Status, status)
			}
			if got.ID != created.ID {
				t.Errorf("ID = %q, want %q", got.ID, created.ID)
			}
		})
	}
}

func TestRepository_UpdateStatus_BumpsUpdatedAt(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creatorID := seedUser(t, pool)
	itemID := seedItem(t, pool, creatorID, creatorID)
	created, err := repo.Create(ctx, itemID, creatorID, time.Hour)
	if err != nil {
		t.Fatalf("seed Create: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	got, err := repo.UpdateStatus(ctx, created.ID, domain.DealStatusConfirmed)
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if !got.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want strictly after the pre-update value %v (trg_chain_deals_updated_at should have fired)", got.UpdatedAt, created.UpdatedAt)
	}
}

func TestRepository_UpdateStatus_NotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)

	_, err := repo.UpdateStatus(context.Background(), "00000000-0000-0000-0000-000000000000", domain.DealStatusConfirmed)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestRepository_UpdateStatus_MalformedIDIsNotMappedToNotFound(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)

	_, err := repo.UpdateStatus(context.Background(), "not-a-uuid", domain.DealStatusConfirmed)
	if err == nil {
		t.Fatalf("UpdateStatus succeeded, want an invalid-uuid error")
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("err wraps ErrNotFound, want the raw cast error surfaced unmapped: %v", err)
	}
	if code := pgErrCode(t, err); code != "22P02" {
		t.Errorf("pg error code = %q, want 22P02 (invalid_text_representation)", code)
	}
}

func TestRepository_LockDeal_BlocksConcurrentLockOnSameDeal(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	creatorID := seedUser(t, pool)
	itemID := seedItem(t, pool, creatorID, creatorID)
	created, err := repo.Create(ctx, itemID, creatorID, time.Hour)
	if err != nil {
		t.Fatalf("seed Create: %v", err)
	}

	release1, err := repo.LockDeal(ctx, created.ID)
	if err != nil {
		t.Fatalf("first LockDeal: %v", err)
	}

	acquired := make(chan func(context.Context))
	go func() {
		release2, err := repo.LockDeal(ctx, created.ID)
		if err != nil {
			t.Errorf("second LockDeal: %v", err)
			close(acquired)
			return
		}
		acquired <- release2
	}()

	select {
	case <-acquired:
		t.Fatalf("second LockDeal acquired the lock while the first still holds it")
	case <-time.After(200 * time.Millisecond):
	}

	release1(ctx)

	select {
	case release2 := <-acquired:
		if release2 != nil {
			release2(ctx)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("second LockDeal did not acquire the lock after the first was released")
	}
}

func TestRepository_LockDeal_DifferentDealsDoNotBlockEachOther(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	release1, err := repo.LockDeal(ctx, "deal-a")
	if err != nil {
		t.Fatalf("LockDeal(deal-a): %v", err)
	}
	defer release1(ctx)

	done := make(chan error, 1)
	go func() {
		release2, err := repo.LockDeal(ctx, "deal-b")
		if err == nil {
			release2(ctx)
		}
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("LockDeal(deal-b): %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("LockDeal on an unrelated deal id blocked behind deal-a's lock")
	}
}

func TestRepository_LockDeal_ReleaseAllowsReacquire(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	release, err := repo.LockDeal(ctx, "deal-reacquire")
	if err != nil {
		t.Fatalf("first LockDeal: %v", err)
	}
	release(ctx)

	release2, err := repo.LockDeal(ctx, "deal-reacquire")
	if err != nil {
		t.Fatalf("second LockDeal after release: %v", err)
	}
	release2(ctx)
}

func TestRepository_LockDeal_FailsWithAlreadyCanceledContext(t *testing.T) {
	t.Parallel()

	pool := dbtest.NewPool(t)
	repo := NewRepository(pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	release, err := repo.LockDeal(ctx, "deal-canceled")
	if err == nil {
		release(context.Background())
		t.Fatalf("LockDeal succeeded with an already-canceled context, want an error")
	}
	if release != nil {
		t.Errorf("release func is non-nil, want nil on error")
	}
}

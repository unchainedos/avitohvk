package chown

import (
	"context"
	"errors"
	"testing"
	"time"

	"avitohvk/internal/domain"
	"avitohvk/internal/dto"
	statusErrors "avitohvk/internal/errors"
	chownrepo "avitohvk/internal/repository/chown"
)

type fakeRepo struct {
	result domain.Chown
	err    error

	called     bool
	gotCtx     context.Context
	gotActorID string
	gotItemID  string
	gotOffers  []domain.OfferedItem
}

func (f *fakeRepo) Chown(ctx context.Context, actorID, itemID string, offers []domain.OfferedItem) (domain.Chown, error) {
	f.called = true
	f.gotCtx = ctx
	f.gotActorID = actorID
	f.gotItemID = itemID
	f.gotOffers = offers
	return f.result, f.err
}

func TestService_Chown_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		itemID    string
		req       dto.ChownRequest
		wantErrIs error
	}{
		{
			name:      "empty item_id",
			itemID:    "",
			req:       dto.ChownRequest{OfferedItems: []dto.OfferedItem{{ItemID: "x", Quantity: 1}}},
			wantErrIs: statusErrors.ErrBadRequest,
		},
		{
			name:      "no offered items",
			itemID:    "target",
			req:       dto.ChownRequest{OfferedItems: nil},
			wantErrIs: statusErrors.ErrBadRequest,
		},
		{
			name:   "an offered item missing item_id",
			itemID: "target",
			req: dto.ChownRequest{OfferedItems: []dto.OfferedItem{
				{ItemID: "ok-one", Quantity: 1},
				{ItemID: "", Quantity: 1},
			}},
			wantErrIs: statusErrors.ErrBadRequest,
		},
		{
			name:      "an offered item with zero quantity",
			itemID:    "target",
			req:       dto.ChownRequest{OfferedItems: []dto.OfferedItem{{ItemID: "x", Quantity: 0}}},
			wantErrIs: statusErrors.ErrBadRequest,
		},
		{
			name:      "an offered item with negative quantity",
			itemID:    "target",
			req:       dto.ChownRequest{OfferedItems: []dto.OfferedItem{{ItemID: "x", Quantity: -1}}},
			wantErrIs: statusErrors.ErrBadRequest,
		},
		{
			name:   "offering the target item itself, as the second offered item",
			itemID: "target",
			req: dto.ChownRequest{OfferedItems: []dto.OfferedItem{
				{ItemID: "ok-one", Quantity: 1},
				{ItemID: "target", Quantity: 1},
			}},
			wantErrIs: statusErrors.ErrBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeRepo{}
			svc := NewService(repo)

			_, err := svc.Chown(context.Background(), "actor-1", tt.itemID, tt.req)

			if err == nil {
				t.Fatalf("Chown() error = nil, want non-nil")
			}
			if !errors.Is(err, tt.wantErrIs) {
				t.Errorf("Chown() error = %v, want wrapping %v", err, tt.wantErrIs)
			}
			if repo.called {
				t.Errorf("Chown() called the repository despite a validation failure")
			}
		})
	}
}

func TestService_Chown_ValidationPrecedence(t *testing.T) {
	t.Parallel()

	// item_id and offered_items are both invalid at once: the item_id check
	// must win, matching the order the checks appear in the source. If a
	// refactor ever reorders them, this pins the observable error message
	// so the change is visible instead of silently altering client-facing
	// behavior.
	repo := &fakeRepo{}
	svc := NewService(repo)

	_, err := svc.Chown(context.Background(), "actor-1", "", dto.ChownRequest{OfferedItems: nil})

	wantText := statusErrors.ErrBadRequest.Error() + ": item_id required"
	if err == nil || err.Error() != wantText {
		t.Errorf("Chown() error = %v, want %q (item_id check must take precedence)", err, wantText)
	}
}

func TestService_Chown_RepositoryErrorMapping(t *testing.T) {
	t.Parallel()

	validReq := dto.ChownRequest{OfferedItems: []dto.OfferedItem{{ItemID: "offer-1", Quantity: 1}}}

	tests := []struct {
		name        string
		repoErr     error
		wantErrIs   error
		wantErrText string
	}{
		{
			name:        "item not found",
			repoErr:     chownrepo.ErrItemNotFound,
			wantErrIs:   statusErrors.ErrNotFound,
			wantErrText: "item not found",
		},
		{
			name:        "not item holder",
			repoErr:     chownrepo.ErrNotItemHolder,
			wantErrIs:   statusErrors.ErrConflict,
			wantErrText: "you do not hold this item",
		},
		{
			name:        "recipient not found",
			repoErr:     chownrepo.ErrRecipientNotFound,
			wantErrIs:   statusErrors.ErrNotFound,
			wantErrText: "no one wishes for this item",
		},
		{
			name:        "own item",
			repoErr:     chownrepo.ErrOwnItem,
			wantErrIs:   statusErrors.ErrBadRequest,
			wantErrText: "cannot chown your own item",
		},
		{
			name:        "item locked",
			repoErr:     chownrepo.ErrItemLocked,
			wantErrIs:   statusErrors.ErrConflict,
			wantErrText: "someone else already has exclusive rights to this item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeRepo{err: tt.repoErr}
			svc := NewService(repo)

			_, err := svc.Chown(context.Background(), "actor-1", "target", validReq)

			if err == nil {
				t.Fatalf("Chown() error = nil, want non-nil")
			}
			if !errors.Is(err, tt.wantErrIs) {
				t.Errorf("Chown() error = %v, want wrapping %v", err, tt.wantErrIs)
			}
			if err.Error() != tt.wantErrIs.Error()+": "+tt.wantErrText {
				t.Errorf("Chown() error text = %q, want %q", err.Error(), tt.wantErrIs.Error()+": "+tt.wantErrText)
			}
			if !repo.called {
				t.Errorf("Chown() did not call the repository")
			}
		})
	}
}

func TestService_Chown_UnmappedRepositoryErrorPassesThroughUnwrapped(t *testing.T) {
	t.Parallel()

	boom := errors.New("connection reset by peer")
	repo := &fakeRepo{err: boom}
	svc := NewService(repo)

	_, err := svc.Chown(context.Background(), "actor-1", "target",
		dto.ChownRequest{OfferedItems: []dto.OfferedItem{{ItemID: "offer-1", Quantity: 1}}})

	if !errors.Is(err, boom) {
		t.Errorf("Chown() error = %v, want it to wrap the original unmapped error %v", err, boom)
	}
	if errors.Is(err, statusErrors.ErrBadRequest) || errors.Is(err, statusErrors.ErrNotFound) ||
		errors.Is(err, statusErrors.ErrConflict) || errors.Is(err, statusErrors.ErrUnauthorized) {
		t.Errorf("Chown() error = %v, an unmapped repository error must not accidentally match a status sentinel", err)
	}
}

func TestService_Chown_Success(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	repo := &fakeRepo{
		result: domain.Chown{
			ItemID:     "target",
			FromUserID: "actor-1",
			Hops: []domain.ChownHop{
				{ItemID: "offer-1", Quantity: 2, ToUserID: "recipient-1"},
				{ItemID: "offer-2", Quantity: 1, ToUserID: "recipient-2"},
			},
			CreatedAt: createdAt,
		},
	}
	svc := NewService(repo)

	req := dto.ChownRequest{OfferedItems: []dto.OfferedItem{
		{ItemID: "offer-1", Quantity: 2},
		{ItemID: "offer-2", Quantity: 1},
	}}

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "trace-id-123")

	got, err := svc.Chown(ctx, "actor-1", "target", req)
	if err != nil {
		t.Fatalf("Chown() unexpected error: %v", err)
	}

	if repo.gotCtx == nil || repo.gotCtx.Value(ctxKey{}) != "trace-id-123" {
		t.Errorf("Chown() did not forward the caller's context to the repository unchanged")
	}
	if repo.gotActorID != "actor-1" {
		t.Errorf("repo received actorID = %q, want %q", repo.gotActorID, "actor-1")
	}
	if repo.gotItemID != "target" {
		t.Errorf("repo received itemID = %q, want %q", repo.gotItemID, "target")
	}
	wantOffers := []domain.OfferedItem{
		{ItemID: "offer-1", Quantity: 2},
		{ItemID: "offer-2", Quantity: 1},
	}
	if len(repo.gotOffers) != len(wantOffers) {
		t.Fatalf("repo received %d offers, want %d", len(repo.gotOffers), len(wantOffers))
	}
	for i, o := range wantOffers {
		if repo.gotOffers[i] != o {
			t.Errorf("repo received offer[%d] = %+v, want %+v", i, repo.gotOffers[i], o)
		}
	}

	want := dto.ChownResponse{
		ItemID:     "target",
		FromUserID: "actor-1",
		Hops: []dto.ChownHop{
			{ItemID: "offer-1", Quantity: 2, ToUserID: "recipient-1"},
			{ItemID: "offer-2", Quantity: 1, ToUserID: "recipient-2"},
		},
		CreatedAt: createdAt,
	}
	if got.ItemID != want.ItemID || got.FromUserID != want.FromUserID || !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("Chown() = %+v, want %+v", got, want)
	}
	if len(got.Hops) != len(want.Hops) {
		t.Fatalf("Chown() hops = %+v, want %+v", got.Hops, want.Hops)
	}
	for i := range want.Hops {
		if got.Hops[i] != want.Hops[i] {
			t.Errorf("Chown() hop[%d] = %+v, want %+v", i, got.Hops[i], want.Hops[i])
		}
	}
}

func TestService_Chown_SuccessWithNoHopsProducesEmptyNotNilSlice(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{result: domain.Chown{ItemID: "target", FromUserID: "actor-1", Hops: nil}}
	svc := NewService(repo)

	got, err := svc.Chown(context.Background(), "actor-1", "target",
		dto.ChownRequest{OfferedItems: []dto.OfferedItem{{ItemID: "offer-1", Quantity: 1}}})
	if err != nil {
		t.Fatalf("Chown() unexpected error: %v", err)
	}
	if got.Hops == nil {
		t.Errorf("Chown() Hops = nil, want a non-nil empty slice")
	}
	if len(got.Hops) != 0 {
		t.Errorf("Chown() Hops = %+v, want empty", got.Hops)
	}
}

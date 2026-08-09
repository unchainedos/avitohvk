package proposal

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"avitohvk/internal/domain"
	"avitohvk/internal/dto"
	statusErrors "avitohvk/internal/errors"
	dealrepo "avitohvk/internal/repository/deal"
	proposalrepo "avitohvk/internal/repository/proposal"

	"github.com/jackc/pgx/v5/pgconn"
)

type fakeDealRepo struct {
	createFunc       func(ctx context.Context, rootItemID, creatorID string, negotiationWindow time.Duration) (domain.Deal, error)
	getByIDFunc      func(ctx context.Context, id string) (domain.Deal, error)
	updateStatusFunc func(ctx context.Context, id string, status domain.DealStatus) (domain.Deal, error)
	lockErr          error

	updateStatusCalls []domain.DealStatus
	lockDealCalled    bool
	lockDealID        string
	released          bool
	releaseCtx        context.Context
}

func (f *fakeDealRepo) Create(ctx context.Context, rootItemID, creatorID string, negotiationWindow time.Duration) (domain.Deal, error) {
	if f.createFunc != nil {
		return f.createFunc(ctx, rootItemID, creatorID, negotiationWindow)
	}
	return domain.Deal{}, nil
}

func (f *fakeDealRepo) GetByID(ctx context.Context, id string) (domain.Deal, error) {
	if f.getByIDFunc != nil {
		return f.getByIDFunc(ctx, id)
	}
	return domain.Deal{}, nil
}

func (f *fakeDealRepo) UpdateStatus(ctx context.Context, id string, status domain.DealStatus) (domain.Deal, error) {
	f.updateStatusCalls = append(f.updateStatusCalls, status)
	if f.updateStatusFunc != nil {
		return f.updateStatusFunc(ctx, id, status)
	}
	return domain.Deal{}, nil
}

func (f *fakeDealRepo) LockDeal(_ context.Context, dealID string) (func(context.Context), error) {
	f.lockDealCalled = true
	f.lockDealID = dealID
	if f.lockErr != nil {
		return nil, f.lockErr
	}
	return func(ctx context.Context) {
		f.released = true
		f.releaseCtx = ctx
	}, nil
}

type fakeProposalRepo struct {
	createFunc              func(ctx context.Context, dealID, participantID, itemID string, quantity float64) (domain.Proposal, error)
	getByDealAndParticipant func(ctx context.Context, dealID, participantID string) (domain.Proposal, error)
	updateFunc              func(ctx context.Context, dealID, participantID string, upd domain.ProposalUpdate) (domain.Proposal, error)
	setStatusFunc           func(ctx context.Context, dealID, participantID string, status domain.ProposalStatus) (domain.Proposal, error)
	listForUserFunc         func(ctx context.Context, userID string) ([]domain.Proposal, error)
	allAcceptedFunc         func(ctx context.Context, dealID string) (bool, error)
	tryLockChainFunc        func(ctx context.Context, dealID string) error
	listTransfersFunc       func(ctx context.Context, dealID string) ([]domain.ItemTransfer, error)

	unlockAllForDealErr  error
	declineAllExceptErr  error
	declineAllForDealErr error

	unlockAllForDealCalls  []string
	declineAllExceptCalls  [][2]string
	declineAllForDealCalls []string
}

func (f *fakeProposalRepo) Create(ctx context.Context, dealID, participantID, itemID string, quantity float64) (domain.Proposal, error) {
	if f.createFunc != nil {
		return f.createFunc(ctx, dealID, participantID, itemID, quantity)
	}
	return domain.Proposal{}, nil
}

func (f *fakeProposalRepo) GetByDealAndParticipant(ctx context.Context, dealID, participantID string) (domain.Proposal, error) {
	if f.getByDealAndParticipant != nil {
		return f.getByDealAndParticipant(ctx, dealID, participantID)
	}
	return domain.Proposal{}, proposalrepo.ErrNotFound
}

func (f *fakeProposalRepo) Update(ctx context.Context, dealID, participantID string, upd domain.ProposalUpdate) (domain.Proposal, error) {
	if f.updateFunc != nil {
		return f.updateFunc(ctx, dealID, participantID, upd)
	}
	return domain.Proposal{}, nil
}

func (f *fakeProposalRepo) SetStatus(ctx context.Context, dealID, participantID string, status domain.ProposalStatus) (domain.Proposal, error) {
	if f.setStatusFunc != nil {
		return f.setStatusFunc(ctx, dealID, participantID, status)
	}
	return domain.Proposal{}, nil
}

func (f *fakeProposalRepo) ListForUser(ctx context.Context, userID string) ([]domain.Proposal, error) {
	if f.listForUserFunc != nil {
		return f.listForUserFunc(ctx, userID)
	}
	return nil, nil
}

func (f *fakeProposalRepo) AllAccepted(ctx context.Context, dealID string) (bool, error) {
	if f.allAcceptedFunc != nil {
		return f.allAcceptedFunc(ctx, dealID)
	}
	return false, nil
}

func (f *fakeProposalRepo) TryLockChain(ctx context.Context, dealID string) error {
	if f.tryLockChainFunc != nil {
		return f.tryLockChainFunc(ctx, dealID)
	}
	return nil
}

func (f *fakeProposalRepo) UnlockAllForDeal(_ context.Context, dealID string) error {
	f.unlockAllForDealCalls = append(f.unlockAllForDealCalls, dealID)
	return f.unlockAllForDealErr
}

func (f *fakeProposalRepo) DeclineAllExcept(_ context.Context, dealID, actorID string) error {
	f.declineAllExceptCalls = append(f.declineAllExceptCalls, [2]string{dealID, actorID})
	return f.declineAllExceptErr
}

func (f *fakeProposalRepo) DeclineAllForDeal(_ context.Context, dealID string) error {
	f.declineAllForDealCalls = append(f.declineAllForDealCalls, dealID)
	return f.declineAllForDealErr
}

func (f *fakeProposalRepo) ListTransfers(ctx context.Context, dealID string) ([]domain.ItemTransfer, error) {
	if f.listTransfersFunc != nil {
		return f.listTransfersFunc(ctx, dealID)
	}
	return nil, nil
}

type fakeChownRepo struct {
	completeTransferFunc func(ctx context.Context, itemID, toUserID string) error
	calls                []domain.ItemTransfer
}

func (f *fakeChownRepo) CompleteTransfer(_ context.Context, itemID, toUserID string) error {
	f.calls = append(f.calls, domain.ItemTransfer{ItemID: itemID, ToUserID: toUserID})
	if f.completeTransferFunc != nil {
		return f.completeTransferFunc(context.Background(), itemID, toUserID)
	}
	return nil
}

func deadlockErr() error {
	return &pgconn.PgError{Code: "40P01", Message: "deadlock detected"}
}

func newService(d *fakeDealRepo, p *fakeProposalRepo, c *fakeChownRepo) *Service {
	if d == nil {
		d = &fakeDealRepo{}
	}
	if p == nil {
		p = &fakeProposalRepo{}
	}
	if c == nil {
		c = &fakeChownRepo{}
	}
	return NewService(d, p, c)
}

func TestIsDeadlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: false},
		{name: "plain non-pg error", err: errors.New("boom"), want: false},
		{name: "pg error with a different code", err: &pgconn.PgError{Code: "23505"}, want: false},
		{name: "pg error with the deadlock code", err: &pgconn.PgError{Code: "40P01"}, want: true},
		{name: "deadlock wrapped by another error", err: fmt.Errorf("query failed: %w", deadlockErr()), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isDeadlock(tt.err); got != tt.want {
				t.Errorf("isDeadlock(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestRetryOnDeadlock(t *testing.T) {
	t.Parallel()

	t.Run("succeeds immediately, exactly one call", func(t *testing.T) {
		t.Parallel()
		calls := 0
		err := retryOnDeadlock(func() error {
			calls++
			return nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if calls != 1 {
			t.Errorf("calls = %d, want 1", calls)
		}
	})

	t.Run("a non-deadlock error is not retried", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("not a deadlock")
		calls := 0
		err := retryOnDeadlock(func() error {
			calls++
			return boom
		})
		if !errors.Is(err, boom) {
			t.Errorf("error = %v, want %v", err, boom)
		}
		if calls != 1 {
			t.Errorf("calls = %d, want 1 (must not retry a non-deadlock error)", calls)
		}
	})

	t.Run("retries past transient deadlocks then succeeds", func(t *testing.T) {
		t.Parallel()
		calls := 0
		err := retryOnDeadlock(func() error {
			calls++
			if calls < 4 {
				return deadlockErr()
			}
			return nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if calls != 4 {
			t.Errorf("calls = %d, want 4", calls)
		}
	})

	t.Run("exhausts the retry budget and surfaces the last deadlock error", func(t *testing.T) {
		t.Parallel()
		calls := 0
		err := retryOnDeadlock(func() error {
			calls++
			return deadlockErr()
		})
		if !isDeadlock(err) {
			t.Errorf("error = %v, want a deadlock error", err)
		}
		if calls != maxDeadlockRetries+1 {
			t.Errorf("calls = %d, want %d", calls, maxDeadlockRetries+1)
		}
	})
}

func TestService_GetByID(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		want := domain.Deal{ID: "deal-1", Status: domain.DealStatusPending}
		deals := &fakeDealRepo{getByIDFunc: func(_ context.Context, id string) (domain.Deal, error) {
			if id != "deal-1" {
				t.Errorf("GetByID called with id = %q, want deal-1", id)
			}
			return want, nil
		}}
		svc := newService(deals, nil, nil)

		got, err := svc.GetByID(context.Background(), "deal-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("GetByID() = %+v, want %+v", got, want)
		}
	})

	t.Run("not found is wrapped", func(t *testing.T) {
		t.Parallel()
		deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
			return domain.Deal{}, dealrepo.ErrNotFound
		}}
		svc := newService(deals, nil, nil)

		_, err := svc.GetByID(context.Background(), "missing")
		if !errors.Is(err, statusErrors.ErrNotFound) {
			t.Errorf("error = %v, want wrapping ErrNotFound", err)
		}
	})

	t.Run("unmapped error passes through", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("connection reset")
		deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
			return domain.Deal{}, boom
		}}
		svc := newService(deals, nil, nil)

		_, err := svc.GetByID(context.Background(), "x")
		if !errors.Is(err, boom) {
			t.Errorf("error = %v, want wrapping %v", err, boom)
		}
	})
}

func TestService_CreateDeal_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  dto.CreateDealRequest
	}{
		{name: "empty item_id", req: dto.CreateDealRequest{RootItemID: "root", ItemID: "", Quantity: 1}},
		{name: "quantity zero", req: dto.CreateDealRequest{RootItemID: "root", ItemID: "x", Quantity: 0}},
		{name: "quantity negative", req: dto.CreateDealRequest{RootItemID: "root", ItemID: "x", Quantity: -1}},
		{name: "empty root_item_id", req: dto.CreateDealRequest{RootItemID: "", ItemID: "x", Quantity: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			deals := &fakeDealRepo{}
			svc := newService(deals, nil, nil)

			_, err := svc.CreateDeal(context.Background(), "actor-1", tt.req)
			if !errors.Is(err, statusErrors.ErrBadRequest) {
				t.Errorf("error = %v, want wrapping ErrBadRequest", err)
			}
			if len(deals.updateStatusCalls) != 0 {
				t.Errorf("deal repo touched despite a validation failure")
			}
		})
	}
}

func TestService_CreateDeal_RootItemNotFound(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{createFunc: func(context.Context, string, string, time.Duration) (domain.Deal, error) {
		return domain.Deal{}, dealrepo.ErrRootItemNotFound
	}}
	svc := newService(deals, nil, nil)

	_, err := svc.CreateDeal(context.Background(), "actor-1", dto.CreateDealRequest{RootItemID: "missing", ItemID: "x", Quantity: 1})
	if !errors.Is(err, statusErrors.ErrNotFound) {
		t.Errorf("error = %v, want wrapping ErrNotFound", err)
	}
}

func TestService_CreateDeal_UnmappedRepositoryErrorPassesThrough(t *testing.T) {
	t.Parallel()

	boom := errors.New("db down")
	deals := &fakeDealRepo{createFunc: func(context.Context, string, string, time.Duration) (domain.Deal, error) {
		return domain.Deal{}, boom
	}}
	svc := newService(deals, nil, nil)

	_, err := svc.CreateDeal(context.Background(), "actor-1", dto.CreateDealRequest{RootItemID: "root", ItemID: "x", Quantity: 1})
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want wrapping %v", err, boom)
	}
}

func TestService_CreateDeal_ValidationPrecedence(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{}
	svc := newService(deals, nil, nil)

	_, err := svc.CreateDeal(context.Background(), "actor-1", dto.CreateDealRequest{RootItemID: "", ItemID: "", Quantity: 1})
	wantText := statusErrors.ErrBadRequest.Error() + ": item_id required"
	if err == nil || err.Error() != wantText {
		t.Errorf("error = %v, want %q (item_id check must take precedence over root_item_id)", err, wantText)
	}
}

func TestService_CreateDeal_Success(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{createFunc: func(_ context.Context, rootItemID, creatorID string, window time.Duration) (domain.Deal, error) {
		if rootItemID != "root" || creatorID != "actor-1" {
			t.Errorf("Create() called with rootItemID=%q creatorID=%q", rootItemID, creatorID)
		}
		if window != defaultNegotiationWindow {
			t.Errorf("Create() negotiation window = %v, want %v", window, defaultNegotiationWindow)
		}
		return domain.Deal{ID: "deal-1"}, nil
	}}
	proposals := &fakeProposalRepo{createFunc: func(_ context.Context, dealID, participantID, itemID string, quantity float64) (domain.Proposal, error) {
		if dealID != "deal-1" || participantID != "actor-1" || itemID != "x" || quantity != 2 {
			t.Errorf("proposals.Create() got dealID=%q participantID=%q itemID=%q quantity=%v", dealID, participantID, itemID, quantity)
		}
		return domain.Proposal{DealID: dealID, ParticipantID: participantID, ItemID: itemID, Quantity: quantity}, nil
	}}
	svc := newService(deals, proposals, nil)

	got, err := svc.CreateDeal(context.Background(), "actor-1", dto.CreateDealRequest{RootItemID: "root", ItemID: "x", Quantity: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.DealID != "deal-1" || got.ParticipantID != "actor-1" {
		t.Errorf("CreateDeal() = %+v", got)
	}
}

func TestService_CreateProposal_RepositoryErrorMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		repoErr     error
		wantErrIs   error
		wantErrText string
	}{
		{"item not found", proposalrepo.ErrItemNotFound, statusErrors.ErrNotFound, "item not found"},
		{"recipient not found", proposalrepo.ErrRecipientNotFound, statusErrors.ErrNotFound, "no one wishes for this item"},
		{"not item holder", proposalrepo.ErrNotItemHolder, statusErrors.ErrConflict, "you do not hold this item"},
		{"already proposed", proposalrepo.ErrAlreadyProposed, statusErrors.ErrConflict, "proposal already exists for this deal"},
		{"chain closed", proposalrepo.ErrChainClosed, statusErrors.ErrConflict, "chain is already closed to new participants"},
		{"item locked", proposalrepo.ErrItemLocked, statusErrors.ErrConflict, "item is locked by another confirmed deal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			deals := &fakeDealRepo{createFunc: func(context.Context, string, string, time.Duration) (domain.Deal, error) {
				return domain.Deal{ID: "deal-1"}, nil
			}}
			proposals := &fakeProposalRepo{createFunc: func(context.Context, string, string, string, float64) (domain.Proposal, error) {
				return domain.Proposal{}, tt.repoErr
			}}
			svc := newService(deals, proposals, nil)

			_, err := svc.CreateDeal(context.Background(), "actor-1", dto.CreateDealRequest{RootItemID: "root", ItemID: "x", Quantity: 1})
			if !errors.Is(err, tt.wantErrIs) {
				t.Errorf("error = %v, want wrapping %v", err, tt.wantErrIs)
			}
			wantText := tt.wantErrIs.Error() + ": " + tt.wantErrText
			if err.Error() != wantText {
				t.Errorf("error text = %q, want %q", err.Error(), wantText)
			}
		})
	}
}

func TestService_CreateProposal_UnmappedRepositoryErrorPassesThrough(t *testing.T) {
	t.Parallel()

	boom := errors.New("db down")
	deals := &fakeDealRepo{createFunc: func(context.Context, string, string, time.Duration) (domain.Deal, error) {
		return domain.Deal{ID: "deal-1"}, nil
	}}
	proposals := &fakeProposalRepo{createFunc: func(context.Context, string, string, string, float64) (domain.Proposal, error) {
		return domain.Proposal{}, boom
	}}
	svc := newService(deals, proposals, nil)

	_, err := svc.CreateDeal(context.Background(), "actor-1", dto.CreateDealRequest{RootItemID: "root", ItemID: "x", Quantity: 1})
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want wrapping %v", err, boom)
	}
}

func TestService_CreateProposal_ValidationFailsBeforeLocking(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{}
	svc := newService(deals, nil, nil)

	_, err := svc.CreateProposal(context.Background(), "actor-1", "deal-1", dto.CreateProposalRequest{ItemID: "", Quantity: 1})
	if !errors.Is(err, statusErrors.ErrBadRequest) {
		t.Errorf("error = %v, want wrapping ErrBadRequest", err)
	}
	if deals.lockDealCalled {
		t.Errorf("LockDeal was called despite a validation failure")
	}
}

func TestService_CreateProposal_LockDealError(t *testing.T) {
	t.Parallel()

	boom := errors.New("advisory lock unavailable")
	deals := &fakeDealRepo{lockErr: boom}
	svc := newService(deals, nil, nil)

	_, err := svc.CreateProposal(context.Background(), "actor-1", "deal-1", dto.CreateProposalRequest{ItemID: "x", Quantity: 1})
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want wrapping %v", err, boom)
	}
}

func TestService_CreateProposal_ReleasesLockEvenOnLaterError(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{}, dealrepo.ErrNotFound
	}}
	svc := newService(deals, nil, nil)

	ctx := t.Context()
	_, err := svc.CreateProposal(ctx, "actor-1", "deal-1", dto.CreateProposalRequest{ItemID: "x", Quantity: 1})
	if err == nil {
		t.Fatalf("expected an error")
	}
	if !deals.released {
		t.Errorf("LockDeal's release was never called despite a later error")
	}
	if deals.releaseCtx == nil || deals.releaseCtx.Done() != nil {
		t.Errorf("release was not called with context.WithoutCancel(ctx) — a canceled caller context must not abort the unlock")
	}
}

func TestService_CreateProposal_DealNotOpen(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{ID: "deal-1", Status: domain.DealStatusCompleted}, nil
	}}
	proposals := &fakeProposalRepo{}
	svc := newService(deals, proposals, nil)

	_, err := svc.CreateProposal(context.Background(), "actor-1", "deal-1", dto.CreateProposalRequest{ItemID: "x", Quantity: 1})
	if !errors.Is(err, statusErrors.ErrConflict) {
		t.Errorf("error = %v, want wrapping ErrConflict", err)
	}
	if err.Error() != "conflict: deal is not open" {
		t.Errorf("error text = %q", err.Error())
	}
}

func TestService_CreateProposal_DeadlineExpiredCascadesAndCancels(t *testing.T) {
	t.Parallel()

	past := time.Now().Add(-time.Hour)
	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{ID: "deal-1", Status: domain.DealStatusPending, DeadlineAt: &past}, nil
	}}
	proposals := &fakeProposalRepo{}
	svc := newService(deals, proposals, nil)

	_, err := svc.CreateProposal(context.Background(), "actor-1", "deal-1", dto.CreateProposalRequest{ItemID: "x", Quantity: 1})
	if !errors.Is(err, statusErrors.ErrConflict) || err.Error() != "conflict: deal deadline has passed" {
		t.Errorf("error = %v, want conflict: deal deadline has passed", err)
	}
	if len(deals.updateStatusCalls) != 1 || deals.updateStatusCalls[0] != domain.DealStatusCanceled {
		t.Errorf("UpdateStatus calls = %v, want exactly one DealStatusCanceled", deals.updateStatusCalls)
	}
	if len(proposals.declineAllForDealCalls) != 1 || proposals.declineAllForDealCalls[0] != "deal-1" {
		t.Errorf("DeclineAllForDeal calls = %v", proposals.declineAllForDealCalls)
	}
	if len(proposals.unlockAllForDealCalls) != 1 || proposals.unlockAllForDealCalls[0] != "deal-1" {
		t.Errorf("UnlockAllForDeal calls = %v", proposals.unlockAllForDealCalls)
	}
}

func TestService_CreateProposal_ChainAlreadyClosed(t *testing.T) {
	t.Parallel()

	future := time.Now().Add(time.Hour)
	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{ID: "deal-1", Status: domain.DealStatusPending, DeadlineAt: &future}, nil
	}}
	proposals := &fakeProposalRepo{}
	svc := newService(deals, proposals, nil)

	_, err := svc.CreateProposal(context.Background(), "actor-1", "deal-1", dto.CreateProposalRequest{ItemID: "x", Quantity: 1})
	if !errors.Is(err, statusErrors.ErrConflict) || err.Error() != "conflict: chain is already closed to new participants" {
		t.Errorf("error = %v, want conflict: chain is already closed to new participants", err)
	}
}

func TestService_CreateProposal_DuplicateProposalRejected(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{ID: "deal-1", Status: domain.DealStatusPending}, nil
	}}
	proposals := &fakeProposalRepo{
		getByDealAndParticipant: func(context.Context, string, string) (domain.Proposal, error) {
			return domain.Proposal{DealID: "deal-1"}, nil
		},
		createFunc: func(context.Context, string, string, string, float64) (domain.Proposal, error) {
			t.Fatalf("Create() must not be called for a duplicate proposal")
			return domain.Proposal{}, nil
		},
	}
	svc := newService(deals, proposals, nil)

	_, err := svc.CreateProposal(context.Background(), "actor-1", "deal-1", dto.CreateProposalRequest{ItemID: "x", Quantity: 1})
	if !errors.Is(err, statusErrors.ErrConflict) || err.Error() != "conflict: proposal already exists for this deal" {
		t.Errorf("error = %v", err)
	}
}

func TestService_CreateProposal_DuplicateCheckPropagatesUnrelatedError(t *testing.T) {
	t.Parallel()

	boom := errors.New("connection reset")
	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{ID: "deal-1", Status: domain.DealStatusPending}, nil
	}}
	proposals := &fakeProposalRepo{
		getByDealAndParticipant: func(context.Context, string, string) (domain.Proposal, error) {
			return domain.Proposal{}, boom
		},
		createFunc: func(context.Context, string, string, string, float64) (domain.Proposal, error) {
			t.Fatalf("Create() must not be called when the duplicate check itself failed")
			return domain.Proposal{}, nil
		},
	}
	svc := newService(deals, proposals, nil)

	_, err := svc.CreateProposal(context.Background(), "actor-1", "deal-1", dto.CreateProposalRequest{ItemID: "x", Quantity: 1})
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want wrapping %v", err, boom)
	}
}

func TestService_CreateProposal_Success(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{ID: "deal-1", Status: domain.DealStatusPending}, nil
	}}
	proposals := &fakeProposalRepo{
		getByDealAndParticipant: func(context.Context, string, string) (domain.Proposal, error) {
			return domain.Proposal{}, proposalrepo.ErrNotFound
		},
		createFunc: func(_ context.Context, dealID, participantID, itemID string, quantity float64) (domain.Proposal, error) {
			return domain.Proposal{DealID: dealID, ParticipantID: participantID, ItemID: itemID, Quantity: quantity}, nil
		},
	}
	svc := newService(deals, proposals, nil)

	got, err := svc.CreateProposal(context.Background(), "actor-1", "deal-1", dto.CreateProposalRequest{ItemID: "x", Quantity: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.DealID != "deal-1" || got.ParticipantID != "actor-1" || got.ItemID != "x" || got.Quantity != 3 {
		t.Errorf("CreateProposal() = %+v", got)
	}
	if !deals.released {
		t.Errorf("lock was not released")
	}
	if deals.lockDealID != "deal-1" {
		t.Errorf("LockDeal was called with dealID = %q, want %q", deals.lockDealID, "deal-1")
	}
}

func TestService_GetProposal(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		want := domain.Proposal{DealID: "deal-1", ParticipantID: "actor-1"}
		var gotDealID, gotParticipantID string
		proposals := &fakeProposalRepo{getByDealAndParticipant: func(_ context.Context, dealID, participantID string) (domain.Proposal, error) {
			gotDealID, gotParticipantID = dealID, participantID
			return want, nil
		}}
		svc := newService(nil, proposals, nil)

		got, err := svc.GetProposal(context.Background(), "actor-1", "deal-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("GetProposal() = %+v, want %+v", got, want)
		}
		if gotDealID != "deal-1" || gotParticipantID != "actor-1" {
			t.Errorf("GetByDealAndParticipant() got dealID=%q participantID=%q, want deal-1/actor-1 (args must not be swapped)", gotDealID, gotParticipantID)
		}
	})

	t.Run("not found is wrapped", func(t *testing.T) {
		t.Parallel()
		proposals := &fakeProposalRepo{getByDealAndParticipant: func(context.Context, string, string) (domain.Proposal, error) {
			return domain.Proposal{}, proposalrepo.ErrNotFound
		}}
		svc := newService(nil, proposals, nil)

		_, err := svc.GetProposal(context.Background(), "actor-1", "deal-1")
		if !errors.Is(err, statusErrors.ErrNotFound) {
			t.Errorf("error = %v, want wrapping ErrNotFound", err)
		}
	})

	t.Run("unmapped error passes through", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("db down")
		proposals := &fakeProposalRepo{getByDealAndParticipant: func(context.Context, string, string) (domain.Proposal, error) {
			return domain.Proposal{}, boom
		}}
		svc := newService(nil, proposals, nil)

		_, err := svc.GetProposal(context.Background(), "actor-1", "deal-1")
		if !errors.Is(err, boom) {
			t.Errorf("error = %v, want wrapping %v", err, boom)
		}
	})
}

func TestService_UpdateProposal_QuantityValidation(t *testing.T) {
	t.Parallel()

	bad := 0.0
	proposals := &fakeProposalRepo{updateFunc: func(context.Context, string, string, domain.ProposalUpdate) (domain.Proposal, error) {
		t.Fatalf("repository must not be called")
		return domain.Proposal{}, nil
	}}
	svc := newService(nil, proposals, nil)

	_, err := svc.UpdateProposal(context.Background(), "actor-1", "deal-1", domain.ProposalUpdate{Quantity: &bad})
	if !errors.Is(err, statusErrors.ErrBadRequest) {
		t.Errorf("error = %v, want wrapping ErrBadRequest", err)
	}
}

func TestService_UpdateProposal_RepositoryErrorMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		repoErr     error
		wantErrIs   error
		wantErrText string
	}{
		{"not found", proposalrepo.ErrNotFound, statusErrors.ErrNotFound, "proposal not found"},
		{"item not found", proposalrepo.ErrItemNotFound, statusErrors.ErrNotFound, "item not found"},
		{"recipient not found", proposalrepo.ErrRecipientNotFound, statusErrors.ErrNotFound, "no one wishes for this item"},
		{"not item holder", proposalrepo.ErrNotItemHolder, statusErrors.ErrConflict, "you do not hold this item"},
		{"not pending", proposalrepo.ErrNotPending, statusErrors.ErrConflict, "proposal is not pending"},
		{"item locked", proposalrepo.ErrItemLocked, statusErrors.ErrConflict, "item is locked by another confirmed deal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			proposals := &fakeProposalRepo{updateFunc: func(context.Context, string, string, domain.ProposalUpdate) (domain.Proposal, error) {
				return domain.Proposal{}, tt.repoErr
			}}
			svc := newService(nil, proposals, nil)

			_, err := svc.UpdateProposal(context.Background(), "actor-1", "deal-1", domain.ProposalUpdate{})
			if !errors.Is(err, tt.wantErrIs) {
				t.Errorf("error = %v, want wrapping %v", err, tt.wantErrIs)
			}
			wantText := tt.wantErrIs.Error() + ": " + tt.wantErrText
			if err.Error() != wantText {
				t.Errorf("error text = %q, want %q", err.Error(), wantText)
			}
		})
	}
}

func TestService_UpdateProposal_UnmappedRepositoryErrorPassesThrough(t *testing.T) {
	t.Parallel()

	boom := errors.New("db down")
	proposals := &fakeProposalRepo{updateFunc: func(context.Context, string, string, domain.ProposalUpdate) (domain.Proposal, error) {
		return domain.Proposal{}, boom
	}}
	svc := newService(nil, proposals, nil)

	_, err := svc.UpdateProposal(context.Background(), "actor-1", "deal-1", domain.ProposalUpdate{})
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want wrapping %v", err, boom)
	}
}

func TestService_UpdateProposal_QuantityNilSkipsValidation(t *testing.T) {
	t.Parallel()

	newItem := "new-item"
	proposals := &fakeProposalRepo{updateFunc: func(_ context.Context, _, _ string, upd domain.ProposalUpdate) (domain.Proposal, error) {
		if upd.Quantity != nil {
			t.Errorf("Update() received Quantity = %v, want nil", *upd.Quantity)
		}
		return domain.Proposal{ItemID: *upd.ItemID}, nil
	}}
	svc := newService(nil, proposals, nil)

	got, err := svc.UpdateProposal(context.Background(), "actor-1", "deal-1", domain.ProposalUpdate{ItemID: &newItem, Quantity: nil})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ItemID != newItem {
		t.Errorf("UpdateProposal() = %+v", got)
	}
}

func TestService_UpdateProposal_Success(t *testing.T) {
	t.Parallel()

	newItem := "new-item"
	newQty := 5.0
	proposals := &fakeProposalRepo{updateFunc: func(_ context.Context, dealID, participantID string, upd domain.ProposalUpdate) (domain.Proposal, error) {
		if dealID != "deal-1" || participantID != "actor-1" || upd.ItemID == nil || *upd.ItemID != newItem || upd.Quantity == nil || *upd.Quantity != newQty {
			t.Errorf("Update() got dealID=%q participantID=%q upd=%+v", dealID, participantID, upd)
		}
		return domain.Proposal{DealID: dealID, ParticipantID: participantID, ItemID: *upd.ItemID, Quantity: *upd.Quantity}, nil
	}}
	svc := newService(nil, proposals, nil)

	got, err := svc.UpdateProposal(context.Background(), "actor-1", "deal-1", domain.ProposalUpdate{ItemID: &newItem, Quantity: &newQty})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ItemID != newItem || got.Quantity != newQty {
		t.Errorf("UpdateProposal() = %+v", got)
	}
}

func TestService_WithdrawProposal_LockDealError(t *testing.T) {
	t.Parallel()

	boom := errors.New("advisory lock unavailable")
	deals := &fakeDealRepo{lockErr: boom}
	svc := newService(deals, nil, nil)

	err := svc.WithdrawProposal(context.Background(), "actor-1", "deal-1")
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want wrapping %v", err, boom)
	}
}

func TestService_WithdrawProposal_ReleasesLockOnLaterError(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{}, dealrepo.ErrNotFound
	}}
	svc := newService(deals, nil, nil)

	ctx := t.Context()
	_ = svc.WithdrawProposal(ctx, "actor-1", "deal-1")
	if !deals.released {
		t.Errorf("lock was not released despite a later error")
	}
	if deals.releaseCtx == nil || deals.releaseCtx.Done() != nil {
		t.Errorf("release was not called with context.WithoutCancel(ctx) — a canceled caller context must not abort the unlock")
	}
}

func TestService_WithdrawProposal_DealNotOpen(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{Status: domain.DealStatusCompleted}, nil
	}}
	proposals := &fakeProposalRepo{setStatusFunc: func(context.Context, string, string, domain.ProposalStatus) (domain.Proposal, error) {
		t.Fatalf("SetStatus must not be called when the deal is not open")
		return domain.Proposal{}, nil
	}}
	svc := newService(deals, proposals, nil)

	err := svc.WithdrawProposal(context.Background(), "actor-1", "deal-1")
	if !errors.Is(err, statusErrors.ErrConflict) || err.Error() != "conflict: deal is not open" {
		t.Errorf("error = %v", err)
	}
}

func TestService_WithdrawProposal_SetStatusErrorMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		repoErr     error
		wantErrIs   error
		wantErrText string
	}{
		{"not found", proposalrepo.ErrNotFound, statusErrors.ErrNotFound, "proposal not found"},
		{"not pending", proposalrepo.ErrNotPending, statusErrors.ErrConflict, "proposal is not pending"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
				return domain.Deal{Status: domain.DealStatusPending}, nil
			}}
			proposals := &fakeProposalRepo{setStatusFunc: func(context.Context, string, string, domain.ProposalStatus) (domain.Proposal, error) {
				return domain.Proposal{}, tt.repoErr
			}}
			svc := newService(deals, proposals, nil)

			err := svc.WithdrawProposal(context.Background(), "actor-1", "deal-1")
			if !errors.Is(err, tt.wantErrIs) {
				t.Errorf("error = %v, want wrapping %v", err, tt.wantErrIs)
			}
			if len(deals.updateStatusCalls) != 0 {
				t.Errorf("UpdateStatus must not be called after a SetStatus failure")
			}
		})
	}
}

func TestService_WithdrawProposal_SetStatusUnmappedErrorPassesThrough(t *testing.T) {
	t.Parallel()

	boom := errors.New("db down")
	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{Status: domain.DealStatusPending}, nil
	}}
	proposals := &fakeProposalRepo{setStatusFunc: func(context.Context, string, string, domain.ProposalStatus) (domain.Proposal, error) {
		return domain.Proposal{}, boom
	}}
	svc := newService(deals, proposals, nil)

	err := svc.WithdrawProposal(context.Background(), "actor-1", "deal-1")
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want wrapping %v", err, boom)
	}
}

func TestService_WithdrawProposal_CascadeStopsOnFirstFailure(t *testing.T) {
	t.Parallel()

	t.Run("UpdateStatus fails, cascade stops there", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("db down")
		deals := &fakeDealRepo{
			getByIDFunc: func(context.Context, string) (domain.Deal, error) {
				return domain.Deal{Status: domain.DealStatusPending}, nil
			},
			updateStatusFunc: func(context.Context, string, domain.DealStatus) (domain.Deal, error) { return domain.Deal{}, boom },
		}
		proposals := &fakeProposalRepo{}
		svc := newService(deals, proposals, nil)

		err := svc.WithdrawProposal(context.Background(), "actor-1", "deal-1")
		if !errors.Is(err, boom) {
			t.Errorf("error = %v, want wrapping %v", err, boom)
		}
		if len(proposals.declineAllExceptCalls) != 0 || len(proposals.unlockAllForDealCalls) != 0 {
			t.Errorf("cascade continued past the UpdateStatus failure")
		}
	})

	t.Run("DeclineAllExcept fails, UnlockAllForDeal is skipped", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("db down")
		deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
			return domain.Deal{Status: domain.DealStatusPending}, nil
		}}
		proposals := &fakeProposalRepo{declineAllExceptErr: boom}
		svc := newService(deals, proposals, nil)

		err := svc.WithdrawProposal(context.Background(), "actor-1", "deal-1")
		if !errors.Is(err, boom) {
			t.Errorf("error = %v, want wrapping %v", err, boom)
		}
		if len(proposals.unlockAllForDealCalls) != 0 {
			t.Errorf("UnlockAllForDeal must not be called after a DeclineAllExcept failure")
		}
	})

	t.Run("UnlockAllForDeal fails, error propagates", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("db down")
		deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
			return domain.Deal{Status: domain.DealStatusPending}, nil
		}}
		proposals := &fakeProposalRepo{unlockAllForDealErr: boom}
		svc := newService(deals, proposals, nil)

		err := svc.WithdrawProposal(context.Background(), "actor-1", "deal-1")
		if !errors.Is(err, boom) {
			t.Errorf("error = %v, want wrapping %v", err, boom)
		}
		if len(proposals.declineAllExceptCalls) != 1 {
			t.Errorf("DeclineAllExcept calls = %d, want 1 (must have run before the failing UnlockAllForDeal)", len(proposals.declineAllExceptCalls))
		}
	})
}

func TestService_WithdrawProposal_Success(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{Status: domain.DealStatusPending}, nil
	}}
	proposals := &fakeProposalRepo{setStatusFunc: func(_ context.Context, dealID, participantID string, status domain.ProposalStatus) (domain.Proposal, error) {
		if status != domain.ProposalStatusDeclined {
			t.Errorf("SetStatus status = %v, want Declined", status)
		}
		return domain.Proposal{DealID: dealID, ParticipantID: participantID}, nil
	}}
	svc := newService(deals, proposals, nil)

	if err := svc.WithdrawProposal(context.Background(), "actor-1", "deal-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deals.updateStatusCalls) != 1 || deals.updateStatusCalls[0] != domain.DealStatusCanceled {
		t.Errorf("UpdateStatus calls = %v, want exactly one Canceled", deals.updateStatusCalls)
	}
	if len(proposals.declineAllExceptCalls) != 1 || proposals.declineAllExceptCalls[0] != [2]string{"deal-1", "actor-1"} {
		t.Errorf("DeclineAllExcept calls = %v", proposals.declineAllExceptCalls)
	}
	if len(proposals.unlockAllForDealCalls) != 1 || proposals.unlockAllForDealCalls[0] != "deal-1" {
		t.Errorf("UnlockAllForDeal calls = %v", proposals.unlockAllForDealCalls)
	}
}

func TestService_ListForUser(t *testing.T) {
	t.Parallel()

	t.Run("actor requesting someone else's list is unauthorized", func(t *testing.T) {
		t.Parallel()
		proposals := &fakeProposalRepo{listForUserFunc: func(context.Context, string) ([]domain.Proposal, error) {
			t.Fatalf("repository must not be called")
			return nil, nil
		}}
		svc := newService(nil, proposals, nil)

		_, err := svc.ListForUser(context.Background(), "actor-1", "someone-else")
		if !errors.Is(err, statusErrors.ErrUnauthorized) {
			t.Errorf("error = %v, want ErrUnauthorized", err)
		}
		if err.Error() != "unauthorized" {
			t.Errorf("error text = %q, want the bare sentinel text (not wrapped)", err.Error())
		}
	})

	t.Run("own list success", func(t *testing.T) {
		t.Parallel()
		want := []domain.Proposal{{DealID: "d1"}, {DealID: "d2"}}
		proposals := &fakeProposalRepo{listForUserFunc: func(_ context.Context, userID string) ([]domain.Proposal, error) {
			if userID != "actor-1" {
				t.Errorf("ListForUser called with userID = %q", userID)
			}
			return want, nil
		}}
		svc := newService(nil, proposals, nil)

		got, err := svc.ListForUser(context.Background(), "actor-1", "actor-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != len(want) {
			t.Errorf("ListForUser() = %+v, want %+v", got, want)
		}
	})

	t.Run("repository error propagates", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("db down")
		proposals := &fakeProposalRepo{listForUserFunc: func(context.Context, string) ([]domain.Proposal, error) {
			return nil, boom
		}}
		svc := newService(nil, proposals, nil)

		_, err := svc.ListForUser(context.Background(), "actor-1", "actor-1")
		if !errors.Is(err, boom) {
			t.Errorf("error = %v, want wrapping %v", err, boom)
		}
	})
}

func TestService_Approve_LockDealError(t *testing.T) {
	t.Parallel()

	boom := errors.New("advisory lock unavailable")
	deals := &fakeDealRepo{lockErr: boom}
	svc := newService(deals, nil, nil)

	_, err := svc.Approve(context.Background(), "actor-1", "deal-1")
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want wrapping %v", err, boom)
	}
}

func TestService_Approve_ReleasesLockOnLaterError(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{}, dealrepo.ErrNotFound
	}}
	svc := newService(deals, nil, nil)

	ctx := t.Context()
	_, _ = svc.Approve(ctx, "actor-1", "deal-1")
	if !deals.released {
		t.Errorf("lock was not released despite a later error")
	}
	if deals.releaseCtx == nil || deals.releaseCtx.Done() != nil {
		t.Errorf("release was not called with context.WithoutCancel(ctx) — a canceled caller context must not abort the unlock")
	}
}

func TestService_Approve_DealNotOpen(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{Status: domain.DealStatusCanceled}, nil
	}}
	proposals := &fakeProposalRepo{setStatusFunc: func(context.Context, string, string, domain.ProposalStatus) (domain.Proposal, error) {
		t.Fatalf("SetStatus must not be called when the deal is not open")
		return domain.Proposal{}, nil
	}}
	svc := newService(deals, proposals, nil)

	_, err := svc.Approve(context.Background(), "actor-1", "deal-1")
	if !errors.Is(err, statusErrors.ErrConflict) || err.Error() != "conflict: deal is not open" {
		t.Errorf("error = %v", err)
	}
}

func TestService_Approve_SetStatusErrorMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		repoErr     error
		wantErrIs   error
		wantErrText string
	}{
		{"not found", proposalrepo.ErrNotFound, statusErrors.ErrNotFound, "proposal not found"},
		{"not pending", proposalrepo.ErrNotPending, statusErrors.ErrConflict, "proposal is not pending"},
		{"out of order", proposalrepo.ErrOutOfOrder, statusErrors.ErrConflict, "your recipient has not accepted yet"},
		{"not item holder", proposalrepo.ErrNotItemHolder, statusErrors.ErrConflict, "you no longer hold the item you offered"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
				return domain.Deal{Status: domain.DealStatusPending}, nil
			}}
			proposals := &fakeProposalRepo{
				setStatusFunc: func(context.Context, string, string, domain.ProposalStatus) (domain.Proposal, error) {
					return domain.Proposal{}, tt.repoErr
				},
				tryLockChainFunc: func(context.Context, string) error {
					t.Fatalf("TryLockChain must not be called after a SetStatus failure")
					return nil
				},
			}
			svc := newService(deals, proposals, nil)

			_, err := svc.Approve(context.Background(), "actor-1", "deal-1")
			if !errors.Is(err, tt.wantErrIs) {
				t.Errorf("error = %v, want wrapping %v", err, tt.wantErrIs)
			}
			wantText := tt.wantErrIs.Error() + ": " + tt.wantErrText
			if err.Error() != wantText {
				t.Errorf("error text = %q, want %q", err.Error(), wantText)
			}
		})
	}
}

func TestService_Approve_SetStatusRetriesOnDeadlockThenSucceeds(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{Status: domain.DealStatusPending}, nil
	}}
	attempts := 0
	proposals := &fakeProposalRepo{
		setStatusFunc: func(context.Context, string, string, domain.ProposalStatus) (domain.Proposal, error) {
			attempts++
			if attempts < 3 {
				return domain.Proposal{}, deadlockErr()
			}
			return domain.Proposal{DealID: "deal-1", ParticipantID: "actor-1"}, nil
		},
	}
	svc := newService(deals, proposals, nil)

	got, err := svc.Approve(context.Background(), "actor-1", "deal-1")
	if err != nil {
		t.Fatalf("unexpected error after retrying past transient deadlocks: %v", err)
	}
	if attempts != 3 {
		t.Errorf("SetStatus was attempted %d times, want 3", attempts)
	}
	if got.DealID != "deal-1" {
		t.Errorf("Approve() = %+v", got)
	}
}

func TestService_Approve_SetStatusDeadlockExhaustsRetryBudget(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{Status: domain.DealStatusPending}, nil
	}}
	attempts := 0
	proposals := &fakeProposalRepo{
		setStatusFunc: func(context.Context, string, string, domain.ProposalStatus) (domain.Proposal, error) {
			attempts++
			return domain.Proposal{}, deadlockErr()
		},
		tryLockChainFunc: func(context.Context, string) error {
			t.Fatalf("TryLockChain must not be called when SetStatus never recovers")
			return nil
		},
	}
	svc := newService(deals, proposals, nil)

	_, err := svc.Approve(context.Background(), "actor-1", "deal-1")
	if !isDeadlock(err) {
		t.Errorf("error = %v, want a deadlock error to surface once the retry budget is exhausted", err)
	}
	if attempts != maxDeadlockRetries+1 {
		t.Errorf("SetStatus was attempted %d times, want %d", attempts, maxDeadlockRetries+1)
	}
}

func TestService_Approve_TryLockChainRetriesOnDeadlockThenSucceeds(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{Status: domain.DealStatusPending}, nil
	}}
	attempts := 0
	proposals := &fakeProposalRepo{
		setStatusFunc: func(context.Context, string, string, domain.ProposalStatus) (domain.Proposal, error) {
			return domain.Proposal{DealID: "deal-1"}, nil
		},
		tryLockChainFunc: func(context.Context, string) error {
			attempts++
			if attempts < 3 {
				return deadlockErr()
			}
			return nil
		},
		allAcceptedFunc: func(context.Context, string) (bool, error) { return false, nil },
	}
	svc := newService(deals, proposals, nil)

	_, err := svc.Approve(context.Background(), "actor-1", "deal-1")
	if err != nil {
		t.Fatalf("unexpected error after retrying past transient deadlocks: %v", err)
	}
	if attempts != 3 {
		t.Errorf("TryLockChain was attempted %d times, want 3", attempts)
	}
}

func TestService_Approve_TryLockChainErrorPropagates(t *testing.T) {
	t.Parallel()

	boom := errors.New("db down")
	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{Status: domain.DealStatusPending}, nil
	}}
	proposals := &fakeProposalRepo{
		setStatusFunc: func(context.Context, string, string, domain.ProposalStatus) (domain.Proposal, error) {
			return domain.Proposal{}, nil
		},
		tryLockChainFunc: func(context.Context, string) error { return boom },
		allAcceptedFunc: func(context.Context, string) (bool, error) {
			t.Fatalf("AllAccepted must not be called after a TryLockChain failure")
			return false, nil
		},
	}
	svc := newService(deals, proposals, nil)

	_, err := svc.Approve(context.Background(), "actor-1", "deal-1")
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want wrapping %v", err, boom)
	}
}

func TestService_Approve_NotAllAcceptedStaysOpen(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{
		getByIDFunc: func(context.Context, string) (domain.Deal, error) {
			return domain.Deal{Status: domain.DealStatusPending}, nil
		},
		updateStatusFunc: func(context.Context, string, domain.DealStatus) (domain.Deal, error) {
			t.Fatalf("UpdateStatus must not be called while the chain is not fully accepted")
			return domain.Deal{}, nil
		},
	}
	proposals := &fakeProposalRepo{
		setStatusFunc: func(context.Context, string, string, domain.ProposalStatus) (domain.Proposal, error) {
			return domain.Proposal{DealID: "deal-1", ParticipantID: "actor-1"}, nil
		},
		allAcceptedFunc: func(context.Context, string) (bool, error) { return false, nil },
		listTransfersFunc: func(context.Context, string) ([]domain.ItemTransfer, error) {
			t.Fatalf("ListTransfers must not be called while the chain is not fully accepted")
			return nil, nil
		},
	}
	svc := newService(deals, proposals, nil)

	got, err := svc.Approve(context.Background(), "actor-1", "deal-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.DealID != "deal-1" {
		t.Errorf("Approve() = %+v", got)
	}
}

func TestService_Approve_AllAcceptedErrorPropagates(t *testing.T) {
	t.Parallel()

	boom := errors.New("db down")
	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{Status: domain.DealStatusPending}, nil
	}}
	proposals := &fakeProposalRepo{
		setStatusFunc: func(context.Context, string, string, domain.ProposalStatus) (domain.Proposal, error) {
			return domain.Proposal{}, nil
		},
		allAcceptedFunc: func(context.Context, string) (bool, error) { return false, boom },
	}
	svc := newService(deals, proposals, nil)

	_, err := svc.Approve(context.Background(), "actor-1", "deal-1")
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want wrapping %v", err, boom)
	}
}

func TestService_Approve_CompletionCascade_ConfirmFails(t *testing.T) {
	t.Parallel()

	boom := errors.New("db down")
	deals := &fakeDealRepo{
		getByIDFunc: func(context.Context, string) (domain.Deal, error) {
			return domain.Deal{Status: domain.DealStatusPending}, nil
		},
		updateStatusFunc: func(_ context.Context, _ string, status domain.DealStatus) (domain.Deal, error) {
			if status == domain.DealStatusConfirmed {
				return domain.Deal{}, boom
			}
			return domain.Deal{}, nil
		},
	}
	proposals := &fakeProposalRepo{
		setStatusFunc: func(context.Context, string, string, domain.ProposalStatus) (domain.Proposal, error) {
			return domain.Proposal{}, nil
		},
		allAcceptedFunc: func(context.Context, string) (bool, error) { return true, nil },
		listTransfersFunc: func(context.Context, string) ([]domain.ItemTransfer, error) {
			t.Fatalf("ListTransfers must not be called when confirming the deal already failed")
			return nil, nil
		},
	}
	svc := newService(deals, proposals, nil)

	_, err := svc.Approve(context.Background(), "actor-1", "deal-1")
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want wrapping %v", err, boom)
	}
}

func TestService_Approve_CompletionCascade_ListTransfersFails(t *testing.T) {
	t.Parallel()

	boom := errors.New("db down")
	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{Status: domain.DealStatusPending}, nil
	}}
	chown := &fakeChownRepo{}
	proposals := &fakeProposalRepo{
		setStatusFunc: func(context.Context, string, string, domain.ProposalStatus) (domain.Proposal, error) {
			return domain.Proposal{}, nil
		},
		allAcceptedFunc:   func(context.Context, string) (bool, error) { return true, nil },
		listTransfersFunc: func(context.Context, string) ([]domain.ItemTransfer, error) { return nil, boom },
	}
	svc := newService(deals, proposals, chown)

	_, err := svc.Approve(context.Background(), "actor-1", "deal-1")
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want wrapping %v", err, boom)
	}
	if len(chown.calls) != 0 {
		t.Errorf("CompleteTransfer must not be called when ListTransfers failed")
	}
	if len(deals.updateStatusCalls) != 1 {
		t.Errorf("UpdateStatus calls = %v, want exactly the Confirmed call (no Completed)", deals.updateStatusCalls)
	}
}

func TestService_Approve_CompletionCascade_StopsOnFirstFailedTransfer(t *testing.T) {
	t.Parallel()

	boom := errors.New("item vanished")
	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{Status: domain.DealStatusPending}, nil
	}}
	chown := &fakeChownRepo{completeTransferFunc: func(_ context.Context, itemID, _ string) error {
		if itemID == "item-2" {
			return boom
		}
		return nil
	}}
	proposals := &fakeProposalRepo{
		setStatusFunc: func(context.Context, string, string, domain.ProposalStatus) (domain.Proposal, error) {
			return domain.Proposal{}, nil
		},
		allAcceptedFunc: func(context.Context, string) (bool, error) { return true, nil },
		listTransfersFunc: func(context.Context, string) ([]domain.ItemTransfer, error) {
			return []domain.ItemTransfer{
				{ItemID: "item-1", ToUserID: "user-1"},
				{ItemID: "item-2", ToUserID: "user-2"},
				{ItemID: "item-3", ToUserID: "user-3"},
			}, nil
		},
	}
	svc := newService(deals, proposals, chown)

	_, err := svc.Approve(context.Background(), "actor-1", "deal-1")
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want wrapping %v", err, boom)
	}
	if len(chown.calls) != 2 {
		t.Errorf("CompleteTransfer was called %d times, want exactly 2 (stopping at the failing transfer)", len(chown.calls))
	}
	if len(deals.updateStatusCalls) != 1 {
		t.Errorf("UpdateStatus calls = %v, want only the Confirmed call, not Completed", deals.updateStatusCalls)
	}
}

func TestService_Approve_CompletionCascade_FinalCompletedUpdateFails(t *testing.T) {
	t.Parallel()

	boom := errors.New("db down")
	deals := &fakeDealRepo{
		getByIDFunc: func(context.Context, string) (domain.Deal, error) {
			return domain.Deal{Status: domain.DealStatusPending}, nil
		},
		updateStatusFunc: func(_ context.Context, _ string, status domain.DealStatus) (domain.Deal, error) {
			if status == domain.DealStatusCompleted {
				return domain.Deal{}, boom
			}
			return domain.Deal{}, nil
		},
	}
	chown := &fakeChownRepo{}
	proposals := &fakeProposalRepo{
		setStatusFunc: func(context.Context, string, string, domain.ProposalStatus) (domain.Proposal, error) {
			return domain.Proposal{}, nil
		},
		allAcceptedFunc: func(context.Context, string) (bool, error) { return true, nil },
		listTransfersFunc: func(context.Context, string) ([]domain.ItemTransfer, error) {
			return []domain.ItemTransfer{{ItemID: "item-1", ToUserID: "user-1"}}, nil
		},
	}
	svc := newService(deals, proposals, chown)

	_, err := svc.Approve(context.Background(), "actor-1", "deal-1")
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want wrapping %v", err, boom)
	}
	if len(chown.calls) != 1 {
		t.Errorf("CompleteTransfer calls = %d, want 1 (transfers themselves must have already succeeded)", len(chown.calls))
	}
	wantStatuses := []domain.DealStatus{domain.DealStatusConfirmed, domain.DealStatusCompleted}
	if len(deals.updateStatusCalls) != len(wantStatuses) {
		t.Fatalf("UpdateStatus calls = %v, want %v (both attempted, the second one failing)", deals.updateStatusCalls, wantStatuses)
	}
}

func TestService_Approve_CompletionCascade_FullSuccess(t *testing.T) {
	t.Parallel()

	deals := &fakeDealRepo{getByIDFunc: func(context.Context, string) (domain.Deal, error) {
		return domain.Deal{Status: domain.DealStatusPending}, nil
	}}
	chown := &fakeChownRepo{}
	proposals := &fakeProposalRepo{
		setStatusFunc: func(context.Context, string, string, domain.ProposalStatus) (domain.Proposal, error) {
			return domain.Proposal{DealID: "deal-1", ParticipantID: "actor-1", Status: domain.ProposalStatusAccepted}, nil
		},
		allAcceptedFunc: func(context.Context, string) (bool, error) { return true, nil },
		listTransfersFunc: func(context.Context, string) ([]domain.ItemTransfer, error) {
			return []domain.ItemTransfer{
				{ItemID: "item-1", ToUserID: "user-1"},
				{ItemID: "item-2", ToUserID: "user-2"},
			}, nil
		},
	}
	svc := newService(deals, proposals, chown)

	got, err := svc.Approve(context.Background(), "actor-1", "deal-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != domain.ProposalStatusAccepted {
		t.Errorf("Approve() returned %+v, want the proposal from SetStatus (not re-fetched)", got)
	}
	want := []domain.ItemTransfer{{ItemID: "item-1", ToUserID: "user-1"}, {ItemID: "item-2", ToUserID: "user-2"}}
	if len(chown.calls) != len(want) {
		t.Fatalf("CompleteTransfer calls = %+v, want %+v", chown.calls, want)
	}
	for i := range want {
		if chown.calls[i] != want[i] {
			t.Errorf("CompleteTransfer call[%d] = %+v, want %+v", i, chown.calls[i], want[i])
		}
	}
	wantStatuses := []domain.DealStatus{domain.DealStatusConfirmed, domain.DealStatusCompleted}
	if len(deals.updateStatusCalls) != len(wantStatuses) {
		t.Fatalf("UpdateStatus calls = %v, want %v", deals.updateStatusCalls, wantStatuses)
	}
	for i := range wantStatuses {
		if deals.updateStatusCalls[i] != wantStatuses[i] {
			t.Errorf("UpdateStatus call[%d] = %v, want %v", i, deals.updateStatusCalls[i], wantStatuses[i])
		}
	}
}

func TestEnsureDealOpen(t *testing.T) {
	t.Parallel()

	t.Run("not pending is rejected without side effects", func(t *testing.T) {
		t.Parallel()
		deals := &fakeDealRepo{}
		proposals := &fakeProposalRepo{}
		svc := newService(deals, proposals, nil)

		err := svc.ensureDealOpen(context.Background(), &domain.Deal{ID: "deal-1", Status: domain.DealStatusConfirmed})
		if !errors.Is(err, statusErrors.ErrConflict) || err.Error() != "conflict: deal is not open" {
			t.Errorf("error = %v", err)
		}
		if len(deals.updateStatusCalls) != 0 || len(proposals.declineAllForDealCalls) != 0 || len(proposals.unlockAllForDealCalls) != 0 {
			t.Errorf("side effects fired for a simple not-open deal")
		}
	})

	t.Run("pending with no deadline is open", func(t *testing.T) {
		t.Parallel()
		svc := newService(nil, nil, nil)
		err := svc.ensureDealOpen(context.Background(), &domain.Deal{ID: "deal-1", Status: domain.DealStatusPending, DeadlineAt: nil})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("pending with a future deadline is open", func(t *testing.T) {
		t.Parallel()
		future := time.Now().Add(time.Hour)
		svc := newService(nil, nil, nil)
		err := svc.ensureDealOpen(context.Background(), &domain.Deal{ID: "deal-1", Status: domain.DealStatusPending, DeadlineAt: &future})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("pending with an expired deadline cascades to canceled", func(t *testing.T) {
		t.Parallel()
		past := time.Now().Add(-time.Nanosecond)
		deals := &fakeDealRepo{}
		proposals := &fakeProposalRepo{}
		svc := newService(deals, proposals, nil)

		err := svc.ensureDealOpen(context.Background(), &domain.Deal{ID: "deal-1", Status: domain.DealStatusPending, DeadlineAt: &past})
		if !errors.Is(err, statusErrors.ErrConflict) || err.Error() != "conflict: deal deadline has passed" {
			t.Errorf("error = %v", err)
		}
		if len(deals.updateStatusCalls) != 1 || deals.updateStatusCalls[0] != domain.DealStatusCanceled {
			t.Errorf("UpdateStatus calls = %v", deals.updateStatusCalls)
		}
		if len(proposals.declineAllForDealCalls) != 1 || proposals.declineAllForDealCalls[0] != "deal-1" {
			t.Errorf("DeclineAllForDeal calls = %v", proposals.declineAllForDealCalls)
		}
		if len(proposals.unlockAllForDealCalls) != 1 || proposals.unlockAllForDealCalls[0] != "deal-1" {
			t.Errorf("UnlockAllForDeal calls = %v", proposals.unlockAllForDealCalls)
		}
	})

	t.Run("cascade stops at the first failure: UpdateStatus", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("db down")
		past := time.Now().Add(-time.Hour)
		deals := &fakeDealRepo{updateStatusFunc: func(context.Context, string, domain.DealStatus) (domain.Deal, error) { return domain.Deal{}, boom }}
		proposals := &fakeProposalRepo{}
		svc := newService(deals, proposals, nil)

		err := svc.ensureDealOpen(context.Background(), &domain.Deal{ID: "deal-1", Status: domain.DealStatusPending, DeadlineAt: &past})
		if !errors.Is(err, boom) {
			t.Errorf("error = %v, want wrapping %v", err, boom)
		}
		if len(proposals.declineAllForDealCalls) != 0 || len(proposals.unlockAllForDealCalls) != 0 {
			t.Errorf("cascade continued past the UpdateStatus failure")
		}
	})

	t.Run("cascade stops at the first failure: DeclineAllForDeal", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("db down")
		past := time.Now().Add(-time.Hour)
		deals := &fakeDealRepo{}
		proposals := &fakeProposalRepo{declineAllForDealErr: boom}
		svc := newService(deals, proposals, nil)

		err := svc.ensureDealOpen(context.Background(), &domain.Deal{ID: "deal-1", Status: domain.DealStatusPending, DeadlineAt: &past})
		if !errors.Is(err, boom) {
			t.Errorf("error = %v, want wrapping %v", err, boom)
		}
		if len(proposals.unlockAllForDealCalls) != 0 {
			t.Errorf("UnlockAllForDeal must not be called after a DeclineAllForDeal failure")
		}
	})

	t.Run("cascade stops at the first failure: UnlockAllForDeal", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("db down")
		past := time.Now().Add(-time.Hour)
		deals := &fakeDealRepo{}
		proposals := &fakeProposalRepo{unlockAllForDealErr: boom}
		svc := newService(deals, proposals, nil)

		err := svc.ensureDealOpen(context.Background(), &domain.Deal{ID: "deal-1", Status: domain.DealStatusPending, DeadlineAt: &past})
		if !errors.Is(err, boom) {
			t.Errorf("error = %v, want wrapping %v", err, boom)
		}
		if len(proposals.declineAllForDealCalls) != 1 {
			t.Errorf("DeclineAllForDeal calls = %d, want 1 (must have run before the failing UnlockAllForDeal)", len(proposals.declineAllForDealCalls))
		}
	})
}

func TestValidateOffer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		itemID   string
		quantity float64
		wantErr  string
	}{
		{name: "valid", itemID: "x", quantity: 1, wantErr: ""},
		{name: "empty item id", itemID: "", quantity: 1, wantErr: "bad request: item_id required"},
		{name: "zero quantity", itemID: "x", quantity: 0, wantErr: "bad request: quantity must be positive"},
		{name: "negative quantity", itemID: "x", quantity: -5, wantErr: "bad request: quantity must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateOffer(tt.itemID, tt.quantity)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("validateOffer() = %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("validateOffer() = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

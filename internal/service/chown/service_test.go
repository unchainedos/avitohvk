package chown

import (
	"context"
	"errors"
	"testing"
	"time"

	"avitohvk/internal/domain"
	"avitohvk/internal/dto"
	statusErrors "avitohvk/internal/errors"
)

type fakeRepo struct {
	ensureWishErr error

	ensureWishCalled  bool
	gotEnsureWishUser string
	gotEnsureWishItem string
}

func (f *fakeRepo) EnsureWish(_ context.Context, userID, itemID string) error {
	f.ensureWishCalled = true
	f.gotEnsureWishUser = userID
	f.gotEnsureWishItem = itemID
	return f.ensureWishErr
}

type fakeProposalService struct {
	findOpenDealID string
	findOpenFound  bool
	findOpenErr    error

	createDealResult domain.Proposal
	createDealErr    error

	createProposalResult domain.Proposal
	createProposalErr    error

	findOpenCalled     bool
	gotFindOpenItemID  string
	gotFindOpenActorID string

	createDealCalled     bool
	gotCtx               context.Context
	gotCreateDealActorID string
	gotCreateDealReq     dto.CreateDealRequest

	createProposalCalled     bool
	gotCreateProposalActorID string
	gotCreateProposalDealID  string
	gotCreateProposalReq     dto.CreateProposalRequest
}

func (f *fakeProposalService) FindOpenDealAsRecipient(ctx context.Context, itemID, participantID string) (string, bool, error) {
	f.findOpenCalled = true
	f.gotCtx = ctx
	f.gotFindOpenItemID = itemID
	f.gotFindOpenActorID = participantID
	return f.findOpenDealID, f.findOpenFound, f.findOpenErr
}

func (f *fakeProposalService) CreateDeal(ctx context.Context, actorID string, req dto.CreateDealRequest) (domain.Proposal, error) {
	f.createDealCalled = true
	f.gotCtx = ctx
	f.gotCreateDealActorID = actorID
	f.gotCreateDealReq = req
	return f.createDealResult, f.createDealErr
}

func (f *fakeProposalService) CreateProposal(ctx context.Context, actorID, dealID string, req dto.CreateProposalRequest) (domain.Proposal, error) {
	f.createProposalCalled = true
	f.gotCtx = ctx
	f.gotCreateProposalActorID = actorID
	f.gotCreateProposalDealID = dealID
	f.gotCreateProposalReq = req
	return f.createProposalResult, f.createProposalErr
}

func TestService_Chown_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		itemID string
		req    dto.CreateProposalRequest
	}{
		{name: "empty item_id", itemID: "", req: dto.CreateProposalRequest{ItemID: "offer-1", Quantity: 1}},
		{name: "empty offered item_id", itemID: "target", req: dto.CreateProposalRequest{ItemID: "", Quantity: 1}},
		{name: "zero quantity", itemID: "target", req: dto.CreateProposalRequest{ItemID: "offer-1", Quantity: 0}},
		{name: "negative quantity", itemID: "target", req: dto.CreateProposalRequest{ItemID: "offer-1", Quantity: -1}},
		{name: "offering the target item itself", itemID: "target", req: dto.CreateProposalRequest{ItemID: "target", Quantity: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeRepo{}
			fake := &fakeProposalService{}
			svc := NewService(repo, fake)

			_, err := svc.Chown(context.Background(), "actor-1", tt.itemID, tt.req)

			if !errors.Is(err, statusErrors.ErrBadRequest) {
				t.Errorf("Chown() error = %v, want wrapping ErrBadRequest", err)
			}
			if fake.findOpenCalled || fake.createDealCalled || fake.createProposalCalled || repo.ensureWishCalled {
				t.Errorf("Chown() called the proposal service or repository despite a validation failure")
			}
		})
	}
}

func TestService_Chown_CreatesNewDealWhenNoOpenDealFound(t *testing.T) {
	t.Parallel()

	want := domain.Proposal{DealID: "deal-new", ParticipantID: "actor-1", ItemID: "offer-1", Status: domain.ProposalStatusPending}
	repo := &fakeRepo{}
	fake := &fakeProposalService{findOpenFound: false, createDealResult: want}
	svc := NewService(repo, fake)

	got, err := svc.Chown(context.Background(), "actor-1", "target", dto.CreateProposalRequest{ItemID: "offer-1", Quantity: 2})
	if err != nil {
		t.Fatalf("Chown(): %v", err)
	}

	if !fake.findOpenCalled {
		t.Fatalf("FindOpenDealAsRecipient was not called")
	}
	if fake.gotFindOpenItemID != "target" || fake.gotFindOpenActorID != "actor-1" {
		t.Errorf("FindOpenDealAsRecipient called with (%q, %q), want (%q, %q)", fake.gotFindOpenItemID, fake.gotFindOpenActorID, "target", "actor-1")
	}
	if !fake.createDealCalled {
		t.Fatalf("CreateDeal was not called")
	}
	if fake.createProposalCalled {
		t.Errorf("CreateProposal was called, want only CreateDeal")
	}
	wantReq := dto.CreateDealRequest{RootItemID: "target", ItemID: "offer-1", Quantity: 2}
	if fake.gotCreateDealReq != wantReq {
		t.Errorf("CreateDeal called with %+v, want %+v", fake.gotCreateDealReq, wantReq)
	}
	if fake.gotCreateDealActorID != "actor-1" {
		t.Errorf("CreateDeal actorID = %q, want %q", fake.gotCreateDealActorID, "actor-1")
	}
	if got != want {
		t.Errorf("Chown() = %+v, want %+v", got, want)
	}

	if !repo.ensureWishCalled {
		t.Errorf("EnsureWish was not called after creating a new chain")
	}
	if repo.gotEnsureWishUser != "actor-1" || repo.gotEnsureWishItem != "target" {
		t.Errorf("EnsureWish called with (%q, %q), want (%q, %q)", repo.gotEnsureWishUser, repo.gotEnsureWishItem, "actor-1", "target")
	}
}

func TestService_Chown_JoinsExistingDealWhenFound(t *testing.T) {
	t.Parallel()

	want := domain.Proposal{DealID: "deal-existing", ParticipantID: "actor-1", ItemID: "offer-1", Status: domain.ProposalStatusPending}
	repo := &fakeRepo{}
	fake := &fakeProposalService{findOpenFound: true, findOpenDealID: "deal-existing", createProposalResult: want}
	svc := NewService(repo, fake)

	got, err := svc.Chown(context.Background(), "actor-1", "target", dto.CreateProposalRequest{ItemID: "offer-1", Quantity: 2})
	if err != nil {
		t.Fatalf("Chown(): %v", err)
	}

	if !fake.createProposalCalled {
		t.Fatalf("CreateProposal was not called")
	}
	if fake.createDealCalled {
		t.Errorf("CreateDeal was called, want only CreateProposal (joining an existing deal)")
	}
	if fake.gotCreateProposalActorID != "actor-1" || fake.gotCreateProposalDealID != "deal-existing" {
		t.Errorf("CreateProposal called with actor=%q deal=%q, want actor-1/deal-existing", fake.gotCreateProposalActorID, fake.gotCreateProposalDealID)
	}
	wantReq := dto.CreateProposalRequest{ItemID: "offer-1", Quantity: 2}
	if fake.gotCreateProposalReq != wantReq {
		t.Errorf("CreateProposal called with %+v, want %+v", fake.gotCreateProposalReq, wantReq)
	}
	if got != want {
		t.Errorf("Chown() = %+v, want %+v", got, want)
	}

	if repo.ensureWishCalled {
		t.Errorf("EnsureWish was called when joining an existing deal — the actor's wish must already exist to have been found as recipient")
	}
}

func TestService_Chown_ForwardsRequestContext(t *testing.T) {
	t.Parallel()

	type ctxKey struct{}
	repo := &fakeRepo{}
	fake := &fakeProposalService{}
	svc := NewService(repo, fake)

	ctx := context.WithValue(context.Background(), ctxKey{}, "trace-id-123")
	if _, err := svc.Chown(ctx, "actor-1", "target", dto.CreateProposalRequest{ItemID: "offer-1", Quantity: 1}); err != nil {
		t.Fatalf("Chown(): %v", err)
	}

	if fake.gotCtx == nil || fake.gotCtx.Value(ctxKey{}) != "trace-id-123" {
		t.Errorf("Chown() did not forward the caller's context unchanged")
	}
}

func TestService_Chown_FindOpenDealErrorPropagates(t *testing.T) {
	t.Parallel()

	boom := errors.New("db down")
	repo := &fakeRepo{}
	fake := &fakeProposalService{findOpenErr: boom}
	svc := NewService(repo, fake)

	_, err := svc.Chown(context.Background(), "actor-1", "target", dto.CreateProposalRequest{ItemID: "offer-1", Quantity: 1})
	if !errors.Is(err, boom) {
		t.Errorf("Chown() error = %v, want wrapping %v", err, boom)
	}
	if fake.createDealCalled || fake.createProposalCalled || repo.ensureWishCalled {
		t.Errorf("Chown() called CreateDeal/CreateProposal/EnsureWish despite a FindOpenDealAsRecipient error")
	}
}

func TestService_Chown_CreateDealErrorPropagatesUnwrapped(t *testing.T) {
	t.Parallel()

	sentinelErr := errors.New("conflict: root item not found")
	repo := &fakeRepo{}
	fake := &fakeProposalService{createDealErr: sentinelErr}
	svc := NewService(repo, fake)

	_, err := svc.Chown(context.Background(), "actor-1", "target", dto.CreateProposalRequest{ItemID: "offer-1", Quantity: 1})
	if !errors.Is(err, sentinelErr) {
		t.Errorf("Chown() error = %v, want exactly %v", err, sentinelErr)
	}
	if repo.ensureWishCalled {
		t.Errorf("EnsureWish was called even though CreateDeal failed")
	}
}

func TestService_Chown_CreateProposalErrorPropagatesUnwrapped(t *testing.T) {
	t.Parallel()

	sentinelErr := errors.New("conflict: proposal already exists for this deal")
	repo := &fakeRepo{}
	fake := &fakeProposalService{findOpenFound: true, findOpenDealID: "deal-1", createProposalErr: sentinelErr}
	svc := NewService(repo, fake)

	_, err := svc.Chown(context.Background(), "actor-1", "target", dto.CreateProposalRequest{ItemID: "offer-1", Quantity: 1})
	if !errors.Is(err, sentinelErr) {
		t.Errorf("Chown() error = %v, want exactly %v", err, sentinelErr)
	}
}

func TestService_Chown_EnsureWishErrorPropagates(t *testing.T) {
	t.Parallel()

	boom := errors.New("db down")
	repo := &fakeRepo{ensureWishErr: boom}
	fake := &fakeProposalService{createDealResult: domain.Proposal{DealID: "deal-1"}}
	svc := NewService(repo, fake)

	_, err := svc.Chown(context.Background(), "actor-1", "target", dto.CreateProposalRequest{ItemID: "offer-1", Quantity: 1})
	if !errors.Is(err, boom) {
		t.Errorf("Chown() error = %v, want wrapping %v", err, boom)
	}
}

func TestService_Chown_Success_ReturnsCreatedProposalUnchanged(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	want := domain.Proposal{
		DealID: "deal-1", TransactionID: "tx-1", ParticipantID: "actor-1",
		ItemID: "offer-1", ToUserID: "recipient-1", Quantity: 2,
		Status: domain.ProposalStatusPending, UpdatedAt: updatedAt,
	}
	repo := &fakeRepo{}
	fake := &fakeProposalService{createDealResult: want}
	svc := NewService(repo, fake)

	got, err := svc.Chown(context.Background(), "actor-1", "target", dto.CreateProposalRequest{ItemID: "offer-1", Quantity: 2})
	if err != nil {
		t.Fatalf("Chown(): %v", err)
	}
	if got != want {
		t.Errorf("Chown() = %+v, want %+v", got, want)
	}
}

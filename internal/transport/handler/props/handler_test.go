package props

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"avitohvk/internal/domain"
	"avitohvk/internal/dto"
	statusErrors "avitohvk/internal/errors"
	"avitohvk/internal/transport/middleware"

	"github.com/go-chi/chi/v5"
)

type fakeService struct {
	createDealResult     domain.Proposal
	createDealErr        error
	gotCreateDealActorID string
	gotCreateDealReq     dto.CreateDealRequest
	createDealCalled     bool

	createProposalResult     domain.Proposal
	createProposalErr        error
	gotCreateProposalActorID string
	gotCreateProposalDealID  string
	gotCreateProposalReq     dto.CreateProposalRequest
	createProposalCalled     bool

	getProposalResult domain.Proposal
	getProposalErr    error
	gotGetActorID     string
	gotGetDealID      string
	getCalled         bool

	updateProposalResult domain.Proposal
	updateProposalErr    error
	gotUpdateActorID     string
	gotUpdateDealID      string
	gotUpdateUpd         domain.ProposalUpdate
	updateCalled         bool

	withdrawErr        error
	gotWithdrawActorID string
	gotWithdrawDealID  string
	withdrawCalled     bool

	listResult     []domain.Proposal
	listErr        error
	gotListActorID string
	gotListUserID  string
	listCalled     bool

	approveResult     domain.Proposal
	approveErr        error
	gotApproveActorID string
	gotApproveDealID  string
	approveCalled     bool

	gotCtx context.Context
}

func (f *fakeService) CreateDeal(ctx context.Context, actorID string, req dto.CreateDealRequest) (domain.Proposal, error) {
	f.createDealCalled = true
	f.gotCtx = ctx
	f.gotCreateDealActorID = actorID
	f.gotCreateDealReq = req
	return f.createDealResult, f.createDealErr
}

func (f *fakeService) CreateProposal(ctx context.Context, actorID, dealID string, req dto.CreateProposalRequest) (domain.Proposal, error) {
	f.createProposalCalled = true
	f.gotCtx = ctx
	f.gotCreateProposalActorID = actorID
	f.gotCreateProposalDealID = dealID
	f.gotCreateProposalReq = req
	return f.createProposalResult, f.createProposalErr
}

func (f *fakeService) GetProposal(ctx context.Context, actorID, dealID string) (domain.Proposal, error) {
	f.getCalled = true
	f.gotCtx = ctx
	f.gotGetActorID = actorID
	f.gotGetDealID = dealID
	return f.getProposalResult, f.getProposalErr
}

func (f *fakeService) UpdateProposal(ctx context.Context, actorID, dealID string, upd domain.ProposalUpdate) (domain.Proposal, error) {
	f.updateCalled = true
	f.gotCtx = ctx
	f.gotUpdateActorID = actorID
	f.gotUpdateDealID = dealID
	f.gotUpdateUpd = upd
	return f.updateProposalResult, f.updateProposalErr
}

func (f *fakeService) WithdrawProposal(ctx context.Context, actorID, dealID string) error {
	f.withdrawCalled = true
	f.gotCtx = ctx
	f.gotWithdrawActorID = actorID
	f.gotWithdrawDealID = dealID
	return f.withdrawErr
}

func (f *fakeService) ListForUser(ctx context.Context, actorID, userID string) ([]domain.Proposal, error) {
	f.listCalled = true
	f.gotCtx = ctx
	f.gotListActorID = actorID
	f.gotListUserID = userID
	return f.listResult, f.listErr
}

func (f *fakeService) Approve(ctx context.Context, actorID, dealID string) (domain.Proposal, error) {
	f.approveCalled = true
	f.gotCtx = ctx
	f.gotApproveActorID = actorID
	f.gotApproveDealID = dealID
	return f.approveResult, f.approveErr
}

func newTestRouter(svc *fakeService) chi.Router {
	r := chi.NewRouter()
	New(svc).RegisterRoutes(r)
	return r
}

func authedRequest(method, path, userID, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if userID != "" {
		req = req.WithContext(middleware.ContextWithUserID(req.Context(), userID))
	}
	return req
}

func strPtr(s string) *string     { return &s }
func floatPtr(f float64) *float64 { return &f }

func assertProposalResponse(t *testing.T, got dto.ProposalResponse, want domain.Proposal) {
	t.Helper()
	if got.DealID != want.DealID ||
		got.TransactionID != want.TransactionID ||
		got.ParticipantID != want.ParticipantID ||
		got.ItemID != want.ItemID ||
		got.ToUserID != want.ToUserID ||
		got.Quantity != want.Quantity ||
		got.Status != string(want.Status) ||
		!got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Errorf("response = %+v, want fields matching %+v", got, want)
	}
}

func TestHandler_CreateDeal_NoAuth(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodPost, "/props/", "", `{"root_item_id":"item-1","item_id":"item-2","quantity":1}`)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if svc.createDealCalled {
		t.Errorf("service was called despite missing auth")
	}
}

func TestHandler_CreateDeal_MalformedBody(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodPost, "/props/", "actor-1", `{"root_item_id":`)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if svc.createDealCalled {
		t.Errorf("service was called despite a malformed body")
	}
}

func TestHandler_CreateDeal_ServiceErrorMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
	}{
		{name: "bad request sentinel", svcErr: statusErrors.ErrBadRequest, wantStatus: http.StatusBadRequest},
		{name: "not found sentinel", svcErr: statusErrors.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "conflict sentinel", svcErr: statusErrors.ErrConflict, wantStatus: http.StatusConflict},
		{name: "unmapped error is a 500", svcErr: errors.New("db down"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeService{createDealErr: tt.svcErr}
			r := newTestRouter(svc)

			req := authedRequest(http.MethodPost, "/props/", "actor-1", `{"root_item_id":"item-1","item_id":"item-2","quantity":1}`)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_CreateDeal_Success(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	want := domain.Proposal{
		DealID: "deal-1", TransactionID: "tx-1", ParticipantID: "actor-1", ItemID: "item-2",
		ToUserID: "recipient-1", Quantity: 3, Status: domain.ProposalStatusPending, UpdatedAt: updatedAt,
	}
	svc := &fakeService{createDealResult: want}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodPost, "/props/", "actor-1", `{"root_item_id":"item-1","item_id":"item-2","quantity":3}`)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if svc.gotCreateDealActorID != "actor-1" {
		t.Errorf("actorID = %q, want %q", svc.gotCreateDealActorID, "actor-1")
	}
	wantReq := dto.CreateDealRequest{RootItemID: "item-1", ItemID: "item-2", Quantity: 3}
	if svc.gotCreateDealReq != wantReq {
		t.Errorf("req = %+v, want %+v", svc.gotCreateDealReq, wantReq)
	}
	var got dto.ProposalResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	assertProposalResponse(t, got, want)
}

func TestHandler_Create_NoAuth(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodPost, "/props/deal-1", "", `{"item_id":"item-2","quantity":1}`)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if svc.createProposalCalled {
		t.Errorf("service was called despite missing auth")
	}
}

func TestHandler_Create_MalformedBody(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodPost, "/props/deal-1", "actor-1", `not json`)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if svc.createProposalCalled {
		t.Errorf("service was called despite a malformed body")
	}
}

func TestHandler_Create_ServiceErrorMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
	}{
		{name: "bad request sentinel", svcErr: statusErrors.ErrBadRequest, wantStatus: http.StatusBadRequest},
		{name: "not found sentinel", svcErr: statusErrors.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "conflict sentinel", svcErr: statusErrors.ErrConflict, wantStatus: http.StatusConflict},
		{name: "unmapped error is a 500", svcErr: errors.New("db down"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeService{createProposalErr: tt.svcErr}
			r := newTestRouter(svc)

			req := authedRequest(http.MethodPost, "/props/deal-1", "actor-1", `{"item_id":"item-2","quantity":1}`)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_Create_Success(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	want := domain.Proposal{
		DealID: "deal-1", TransactionID: "tx-1", ParticipantID: "actor-1", ItemID: "item-2",
		ToUserID: "recipient-1", Quantity: 2, Status: domain.ProposalStatusPending, UpdatedAt: updatedAt,
	}
	svc := &fakeService{createProposalResult: want}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodPost, "/props/deal-1", "actor-1", `{"item_id":"item-2","quantity":2}`)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if svc.gotCreateProposalDealID != "deal-1" {
		t.Errorf("dealID = %q, want %q (from URL)", svc.gotCreateProposalDealID, "deal-1")
	}
	wantReq := dto.CreateProposalRequest{ItemID: "item-2", Quantity: 2}
	if svc.gotCreateProposalReq != wantReq {
		t.Errorf("req = %+v, want %+v", svc.gotCreateProposalReq, wantReq)
	}
	var got dto.ProposalResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	assertProposalResponse(t, got, want)
}

func TestHandler_Get_NoAuth(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodGet, "/props/deal-1", "", "")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if svc.getCalled {
		t.Errorf("service was called despite missing auth")
	}
}

func TestHandler_Get_ServiceErrorMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
	}{
		{name: "bad request sentinel", svcErr: statusErrors.ErrBadRequest, wantStatus: http.StatusBadRequest},
		{name: "not found sentinel", svcErr: statusErrors.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "conflict sentinel", svcErr: statusErrors.ErrConflict, wantStatus: http.StatusConflict},
		{name: "unmapped error is a 500", svcErr: errors.New("db down"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeService{getProposalErr: tt.svcErr}
			r := newTestRouter(svc)

			req := authedRequest(http.MethodGet, "/props/deal-1", "actor-1", "")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_Get_Success(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	want := domain.Proposal{
		DealID: "deal-1", TransactionID: "tx-1", ParticipantID: "actor-1", ItemID: "item-2",
		ToUserID: "recipient-1", Quantity: 4, Status: domain.ProposalStatusAccepted, UpdatedAt: updatedAt,
	}
	svc := &fakeService{getProposalResult: want}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodGet, "/props/deal-1", "actor-1", "")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if svc.gotGetActorID != "actor-1" || svc.gotGetDealID != "deal-1" {
		t.Errorf("actorID/dealID = %q/%q, want actor-1/deal-1", svc.gotGetActorID, svc.gotGetDealID)
	}
	var got dto.ProposalResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	assertProposalResponse(t, got, want)
}

func TestHandler_Update_NoAuth(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodPatch, "/props/deal-1", "", `{"quantity":2}`)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if svc.updateCalled {
		t.Errorf("service was called despite missing auth")
	}
}

func TestHandler_Update_MalformedBody(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodPatch, "/props/deal-1", "actor-1", `{`)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if svc.updateCalled {
		t.Errorf("service was called despite a malformed body")
	}
}

func TestHandler_Update_MapsRequestFieldsToDomainUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want domain.ProposalUpdate
	}{
		{
			name: "both fields set",
			body: `{"item_id":"item-9","quantity":5}`,
			want: domain.ProposalUpdate{ItemID: strPtr("item-9"), Quantity: floatPtr(5)},
		},
		{
			name: "only item_id set — quantity stays nil",
			body: `{"item_id":"item-9"}`,
			want: domain.ProposalUpdate{ItemID: strPtr("item-9"), Quantity: nil},
		},
		{
			name: "only quantity set — item_id stays nil",
			body: `{"quantity":5}`,
			want: domain.ProposalUpdate{ItemID: nil, Quantity: floatPtr(5)},
		},
		{
			name: "empty body — both nil",
			body: `{}`,
			want: domain.ProposalUpdate{ItemID: nil, Quantity: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeService{}
			r := newTestRouter(svc)

			req := authedRequest(http.MethodPatch, "/props/deal-1", "actor-1", tt.body)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if !svc.updateCalled {
				t.Fatalf("service was not called")
			}
			got := svc.gotUpdateUpd
			if (got.ItemID == nil) != (tt.want.ItemID == nil) || (got.ItemID != nil && *got.ItemID != *tt.want.ItemID) {
				t.Errorf("ItemID = %v, want %v", got.ItemID, tt.want.ItemID)
			}
			if (got.Quantity == nil) != (tt.want.Quantity == nil) || (got.Quantity != nil && *got.Quantity != *tt.want.Quantity) {
				t.Errorf("Quantity = %v, want %v", got.Quantity, tt.want.Quantity)
			}
		})
	}
}

func TestHandler_Update_ServiceErrorMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
	}{
		{name: "bad request sentinel", svcErr: statusErrors.ErrBadRequest, wantStatus: http.StatusBadRequest},
		{name: "not found sentinel", svcErr: statusErrors.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "conflict sentinel", svcErr: statusErrors.ErrConflict, wantStatus: http.StatusConflict},
		{name: "unmapped error is a 500", svcErr: errors.New("db down"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeService{updateProposalErr: tt.svcErr}
			r := newTestRouter(svc)

			req := authedRequest(http.MethodPatch, "/props/deal-1", "actor-1", `{"quantity":2}`)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_Update_Success(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	want := domain.Proposal{
		DealID: "deal-1", TransactionID: "tx-1", ParticipantID: "actor-1", ItemID: "item-9",
		ToUserID: "recipient-1", Quantity: 5, Status: domain.ProposalStatusPending, UpdatedAt: updatedAt,
	}
	svc := &fakeService{updateProposalResult: want}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodPatch, "/props/deal-1", "actor-1", `{"item_id":"item-9","quantity":5}`)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got dto.ProposalResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	assertProposalResponse(t, got, want)
}

func TestHandler_Delete_NoAuth(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodDelete, "/props/deal-1", "", "")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if svc.withdrawCalled {
		t.Errorf("service was called despite missing auth")
	}
}

func TestHandler_Delete_ServiceErrorMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
	}{
		{name: "bad request sentinel", svcErr: statusErrors.ErrBadRequest, wantStatus: http.StatusBadRequest},
		{name: "not found sentinel", svcErr: statusErrors.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "conflict sentinel", svcErr: statusErrors.ErrConflict, wantStatus: http.StatusConflict},
		{name: "unmapped error is a 500", svcErr: errors.New("db down"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeService{withdrawErr: tt.svcErr}
			r := newTestRouter(svc)

			req := authedRequest(http.MethodDelete, "/props/deal-1", "actor-1", "")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_Delete_Success(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodDelete, "/props/deal-1", "actor-1", "")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty for 204", rec.Body.String())
	}
	if svc.gotWithdrawActorID != "actor-1" || svc.gotWithdrawDealID != "deal-1" {
		t.Errorf("actorID/dealID = %q/%q, want actor-1/deal-1", svc.gotWithdrawActorID, svc.gotWithdrawDealID)
	}
}

func TestHandler_GetByUser_NoAuth(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodGet, "/props/users/user-1", "", "")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if svc.listCalled {
		t.Errorf("service was called despite missing auth")
	}
}

func TestHandler_GetByUser_ServiceErrorMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
	}{
		{name: "bad request sentinel", svcErr: statusErrors.ErrBadRequest, wantStatus: http.StatusBadRequest},
		{name: "not found sentinel", svcErr: statusErrors.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "conflict sentinel", svcErr: statusErrors.ErrConflict, wantStatus: http.StatusConflict},
		{name: "unmapped error is a 500", svcErr: errors.New("db down"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeService{listErr: tt.svcErr}
			r := newTestRouter(svc)

			req := authedRequest(http.MethodGet, "/props/users/user-1", "actor-1", "")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_GetByUser_Success(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	want := []domain.Proposal{
		{DealID: "deal-1", TransactionID: "tx-1", ParticipantID: "actor-1", ItemID: "item-1", ToUserID: "recipient-1", Quantity: 1, Status: domain.ProposalStatusPending, UpdatedAt: updatedAt},
		{DealID: "deal-2", TransactionID: "tx-2", ParticipantID: "actor-1", ItemID: "item-2", ToUserID: "recipient-2", Quantity: 2, Status: domain.ProposalStatusAccepted, UpdatedAt: updatedAt},
	}
	svc := &fakeService{listResult: want}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodGet, "/props/users/user-1", "actor-1", "")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if svc.gotListActorID != "actor-1" || svc.gotListUserID != "user-1" {
		t.Errorf("actorID/userID = %q/%q, want actor-1/user-1", svc.gotListActorID, svc.gotListUserID)
	}
	var got dto.ProposalListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(got.Proposals) != len(want) {
		t.Fatalf("proposals count = %d, want %d", len(got.Proposals), len(want))
	}
	for i, w := range want {
		assertProposalResponse(t, got.Proposals[i], w)
	}
}

func TestHandler_GetByUser_EmptyListProducesEmptyArrayNotNull(t *testing.T) {
	t.Parallel()

	svc := &fakeService{listResult: nil}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodGet, "/props/users/user-1", "actor-1", "")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	body := strings.TrimSpace(rec.Body.String())
	if !strings.Contains(body, `"proposals":[]`) {
		t.Errorf("body = %q, want a JSON array (`[]`) not `null` for an empty proposal list", body)
	}
}

func TestHandler_Approve_NoAuth(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodPost, "/props/approve/deal-1", "", "")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if svc.approveCalled {
		t.Errorf("service was called despite missing auth")
	}
}

func TestHandler_Approve_ServiceErrorMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
	}{
		{name: "bad request sentinel", svcErr: statusErrors.ErrBadRequest, wantStatus: http.StatusBadRequest},
		{name: "not found sentinel", svcErr: statusErrors.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "conflict sentinel", svcErr: statusErrors.ErrConflict, wantStatus: http.StatusConflict},
		{name: "unmapped error is a 500", svcErr: errors.New("db down"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeService{approveErr: tt.svcErr}
			r := newTestRouter(svc)

			req := authedRequest(http.MethodPost, "/props/approve/deal-1", "actor-1", "")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_Approve_Success(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	want := domain.Proposal{
		DealID: "deal-1", TransactionID: "tx-1", ParticipantID: "actor-1", ItemID: "item-1",
		ToUserID: "recipient-1", Quantity: 1, Status: domain.ProposalStatusAccepted, UpdatedAt: updatedAt,
	}
	svc := &fakeService{approveResult: want}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodPost, "/props/approve/deal-1", "actor-1", "")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if svc.gotApproveActorID != "actor-1" || svc.gotApproveDealID != "deal-1" {
		t.Errorf("actorID/dealID = %q/%q, want actor-1/deal-1", svc.gotApproveActorID, svc.gotApproveDealID)
	}
	var got dto.ProposalResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	assertProposalResponse(t, got, want)
}

func TestHandler_WrongMethodIsNotAllowed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "PUT on /props/{deal_id} (only POST/GET/PATCH/DELETE registered)", method: http.MethodPut, path: "/props/deal-1"},
		{name: "POST on /props/users/{user_id} (only GET registered)", method: http.MethodPost, path: "/props/users/user-1"},
		{name: "GET on /props/approve/{deal_id} (only POST registered)", method: http.MethodGet, path: "/props/approve/deal-1"},
		{name: "GET on /props/ (deal_id is a distinct static pattern, only POST registered)", method: http.MethodGet, path: "/props/"},
		{name: "DELETE on /props/ (deal_id is a distinct static pattern, only POST registered)", method: http.MethodDelete, path: "/props/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeService{}
			r := newTestRouter(svc)

			req := authedRequest(tt.method, tt.path, "actor-1", "")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestHandler_ForwardsRequestContext(t *testing.T) {
	t.Parallel()

	type ctxKey struct{}
	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodGet, "/props/deal-1", "actor-1", "")
	req = req.WithContext(context.WithValue(req.Context(), ctxKey{}, "trace-id-123"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if svc.gotCtx == nil || svc.gotCtx.Value(ctxKey{}) != "trace-id-123" {
		t.Errorf("service did not receive the request's own context (with its values) unchanged")
	}
}

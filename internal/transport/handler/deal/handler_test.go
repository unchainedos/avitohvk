package deal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"avitohvk/internal/domain"
	statusErrors "avitohvk/internal/errors"
	"avitohvk/internal/transport/middleware"

	"github.com/go-chi/chi/v5"
)

type fakeService struct {
	result domain.Deal
	err    error

	called    bool
	gotCtx    context.Context
	gotDealID string
}

func (f *fakeService) GetByID(ctx context.Context, id string) (domain.Deal, error) {
	f.called = true
	f.gotCtx = ctx
	f.gotDealID = id
	return f.result, f.err
}

func newTestRouter(svc *fakeService) chi.Router {
	r := chi.NewRouter()
	New(svc).RegisterRoutes(r)
	return r
}

func authedRequest(method, path, userID string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	if userID != "" {
		req = req.WithContext(middleware.ContextWithUserID(req.Context(), userID))
	}
	return req
}

func TestHandler_Get_NoAuth(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodGet, "/deal/deal-1", "")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if svc.called {
		t.Errorf("service was called despite missing auth")
	}
}

func TestHandler_Get_WrongMethodIsNotAllowed(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodPost, "/deal/deal-1", "actor-1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d — RegisterRoutes must only accept GET", rec.Code, http.StatusMethodNotAllowed)
	}
	if svc.called {
		t.Errorf("service was called for a disallowed method")
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

			svc := &fakeService{err: tt.svcErr}
			r := newTestRouter(svc)

			req := authedRequest(http.MethodGet, "/deal/deal-1", "actor-1")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_Get_ForwardsRequestContext(t *testing.T) {
	t.Parallel()

	type ctxKey struct{}
	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodGet, "/deal/deal-1", "actor-1")
	req = req.WithContext(context.WithValue(req.Context(), ctxKey{}, "trace-id-123"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if svc.gotCtx == nil || svc.gotCtx.Value(ctxKey{}) != "trace-id-123" {
		t.Errorf("service did not receive the request's own context (with its values) unchanged")
	}
}

func TestHandler_Get_DealIDFromURL(t *testing.T) {
	t.Parallel()

	tests := []string{"some-deal-id", "not-a-uuid"}

	for _, dealID := range tests {
		t.Run(dealID, func(t *testing.T) {
			t.Parallel()

			svc := &fakeService{}
			r := newTestRouter(svc)

			req := authedRequest(http.MethodGet, "/deal/"+dealID, "actor-1")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if svc.gotDealID != dealID {
				t.Errorf("service received dealID = %q, want %q", svc.gotDealID, dealID)
			}
		})
	}
}

func TestHandler_Get_Success(t *testing.T) {
	t.Parallel()

	deadline := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	svc := &fakeService{result: domain.Deal{
		ID:                "deal-1",
		RootItemID:        "item-1",
		CreatorID:         "actor-1",
		Status:            domain.DealStatusPending,
		NegotiationWindow: 24 * time.Hour,
		DeadlineAt:        &deadline,
		CreatedAt:         created,
	}}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodGet, "/deal/deal-1", "actor-1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v (%q)", err, rec.Body.String())
	}
	if got["id"] != "deal-1" || got["root_item_id"] != "item-1" || got["creator_id"] != "actor-1" || got["status"] != "PENDING" {
		t.Errorf("response body = %+v", got)
	}
	if got["negotiation_window_seconds"] != float64(24*60*60) {
		t.Errorf("negotiation_window_seconds = %v, want %v", got["negotiation_window_seconds"], 24*60*60)
	}
	if got["created_at"] != created.Format(time.RFC3339Nano) {
		t.Errorf("created_at = %v, want %v", got["created_at"], created.Format(time.RFC3339Nano))
	}
	if got["deadline_at"] != deadline.Format(time.RFC3339Nano) {
		t.Errorf("deadline_at = %v, want %v", got["deadline_at"], deadline.Format(time.RFC3339Nano))
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestHandler_Get_SuccessWithNilDeadline(t *testing.T) {
	t.Parallel()

	svc := &fakeService{result: domain.Deal{ID: "deal-1", Status: domain.DealStatusPending, DeadlineAt: nil}}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodGet, "/deal/deal-1", "actor-1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if v, ok := got["deadline_at"]; !ok || v != nil {
		t.Errorf("deadline_at = %v, want JSON null", got["deadline_at"])
	}
}

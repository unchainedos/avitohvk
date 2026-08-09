package chown

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"avitohvk/internal/dto"
	statusErrors "avitohvk/internal/errors"
	"avitohvk/internal/transport/middleware"

	"github.com/go-chi/chi/v5"
)

type fakeService struct {
	result dto.ChownResponse
	err    error

	called     bool
	gotCtx     context.Context
	gotActorID string
	gotItemID  string
	gotReq     dto.ChownRequest
}

func (f *fakeService) Chown(ctx context.Context, actorID, itemID string, req dto.ChownRequest) (dto.ChownResponse, error) {
	f.called = true
	f.gotCtx = ctx
	f.gotActorID = actorID
	f.gotItemID = itemID
	f.gotReq = req
	return f.result, f.err
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

func TestHandler_Chown_NoAuth(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodPost, "/chown/item-1", "", `{"offered_items":[{"item_id":"x","quantity":1}]}`)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if svc.called {
		t.Errorf("service was called despite missing auth")
	}
}

func TestHandler_Chown_MalformedBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ""},
		{name: "truncated json", body: `{"offered_items":`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeService{}
			r := newTestRouter(svc)

			req := authedRequest(http.MethodPost, "/chown/item-1", "actor-1", tt.body)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
			if svc.called {
				t.Errorf("service was called despite a malformed body")
			}
			wantBody := `{"message":"bad request"}`
			if got := strings.TrimSpace(rec.Body.String()); got != wantBody {
				t.Errorf("body = %q, want %q (must not leak the underlying JSON decode error)", got, wantBody)
			}
		})
	}
}

func TestHandler_Chown_WrongMethodIsNotAllowed(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodGet, "/chown/item-1", "actor-1", "")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d — RegisterRoutes must only accept POST", rec.Code, http.StatusMethodNotAllowed)
	}
	if svc.called {
		t.Errorf("service was called for a disallowed method")
	}
}

func TestHandler_Chown_ForwardsRequestContext(t *testing.T) {
	t.Parallel()

	type ctxKey struct{}
	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodPost, "/chown/item-1", "actor-1", `{"offered_items":[{"item_id":"x","quantity":1}]}`)
	req = req.WithContext(context.WithValue(req.Context(), ctxKey{}, "trace-id-123"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if svc.gotCtx == nil || svc.gotCtx.Value(ctxKey{}) != "trace-id-123" {
		t.Errorf("service did not receive the request's own context (with its values) unchanged")
	}
}

func TestHandler_Chown_ServiceErrorMapping(t *testing.T) {
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

			req := authedRequest(http.MethodPost, "/chown/item-1", "actor-1", `{"offered_items":[{"item_id":"x","quantity":1}]}`)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_Chown_Success(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	svc := &fakeService{result: dto.ChownResponse{
		ItemID:     "item-1",
		FromUserID: "actor-1",
		Hops:       []dto.ChownHop{{ItemID: "offer-1", Quantity: 2, ToUserID: "recipient-1"}},
		CreatedAt:  createdAt,
	}}
	r := newTestRouter(svc)

	body := `{"offered_items":[{"item_id":"offer-1","quantity":2}]}`
	req := authedRequest(http.MethodPost, "/chown/item-1", "actor-1", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	if !svc.called {
		t.Fatalf("service was not called")
	}
	if svc.gotActorID != "actor-1" {
		t.Errorf("service received actorID = %q, want %q", svc.gotActorID, "actor-1")
	}
	if svc.gotItemID != "item-1" {
		t.Errorf("service received itemID = %q (from the URL param), want %q", svc.gotItemID, "item-1")
	}
	wantReq := dto.ChownRequest{OfferedItems: []dto.OfferedItem{{ItemID: "offer-1", Quantity: 2}}}
	if len(svc.gotReq.OfferedItems) != 1 || svc.gotReq.OfferedItems[0] != wantReq.OfferedItems[0] {
		t.Errorf("service received req = %+v, want %+v", svc.gotReq, wantReq)
	}

	var got dto.ChownResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v (%q)", err, rec.Body.String())
	}
	if got.ItemID != "item-1" || got.FromUserID != "actor-1" || len(got.Hops) != 1 {
		t.Errorf("response body = %+v", got)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestHandler_Chown_ItemIDComesFromURLNotBody(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	r := newTestRouter(svc)

	req := authedRequest(http.MethodPost, "/chown/from-url", "actor-1", `{"offered_items":[{"item_id":"offer-1","quantity":1}]}`)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if svc.gotItemID != "from-url" {
		t.Errorf("service received itemID = %q, want %q", svc.gotItemID, "from-url")
	}
}

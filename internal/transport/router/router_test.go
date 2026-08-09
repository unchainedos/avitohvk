package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func getRoute(path, body string) RouteRegistrator {
	return RegistratorFunc(func(r chi.Router) {
		r.Get(path, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(body))
		})
	})
}

func headerGateMiddleware(headerName string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(headerName) == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func TestRegistratorFunc(t *testing.T) {
	t.Parallel()

	var called bool
	var gotRouter chi.Router
	f := RegistratorFunc(func(r chi.Router) {
		called = true
		gotRouter = r
	})

	r := chi.NewRouter()
	f.RegisterRoutes(r)

	if !called {
		t.Fatalf("RegistratorFunc.RegisterRoutes() did not invoke the wrapped func")
	}
	if gotRouter != r {
		t.Errorf("RegistratorFunc.RegisterRoutes() passed a different router than given")
	}
}

func TestWithGroup(t *testing.T) {
	t.Parallel()

	reg1 := getRoute("/a", "a")
	reg2 := getRoute("/b", "b")
	mw1 := headerGateMiddleware("X-One")
	mw2 := headerGateMiddleware("X-Two")

	tests := []struct {
		name         string
		registrators []RouteRegistrator
		middlewares  []Middleware
		wantRegCount int
		wantMwCount  int
	}{
		{
			name:         "no middlewares",
			registrators: []RouteRegistrator{reg1},
			middlewares:  nil,
			wantRegCount: 1,
			wantMwCount:  0,
		},
		{
			name:         "with middlewares",
			registrators: []RouteRegistrator{reg1, reg2},
			middlewares:  []Middleware{mw1, mw2},
			wantRegCount: 2,
			wantMwCount:  2,
		},
		{
			name:         "empty group is still recorded",
			registrators: nil,
			middlewares:  nil,
			wantRegCount: 0,
			wantMwCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &GroupConfig{}
			opt := WithGroup(tt.registrators, tt.middlewares...)
			opt(cfg)

			if len(cfg.groups) != 1 {
				t.Fatalf("groups recorded = %d, want 1", len(cfg.groups))
			}
			g := cfg.groups[0]
			if len(g.registrators) != tt.wantRegCount {
				t.Errorf("registrators recorded = %d, want %d", len(g.registrators), tt.wantRegCount)
			}
			if len(g.middlewares) != tt.wantMwCount {
				t.Errorf("middlewares recorded = %d, want %d", len(g.middlewares), tt.wantMwCount)
			}
		})
	}
}

func TestWithGroup_MultipleCallsAccumulate(t *testing.T) {
	t.Parallel()

	cfg := &GroupConfig{}
	WithGroup([]RouteRegistrator{getRoute("/a", "a")})(cfg)
	WithGroup([]RouteRegistrator{getRoute("/b", "b")}, headerGateMiddleware("X-Auth"))(cfg)

	if len(cfg.groups) != 2 {
		t.Fatalf("groups recorded = %d, want 2", len(cfg.groups))
	}
	if len(cfg.groups[0].middlewares) != 0 {
		t.Errorf("first group middlewares = %d, want 0", len(cfg.groups[0].middlewares))
	}
	if len(cfg.groups[1].middlewares) != 1 {
		t.Errorf("second group middlewares = %d, want 1", len(cfg.groups[1].middlewares))
	}
}

func TestNew_NoGroups(t *testing.T) {
	t.Parallel()

	handler := New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/anything", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d for an unregistered route on an empty router", rec.Code, http.StatusNotFound)
	}
}

func TestNew_RoutesAreMountedUnderAPIV1Prefix(t *testing.T) {
	t.Parallel()

	handler := New(WithGroup([]RouteRegistrator{getRoute("/ping", "pong")}))

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{name: "with prefix", path: "/api/v1/ping", wantStatus: http.StatusOK},
		{name: "without prefix", path: "/ping", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestNew_MultipleRegistratorsInSameGroupAllRegister(t *testing.T) {
	t.Parallel()

	handler := New(WithGroup([]RouteRegistrator{
		getRoute("/a", "a"),
		getRoute("/b", "b"),
	}))

	for _, path := range []string{"/api/v1/a", "/api/v1/b"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", path, rec.Code, http.StatusOK)
		}
	}
}

func TestNew_MiddlewareAppliesOnlyToItsOwnGroup(t *testing.T) {
	t.Parallel()

	handler := New(
		WithGroup([]RouteRegistrator{getRoute("/public", "public")}),
		WithGroup([]RouteRegistrator{getRoute("/private", "private")}, headerGateMiddleware("X-Auth")),
	)

	tests := []struct {
		name       string
		path       string
		authHeader bool
		wantStatus int
	}{
		{name: "public route needs no auth", path: "/api/v1/public", authHeader: false, wantStatus: http.StatusOK},
		{name: "private route rejects without auth", path: "/api/v1/private", authHeader: false, wantStatus: http.StatusUnauthorized},
		{name: "private route allows with auth", path: "/api/v1/private", authHeader: true, wantStatus: http.StatusOK},
		{name: "public route still fine even with auth header present", path: "/api/v1/public", authHeader: true, wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.authHeader {
				req.Header.Set("X-Auth", "1")
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestNew_MultipleMiddlewaresInAGroupAllApply(t *testing.T) {
	t.Parallel()

	handler := New(WithGroup(
		[]RouteRegistrator{getRoute("/gated", "ok")},
		headerGateMiddleware("X-One"),
		headerGateMiddleware("X-Two"),
	))

	tests := []struct {
		name       string
		headers    map[string]string
		wantStatus int
	}{
		{name: "neither header", headers: nil, wantStatus: http.StatusUnauthorized},
		{name: "only first header", headers: map[string]string{"X-One": "1"}, wantStatus: http.StatusUnauthorized},
		{name: "only second header", headers: map[string]string{"X-Two": "1"}, wantStatus: http.StatusUnauthorized},
		{name: "both headers", headers: map[string]string{"X-One": "1", "X-Two": "1"}, wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/api/v1/gated", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestNew_MiddlewareExecutesInRegistrationOrder(t *testing.T) {
	t.Parallel()

	var order []string
	trace := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}

	handler := New(WithGroup(
		[]RouteRegistrator{getRoute("/traced", "ok")},
		trace("first"), trace("second"),
	))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/traced", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	want := []string{"first", "second"}
	if len(order) != len(want) {
		t.Fatalf("execution order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Errorf("execution order = %v, want %v (the first-registered middleware must be the outermost, executing before the second)", order, want)
			break
		}
	}
}

func TestNew_GroupWithNoRegistratorsDoesNotPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("New() panicked building a group with no registrators but a middleware: %v", r)
		}
	}()

	handler := New(WithGroup(nil, headerGateMiddleware("X-Auth")))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/anything", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestNew_RegistratorCanRegisterMultipleRoutes(t *testing.T) {
	t.Parallel()

	multi := RegistratorFunc(func(r chi.Router) {
		r.Get("/multi/a", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("a")) })
		r.Post("/multi/b", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("b")) })
	})

	handler := New(WithGroup([]RouteRegistrator{multi}))

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/multi/a"},
		{method: http.MethodPost, path: "/api/v1/multi/b"},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("%s %s status = %d, want 200", tt.method, tt.path, rec.Code)
		}
	}
}

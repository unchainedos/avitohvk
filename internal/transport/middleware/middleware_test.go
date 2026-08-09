package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("test-secret")

func signToken(t *testing.T, method jwt.SigningMethod, key any, claims UserClaims) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("failed to construct test token: %v", err)
	}
	return tok
}

func validClaims(userID string, ttl time.Duration) UserClaims {
	now := time.Now()
	return UserClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
}

func TestNewJWTAuth(t *testing.T) {
	t.Parallel()

	validToken := signToken(t, jwt.SigningMethodHS256, testSecret, validClaims("user-123", time.Hour))
	wrongSecretToken := signToken(t, jwt.SigningMethodHS256, []byte("wrong-secret"), validClaims("user-123", time.Hour))
	expiredToken := signToken(t, jwt.SigningMethodHS256, testSecret, validClaims("user-123", -time.Hour))
	emptyUserIDToken := signToken(t, jwt.SigningMethodHS256, testSecret, validClaims("", time.Hour))
	noneAlgToken := signToken(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims("user-123", time.Hour))

	noExpClaims := UserClaims{UserID: "user-123", RegisteredClaims: jwt.RegisteredClaims{IssuedAt: jwt.NewNumericDate(time.Now())}}
	noExpToken := signToken(t, jwt.SigningMethodHS256, testSecret, noExpClaims)

	tests := []struct {
		name       string
		cookie     *http.Cookie
		wantStatus int
		wantNext   bool
		wantUserID string
	}{
		{
			name:       "missing cookie",
			cookie:     nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "empty cookie value",
			cookie:     &http.Cookie{Name: AccessTokenCookie, Value: ""},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "malformed token string",
			cookie:     &http.Cookie{Name: AccessTokenCookie, Value: "not-a-valid-jwt"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "token signed with a different secret",
			cookie:     &http.Cookie{Name: AccessTokenCookie, Value: wrongSecretToken},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "non-HMAC signing method is rejected regardless of validity",
			cookie:     &http.Cookie{Name: AccessTokenCookie, Value: noneAlgToken},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "expired token",
			cookie:     &http.Cookie{Name: AccessTokenCookie, Value: expiredToken},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "token missing the exp claim entirely",
			cookie:     &http.Cookie{Name: AccessTokenCookie, Value: noExpToken},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "token with an empty user_id claim",
			cookie:     &http.Cookie{Name: AccessTokenCookie, Value: emptyUserIDToken},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "valid token passes through with the user id in context",
			cookie:     &http.Cookie{Name: AccessTokenCookie, Value: validToken},
			wantStatus: http.StatusOK,
			wantNext:   true,
			wantUserID: "user-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var nextCalled bool
			var gotUserID string
			var gotOK bool
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				gotUserID, gotOK = UserIDFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			handler := NewJWTAuth(testSecret, slog.New(slog.DiscardHandler))(next)

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if nextCalled != tt.wantNext {
				t.Errorf("next called = %v, want %v", nextCalled, tt.wantNext)
			}
			if tt.wantNext {
				if !gotOK {
					t.Errorf("UserIDFromContext() ok = false inside next handler, want true")
				}
				if gotUserID != tt.wantUserID {
					t.Errorf("UserIDFromContext() = %q, want %q", gotUserID, tt.wantUserID)
				}
				return
			}

			wantBody := `{"message":"unauthorized"}`
			if got := strings.TrimSpace(rec.Body.String()); got != wantBody {
				t.Errorf("body = %q, want %q (WriteError must actually be wired up, not just a bare status code)", got, wantBody)
			}
		})
	}
}

// parentCtxKey is a distinct type from the middleware's own unexported
// ctxKey, so this cannot accidentally collide with userIDKey.
type parentCtxKey struct{}

func TestNewJWTAuth_PreservesParentContext(t *testing.T) {
	t.Parallel()

	validToken := signToken(t, jwt.SigningMethodHS256, testSecret, validClaims("user-123", time.Hour))

	var gotParentValue any
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotParentValue = r.Context().Value(parentCtxKey{})
		w.WriteHeader(http.StatusOK)
	})
	handler := NewJWTAuth(testSecret, slog.New(slog.DiscardHandler))(next)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: AccessTokenCookie, Value: validToken})
	req = req.WithContext(context.WithValue(req.Context(), parentCtxKey{}, "parent-value"))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotParentValue != "parent-value" {
		t.Errorf("parent context value = %v, want %q — middleware must derive from r.Context(), not replace it", gotParentValue, "parent-value")
	}
}

func TestNewJWTAuth_NilLoggerDoesNotPanic(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := NewJWTAuth(testSecret, nil)(next)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestUserIDFromContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ctx     func() context.Context
		wantID  string
		wantOK  bool
		comment string
	}{
		{
			name:   "no value set",
			ctx:    context.Background,
			wantOK: false,
		},
		{
			name: "correct string value set",
			ctx: func() context.Context {
				return context.WithValue(context.Background(), userIDKey, "user-abc")
			},
			wantID: "user-abc",
			wantOK: true,
		},
		{
			name: "wrong type stored under the key fails the assertion gracefully",
			ctx: func() context.Context {
				return context.WithValue(context.Background(), userIDKey, 12345)
			},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotID, gotOK := UserIDFromContext(tt.ctx())
			if gotOK != tt.wantOK {
				t.Errorf("ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotOK && gotID != tt.wantID {
				t.Errorf("id = %q, want %q", gotID, tt.wantID)
			}
		})
	}
}

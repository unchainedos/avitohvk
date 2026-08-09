package utilhttp

import (
	statusErrors "avitohvk/internal/errors"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type spyReadCloser struct {
	io.Reader
	closed bool
}

type brokenResponseWriter struct {
	header http.Header
}

func (s *spyReadCloser) Close() error {
	s.closed = true
	return nil
}

func (b *brokenResponseWriter) Header() http.Header {
	if b.header == nil {
		b.header = http.Header{}
	}
	return b.header
}

func (b *brokenResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("write: broken pipe")
}

func (b *brokenResponseWriter) WriteHeader(int) {}

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		data       any
		wantErr    bool
		wantStatus int
		wantBody   string
	}{
		{
			name:       "simple map",
			status:     http.StatusOK,
			data:       map[string]string{"hello": "world"},
			wantStatus: http.StatusOK,
			wantBody:   `{"hello":"world"}`,
		},
		{
			name:       "created status with struct",
			status:     http.StatusCreated,
			data:       statusErrors.ErrorResponse{Message: "made it"},
			wantStatus: http.StatusCreated,
			wantBody:   `{"message":"made it"}`,
		},
		{
			name:    "unmarshalable data returns error and writes nothing",
			status:  http.StatusOK,
			data:    make(chan int),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			err := WriteJSON(rec, tt.status, tt.data)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("WriteJSON() error = nil, want non-nil")
				}
				if rec.Code != 200 || rec.Body.Len() != 0 {
					t.Fatalf("WriteJSON() wrote a response despite marshal failure: code=%d body=%q", rec.Code, rec.Body.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("WriteJSON() unexpected error: %v", err)
			}
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := strings.TrimSpace(rec.Body.String()); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
		})
	}
}

func TestWriteJSON_PropagatesWriteFailure(t *testing.T) {
	t.Parallel()

	w := &brokenResponseWriter{}
	err := WriteJSON(w, http.StatusOK, map[string]string{"hello": "world"})
	if err == nil {
		t.Fatalf("WriteJSON() error = nil, want the underlying Write() error to propagate")
	}
}

func TestWriteError(t *testing.T) {
	t.Parallel()

	pgInvalidText := &pgconn.PgError{Code: "22P02", Message: "invalid input syntax for type uuid"}
	pgOther := &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}

	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantMessage string
	}{
		{
			name:        "nil error writes 200 with no body",
			err:         nil,
			wantStatus:  http.StatusOK,
			wantMessage: "",
		},
		{
			name:        "bad request sentinel",
			err:         fmt.Errorf("%w: root_item_id required", statusErrors.ErrBadRequest),
			wantStatus:  http.StatusBadRequest,
			wantMessage: "bad request: root_item_id required",
		},
		{
			name:        "unauthorized sentinel",
			err:         statusErrors.ErrUnauthorized,
			wantStatus:  http.StatusUnauthorized,
			wantMessage: "unauthorized",
		},
		{
			name:        "not found sentinel",
			err:         fmt.Errorf("%w: proposal not found", statusErrors.ErrNotFound),
			wantStatus:  http.StatusNotFound,
			wantMessage: "not found: proposal not found",
		},
		{
			name:        "conflict sentinel",
			err:         fmt.Errorf("%w: item is locked by another confirmed deal", statusErrors.ErrConflict),
			wantStatus:  http.StatusConflict,
			wantMessage: "conflict: item is locked by another confirmed deal",
		},
		{
			name:        "malformed input postgres error maps to 400 without leaking detail",
			err:         pgInvalidText,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "invalid input",
		},
		{
			name:        "malformed input postgres error wrapped still maps to 400",
			err:         fmt.Errorf("query failed: %w", pgInvalidText),
			wantStatus:  http.StatusBadRequest,
			wantMessage: "invalid input",
		},
		{
			name:        "unrelated postgres error is a 500 and does not leak SQLSTATE or detail",
			err:         pgOther,
			wantStatus:  http.StatusInternalServerError,
			wantMessage: "internal server error",
		},
		{
			name:        "unmapped plain error is a 500 and does not leak its text",
			err:         errors.New("connection reset by peer on host db-primary-3"),
			wantStatus:  http.StatusInternalServerError,
			wantMessage: "internal server error",
		},
		{
			name:        "an explicit sentinel takes precedence over the pg-code fallback when an error wraps both",
			err:         fmt.Errorf("%w: %w", statusErrors.ErrConflict, pgInvalidText),
			wantStatus:  http.StatusConflict,
			wantMessage: fmt.Sprintf("conflict: %s", pgInvalidText.Error()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			WriteError(rec, tt.err)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.err == nil {
				if rec.Body.Len() != 0 {
					t.Errorf("body = %q, want empty for nil error", rec.Body.String())
				}
				if ct := rec.Header().Get("Content-Type"); ct != "" {
					t.Errorf("Content-Type = %q, want unset for the nil-error short-circuit path", ct)
				}
				return
			}

			var got statusErrors.ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("response body is not valid JSON: %v (%q)", err, rec.Body.String())
			}
			if got.Message != tt.wantMessage {
				t.Errorf("message = %q, want %q", got.Message, tt.wantMessage)
			}
			if rec.Code == http.StatusInternalServerError {
				if strings.Contains(got.Message, "SQLSTATE") {
					t.Errorf("500 response leaked SQLSTATE: %q", got.Message)
				}
			}
		})
	}
}

func TestReadFromJSON(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}

	tests := []struct {
		name    string
		body    string
		wantErr bool
		want    payload
	}{
		{
			name: "valid json",
			body: `{"name":"widget","n":3}`,
			want: payload{Name: "widget", N: 3},
		},
		{
			name:    "empty body is a decode error",
			body:    "",
			wantErr: true,
		},
		{
			name:    "malformed json is a decode error",
			body:    `{"name":`,
			wantErr: true,
		},
		{
			name: "unknown fields are ignored, not an error",
			body: `{"name":"widget","n":3,"extra":true}`,
			want: payload{Name: "widget", N: 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/whatever", strings.NewReader(tt.body))
			got, err := ReadFromJSON[payload](req)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("ReadFromJSON() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadFromJSON() unexpected error: %v", err)
			}
			if *got != tt.want {
				t.Errorf("ReadFromJSON() = %+v, want %+v", *got, tt.want)
			}
		})
	}
}

func TestReadFromJSON_ClosesBodyOnBothPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "success path", body: `{"name":"widget","n":3}`},
		{name: "decode-error path", body: `not json`},
	}

	type payload struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spy := &spyReadCloser{Reader: strings.NewReader(tt.body)}
			req := httptest.NewRequest(http.MethodPost, "/whatever", spy)
			req.Body = spy

			_, _ = ReadFromJSON[payload](req)

			if !spy.closed {
				t.Errorf("ReadFromJSON() did not close the request body")
			}
		})
	}
}

func TestGenerateToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		userID string
		secret []byte
		ttl    time.Duration
	}{
		{
			name:   "typical user id",
			userID: "d4b23255-8a6a-4935-8d15-2f5690390462",
			secret: []byte("test-secret"),
			ttl:    time.Hour,
		},
		{
			name:   "empty user id still produces a parseable token",
			userID: "",
			secret: []byte("test-secret"),
			ttl:    time.Minute,
		},
		{
			name:   "very short ttl",
			userID: "u1",
			secret: []byte("another-secret"),
			ttl:    time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			before := time.Now()
			tok, err := GenerateToken(tt.userID, tt.secret, tt.ttl)
			if err != nil {
				t.Fatalf("GenerateToken() unexpected error: %v", err)
			}
			if tok == "" {
				t.Fatalf("GenerateToken() returned empty token")
			}

			var claims UserClaims
			parsed, err := jwt.ParseWithClaims(tok, &claims, func(_ *jwt.Token) (any, error) {
				return tt.secret, nil
			})
			if err != nil {
				t.Fatalf("token did not parse with the same secret: %v", err)
			}
			if !parsed.Valid {
				t.Fatalf("parsed token is not valid")
			}
			if claims.UserID != tt.userID {
				t.Errorf("claims.UserID = %q, want %q", claims.UserID, tt.userID)
			}
			if alg := parsed.Method.Alg(); alg != "HS256" {
				t.Errorf("signing algorithm = %q, want HS256 (the middleware rejects anything else)", alg)
			}

			expiry := claims.ExpiresAt.Time
			wantExpiry := before.Add(tt.ttl)
			if diff := expiry.Sub(wantExpiry); diff < -2*time.Second || diff > 2*time.Second {
				t.Errorf("expiry = %v, want close to %v (diff %v)", expiry, wantExpiry, diff)
			}

			issuedAt := claims.IssuedAt.Time
			if diff := issuedAt.Sub(before); diff < -2*time.Second || diff > 2*time.Second {
				t.Errorf("issuedAt = %v, want close to %v (diff %v)", issuedAt, before, diff)
			}
		})
	}
}

func TestGenerateToken_WrongSecretFailsToParse(t *testing.T) {
	t.Parallel()

	tok, err := GenerateToken("u1", []byte("right-secret"), time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() unexpected error: %v", err)
	}

	var claims UserClaims
	_, err = jwt.ParseWithClaims(tok, &claims, func(_ *jwt.Token) (any, error) {
		return []byte("wrong-secret"), nil
	})
	if err == nil {
		t.Fatalf("expected parsing with the wrong secret to fail, it succeeded")
	}
}

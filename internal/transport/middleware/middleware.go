package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	httpErrors "avitohvk/internal/errors"
	"avitohvk/internal/transport/utilhttp"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey int

const userIDKey ctxKey = iota

const AccessTokenCookie = "access_token"

type UserClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func NewJWTAuth(secret []byte, logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(AccessTokenCookie)
			if err != nil || cookie.Value == "" {
				logger.Warn("auth: missing access_token cookie")
				utilhttp.WriteError(w, httpErrors.ErrUnauthorized)
				return
			}

			tokenStr := cookie.Value

			var claims UserClaims
			token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected signing method")
				}
				return secret, nil
			}, jwt.WithExpirationRequired())

			if err != nil || !token.Valid {
				logger.Warn("auth: invalid token", "error", err)
				utilhttp.WriteError(w, httpErrors.ErrUnauthorized)
				return
			}

			if claims.UserID == "" {
				logger.Warn("auth: token has empty user_id")
				utilhttp.WriteError(w, httpErrors.ErrUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

// ContextWithUserID returns a copy of ctx carrying userID exactly as
// NewJWTAuth's handler would set it after successfully validating a
// request. It exists so handler-package tests can simulate an
// authenticated request without needing to sign a real JWT and drive it
// through the middleware; userIDKey is unexported, so this is the only way
// for another package's tests to construct such a context.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

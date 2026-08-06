package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	httpErrors "avitohvk/internal/errors"
	"avitohvk/internal/transport/utilhttp"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey int

const userIDKey ctxKey = iota

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
			header := r.Header.Get("Authorization")

			tokenStr, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || tokenStr == "" {
				logger.Warn("auth: missing or malformed Authorization header")
				utilhttp.WriteError(w, httpErrors.ErrUnauthorized)
				return
			}

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

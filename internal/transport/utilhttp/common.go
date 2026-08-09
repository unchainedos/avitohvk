package utilhttp

import (
	statusErrors "avitohvk/internal/errors"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type UserClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func ReadFromJSON[T any](response *http.Request) (*T, error) {
	var req T
	body, err := io.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func WriteJSON(w http.ResponseWriter, status int, data any) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(body)
	return err
}

func WriteError(w http.ResponseWriter, err error) {
	if err == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	status := http.StatusInternalServerError
	message := err.Error()
	switch {
	case errors.Is(err, statusErrors.ErrBadRequest):
		status = http.StatusBadRequest
	case errors.Is(err, statusErrors.ErrUnauthorized):
		status = http.StatusUnauthorized
	case errors.Is(err, statusErrors.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, statusErrors.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, statusErrors.ErrBadGateway):
		status = http.StatusBadGateway
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			status = http.StatusBadRequest
			message = "invalid input"
		}
	}

	if status == http.StatusInternalServerError {
		slog.Error("unhandled request error", "error", err)
		message = "internal server error"
	}

	WriteJSON(w, status, statusErrors.ErrorResponse{Message: message})
}

func GenerateToken(userID string, secret []byte, ttl time.Duration) (string, error) {
	now := time.Now()

	claims := UserClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

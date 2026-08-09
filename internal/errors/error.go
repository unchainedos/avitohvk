package error

import "errors"

var (
	ErrBadRequest   = errors.New("bad request")
	ErrUnauthorized = errors.New("unauthorized")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrBadGateway   = errors.New("bad gateway")
)

type ErrorResponse struct {
	Message string `json:"message"`
}

package handler

import (
	statusErrors "avitohvk/internal/errors"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func readFromJSON[T any](response *http.Request) (*T, error) {
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

func writeJSON(w http.ResponseWriter, status int, data any) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(body)
	return err
}

func writeError(w http.ResponseWriter, err error) {
	if err == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, statusErrors.ErrBadRequest):
		status = http.StatusBadRequest
	case errors.Is(err, statusErrors.ErrUnauthorized):
		status = http.StatusUnauthorized
	case errors.Is(err, statusErrors.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, statusErrors.ErrConflict):
		status = http.StatusConflict
	}

	writeJSON(w, status, statusErrors.ErrorResponse{Message: err.Error()})
}

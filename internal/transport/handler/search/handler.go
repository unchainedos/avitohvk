package search

import (
	"net/http"
	"strconv"

	statusErrors "avitohvk/internal/errors"
	service "avitohvk/internal/service/search"
	"avitohvk/internal/transport/utilhttp"

	"github.com/go-chi/chi/v5"
)

type SearchHandler struct {
	Svc service.SearchService
}

func New(sv service.SearchService) *SearchHandler {
	return &SearchHandler{Svc: sv}
}

func (h *SearchHandler) RegisterRoutes(r chi.Router) {
	r.Get("/search", h.Search)
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if r.URL.Query().Get("limit") == "" {
		limit = 10
		err = nil
	}
	if err != nil {
		utilhttp.WriteError(w, statusErrors.ErrBadRequest)
	}
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if r.URL.Query().Get("offset") == "" {
		offset = 0
		err = nil
	}
	if err != nil {
		utilhttp.WriteError(w, statusErrors.ErrBadRequest)
	}
	result, err := h.Svc.Search(r.Context(), query, limit, offset)
	if err != nil {
		utilhttp.WriteError(w, statusErrors.ErrBadGateway)
	}
	_ = utilhttp.WriteJSON(w, http.StatusOK, result)
}

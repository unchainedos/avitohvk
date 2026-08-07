package user

import (
	"context"
	"net/http"

	"avitohvk/internal/domain"
	"avitohvk/internal/dto"
	statusErrors "avitohvk/internal/errors"
	"avitohvk/internal/transport/middleware"
	"avitohvk/internal/transport/utilhttp"

	"github.com/go-chi/chi/v5"
)

type ItemCreator interface {
	Create(ctx context.Context, userID, id string, in dto.CreateItemRequest) (string, error)
}

type UserService interface {
	Register(ctx context.Context, username, password string) (string, error)
	Get(ctx context.Context, userID string) (domain.User, error)
	Update(ctx context.Context, actorID, userID string, username, password, email, tg, phone *string) (domain.User, error)
	Delete(ctx context.Context, actorID, userID string) error
}

type UserHandler struct {
	service     UserService
	itemService ItemCreator
}

func New(service UserService, itemService ItemCreator) *UserHandler {
	return &UserHandler{
		service:     service,
		itemService: itemService}
}

func (h *UserHandler) RegisterRoutes(r chi.Router) {
	r.Post("/user", h.Create)
	r.Get("/user", h.Get)
	r.Patch("/user", h.Update)
	r.Delete("/user", h.Delete)
	r.Post("/user/add", h.AddItem)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := utilhttp.ReadFromJSON[dto.RegisterRequest](r)
	if err != nil {
		utilhttp.WriteError(w, statusErrors.ErrBadRequest)
		return
	}
	id, err := h.service.Register(r.Context(), req.Username, req.Password)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusCreated, dto.RegisterResponse{ID: id})
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	u, err := h.service.Get(r.Context(), userID)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusOK, toUserResponse(u))
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	req, err := utilhttp.ReadFromJSON[dto.UpdateUserRequest](r)
	if err != nil {
		utilhttp.WriteError(w, statusErrors.ErrBadRequest)
		return
	}
	u, err := h.service.Update(
		r.Context(),
		userID,
		userID,
		req.Username,
		req.Password,
		req.Email,
		req.TG,
		req.PhoneNumber,
	)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusOK, toUserResponse(u))
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	if err := h.service.Delete(r.Context(), userID, userID); err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		utilhttp.WriteError(w, statusErrors.ErrUnauthorized)
		return
	}
	req, err := utilhttp.ReadFromJSON[dto.CreateItemRequest](r)
	if err != nil {
		utilhttp.WriteError(w, statusErrors.ErrBadRequest)
		return
	}
	id, err := h.itemService.Create(r.Context(), userID, "", *req)
	if err != nil {
		utilhttp.WriteError(w, err)
		return
	}
	_ = utilhttp.WriteJSON(w, http.StatusCreated, dto.CreateItemResponse{ID: id})
}

func toUserResponse(u domain.User) dto.UserResponse {
	return dto.UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		TG:          u.TG,
		PhoneNumber: u.PhoneNumber,
		CreatedAt:   u.CreatedAt,
	}
}

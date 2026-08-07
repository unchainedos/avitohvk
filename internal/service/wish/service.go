package wish

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"avitohvk/internal/domain"
	statusErrors "avitohvk/internal/errors"
	wishrepo "avitohvk/internal/repository/wish"
)

type Repository interface {
	Create(ctx context.Context, wish domain.Wish) (string, error)
	GetByID(ctx context.Context, id string) (domain.Wish, error)
	ListByUserID(ctx context.Context, userID string) ([]domain.Wish, error)
	ListItemsByUserID(ctx context.Context, userID string) ([]domain.Item, error)
	Update(ctx context.Context, id string, wish domain.Wish) error
	Delete(ctx context.Context, id string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID, id, title string, description *string) (string, error) {
	title = strings.TrimSpace(title)
	if userID == "" {
		return "", statusErrors.ErrUnauthorized
	}
	if title == "" {
		return "", fmt.Errorf("%w: title required", statusErrors.ErrBadRequest)
	}

	wish := domain.Wish{
		ID:          id,
		UserID:      userID,
		Title:       title,
		Description: description,
	}

	newID, err := s.repo.Create(ctx, wish)
	if err != nil {
		return "", err
	}
	return newID, nil
}

func (s *Service) Get(ctx context.Context, id string) (domain.Wish, error) {
	wish, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, wishrepo.ErrNotFound) {
			return domain.Wish{}, statusErrors.ErrNotFound
		}
		return domain.Wish{}, err
	}
	return wish, nil
}

func (s *Service) ListByUser(ctx context.Context, userID string) ([]domain.Wish, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *Service) ListItemsByUser(ctx context.Context, userID string) ([]domain.Item, error) {
	return s.repo.ListItemsByUserID(ctx, userID)
}

func (s *Service) Update(ctx context.Context, actorID, wishID string, title, description *string) (domain.Wish, error) {
	if actorID == "" {
		return domain.Wish{}, statusErrors.ErrUnauthorized
	}

	current, err := s.repo.GetByID(ctx, wishID)
	if err != nil {
		if errors.Is(err, wishrepo.ErrNotFound) {
			return domain.Wish{}, statusErrors.ErrNotFound
		}
		return domain.Wish{}, err
	}

	if current.UserID != actorID {
		return domain.Wish{}, statusErrors.ErrUnauthorized
	}

	if title != nil {
		t := strings.TrimSpace(*title)
		if t == "" {
			return domain.Wish{}, fmt.Errorf("%w: title cannot be empty", statusErrors.ErrBadRequest)
		}
		current.Title = t
	}
	if description != nil {
		current.Description = description
	}

	if err := s.repo.Update(ctx, wishID, current); err != nil {
		if errors.Is(err, wishrepo.ErrNotFound) {
			return domain.Wish{}, statusErrors.ErrNotFound
		}
		return domain.Wish{}, err
	}
	return s.Get(ctx, wishID)
}

func (s *Service) Delete(ctx context.Context, actorID, wishID string) error {
	if actorID == "" {
		return statusErrors.ErrUnauthorized
	}

	current, err := s.repo.GetByID(ctx, wishID)
	if err != nil {
		if errors.Is(err, wishrepo.ErrNotFound) {
			return statusErrors.ErrNotFound
		}
		return err
	}

	if current.UserID != actorID {
		return statusErrors.ErrUnauthorized
	}

	if err := s.repo.Delete(ctx, wishID); err != nil {
		if errors.Is(err, wishrepo.ErrNotFound) {
			return statusErrors.ErrNotFound
		}
		return err
	}
	return nil
}

package item

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"avitohvk/internal/domain"
	"avitohvk/internal/dto"
	statusErrors "avitohvk/internal/errors"
	itemrepo "avitohvk/internal/repository/item"
)

type Repository interface {
	Create(ctx context.Context, item domain.Item) (string, error)
	GetByID(ctx context.Context, id string) (domain.Item, error)
	ListByUserID(ctx context.Context, userID string) ([]domain.Item, error)
	Update(ctx context.Context, id string, item domain.Item) error
	Delete(ctx context.Context, id string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID, id string, in dto.CreateItemRequest) (string, error) {
	title := strings.TrimSpace(in.Title)
	if userID == "" {
		return "", statusErrors.ErrUnauthorized
	}
	if title == "" {
		return "", fmt.Errorf("%w: title required", statusErrors.ErrBadRequest)
	}
	quantity := in.Quantity
	if quantity <= 0 {
		quantity = 1
	}
	item := domain.Item{
		ID:          id,
		AuthorID:    userID,
		HolderID:    userID,
		Title:       title,
		Description: in.Description,
		ImageURL:    in.ImageURL,
		Category:    in.Category,
		Unit:        in.Unit,
		Quantity:    quantity,
		IsLocked:    false,
	}
	return s.repo.Create(ctx, item)
}

func (s *Service) Get(ctx context.Context, id string) (domain.Item, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, itemrepo.ErrNotFound) {
			return domain.Item{}, statusErrors.ErrNotFound
		}
		return domain.Item{}, err
	}
	return item, nil
}

func (s *Service) ListByUser(ctx context.Context, userID string) ([]domain.Item, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *Service) Update(ctx context.Context, actorID, itemID string, in dto.UpdateItemRequest) (domain.Item, error) {
	if actorID == "" {
		return domain.Item{}, statusErrors.ErrUnauthorized
	}
	current, err := s.repo.GetByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, itemrepo.ErrNotFound) {
			return domain.Item{}, statusErrors.ErrNotFound
		}
		return domain.Item{}, err
	}
	if current.AuthorID != actorID {
		return domain.Item{}, statusErrors.ErrUnauthorized
	}
	if current.IsLocked {
		return domain.Item{}, fmt.Errorf("%w: item is locked", statusErrors.ErrConflict)
	}
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" {
			return domain.Item{}, fmt.Errorf("%w: title cannot be empty", statusErrors.ErrBadRequest)
		}
		current.Title = t
	}
	if in.Description != nil {
		current.Description = in.Description
	}
	if in.ImageURL != nil {
		current.ImageURL = in.ImageURL
	}
	if in.Category != nil {
		current.Category = in.Category
	}
	if in.Unit != nil {
		current.Unit = in.Unit
	}
	if in.Quantity != nil {
		if *in.Quantity <= 0 {
			return domain.Item{}, fmt.Errorf("%w: quantity must be > 0", statusErrors.ErrBadRequest)
		}
		current.Quantity = *in.Quantity
	}
	if err := s.repo.Update(ctx, itemID, current); err != nil {
		if errors.Is(err, itemrepo.ErrNotFound) {
			return domain.Item{}, statusErrors.ErrNotFound
		}
		return domain.Item{}, err
	}
	return s.Get(ctx, itemID)
}

func (s *Service) Delete(ctx context.Context, actorID, itemID string) error {
	if actorID == "" {
		return statusErrors.ErrUnauthorized
	}

	current, err := s.repo.GetByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, itemrepo.ErrNotFound) {
			return statusErrors.ErrNotFound
		}
		return err
	}

	if current.AuthorID != actorID {
		return statusErrors.ErrUnauthorized
	}
	if current.IsLocked {
		return fmt.Errorf("%w: item is locked", statusErrors.ErrConflict)
	}

	if err := s.repo.Delete(ctx, itemID); err != nil {
		if errors.Is(err, itemrepo.ErrNotFound) {
			return statusErrors.ErrNotFound
		}
		return err
	}
	return nil
}

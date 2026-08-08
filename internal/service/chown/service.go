package chown

import (
	"context"
	"errors"
	"fmt"

	"avitohvk/internal/domain"
	"avitohvk/internal/dto"
	statusErrors "avitohvk/internal/errors"
	chownrepo "avitohvk/internal/repository/chown"
)

type Repository interface {
	Chown(ctx context.Context, actorID, itemID string, offers []domain.OfferedItem) (domain.Chown, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Chown(ctx context.Context, actorID, itemID string, req dto.ChownRequest) (dto.ChownResponse, error) {
	if itemID == "" {
		return dto.ChownResponse{}, fmt.Errorf("%w: item_id required", statusErrors.ErrBadRequest)
	}
	if len(req.OfferedItems) == 0 {
		return dto.ChownResponse{}, fmt.Errorf("%w: offered_items required", statusErrors.ErrBadRequest)
	}

	offers := make([]domain.OfferedItem, 0, len(req.OfferedItems))
	for _, oi := range req.OfferedItems {
		if oi.ItemID == "" {
			return dto.ChownResponse{}, fmt.Errorf("%w: item_id required for every offered item", statusErrors.ErrBadRequest)
		}
		if oi.Quantity <= 0 {
			return dto.ChownResponse{}, fmt.Errorf("%w: quantity must be positive", statusErrors.ErrBadRequest)
		}
		if oi.ItemID == itemID {
			return dto.ChownResponse{}, fmt.Errorf("%w: cannot offer the item you are chowning", statusErrors.ErrBadRequest)
		}
		offers = append(offers, domain.OfferedItem{ItemID: oi.ItemID, Quantity: oi.Quantity})
	}

	result, err := s.repo.Chown(ctx, actorID, itemID, offers)
	if err != nil {
		if errors.Is(err, chownrepo.ErrItemNotFound) {
			return dto.ChownResponse{}, fmt.Errorf("%w: item not found", statusErrors.ErrNotFound)
		}
		if errors.Is(err, chownrepo.ErrNotItemHolder) {
			return dto.ChownResponse{}, fmt.Errorf("%w: you do not hold this item", statusErrors.ErrConflict)
		}
		if errors.Is(err, chownrepo.ErrRecipientNotFound) {
			return dto.ChownResponse{}, fmt.Errorf("%w: no one wishes for this item", statusErrors.ErrNotFound)
		}
		if errors.Is(err, chownrepo.ErrOwnItem) {
			return dto.ChownResponse{}, fmt.Errorf("%w: cannot chown your own item", statusErrors.ErrBadRequest)
		}
		if errors.Is(err, chownrepo.ErrItemLocked) {
			return dto.ChownResponse{}, fmt.Errorf("%w: someone else already has exclusive rights to this item", statusErrors.ErrConflict)
		}
		return dto.ChownResponse{}, err
	}

	return toChownResponse(result), nil
}

func toChownResponse(c domain.Chown) dto.ChownResponse {
	hops := make([]dto.ChownHop, 0, len(c.Hops))
	for _, h := range c.Hops {
		hops = append(hops, dto.ChownHop{ItemID: h.ItemID, Quantity: h.Quantity, ToUserID: h.ToUserID})
	}
	return dto.ChownResponse{
		ItemID:     c.ItemID,
		FromUserID: c.FromUserID,
		Hops:       hops,
		CreatedAt:  c.CreatedAt,
	}
}

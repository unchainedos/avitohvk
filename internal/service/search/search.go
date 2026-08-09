package search

import (
	"context"

	dto "avitohvk/internal/dto"
	rp "avitohvk/internal/repository/search"
)

type SearchService struct {
	repo rp.SearchRepo
}

func NewService(r rp.SearchRepo) SearchService {
	return SearchService{repo: r}
}

func (s *SearchService) Search(ctx context.Context, query string, limit, offset int) ([]dto.ItemResponse, error) {
	return s.repo.SearchItems(ctx, query, limit, offset)
}

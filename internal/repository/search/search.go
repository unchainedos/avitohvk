package search

import (
	"avitohvk/internal/dto"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SearchRepo struct {
	pool *pgxpool.Pool
}

func NewRepository(ol *pgxpool.Pool) SearchRepo {
	return SearchRepo{pool: ol}
}

func (s *SearchRepo) SearchItems(ctx context.Context, query string, limit, offset int) ([]dto.ItemResponse, error) {
	var dbq string
	var args []interface{}

	if query == "" {
		// Если запрос пустой - просто возвращаем все записи
		dbq = `
			SELECT 
			    id, author_id, holder_id, title, description,
			    image_url, category, unit, quantity, is_locked, created_at
			FROM items
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}
	} else {
		// Если есть поисковый запрос - используем полнотекстовый поиск
		dbq = `
			SELECT 
			    id, author_id, holder_id, title, description,
			    image_url, category, unit, quantity, is_locked, created_at
			FROM items
			WHERE search_vector @@ plainto_tsquery('russian', $1)
			ORDER BY 
			    ts_rank(search_vector, plainto_tsquery('russian', $1)) DESC,
			    created_at DESC
			LIMIT $2 OFFSET $3`
		args = []interface{}{query, limit, offset}
	}

	rows, err := s.pool.Query(ctx, dbq, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []dto.ItemResponse
	for rows.Next() {
		var item dto.ItemResponse
		err = rows.Scan(
			&item.ID,
			&item.AuthorID,
			&item.HolderID,
			&item.Title,
			&item.Description,
			&item.ImageURL,
			&item.Category,
			&item.Unit,
			&item.Quantity,
			&item.IsLocked,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

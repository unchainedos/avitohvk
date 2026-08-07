package item

import (
	"context"
	"errors"

	"avitohvk/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("item not found")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, item domain.Item) (string, error) {
	if item.ID != "" {
		const q = `
			INSERT INTO items (
				id, author_id, holder_id, title, description,
				image_url, category, unit, quantity, is_locked
			) VALUES (
				$1::uuid, $2::uuid, $3::uuid, $4, $5,
				$6, $7, $8, $9, $10
			)
			RETURNING id::text
		`
		var id string
		err := r.pool.QueryRow(ctx, q,
			item.ID,
			item.AuthorID,
			item.HolderID,
			item.Title,
			item.Description,
			item.ImageURL,
			item.Category,
			item.Unit,
			item.Quantity,
			item.IsLocked,
		).Scan(&id)
		return id, err
	}

	const q = `
		INSERT INTO items (
			author_id, holder_id, title, description,
			image_url, category, unit, quantity, is_locked
		) VALUES (
			$1::uuid, $2::uuid, $3, $4,
			$5, $6, $7, $8, $9
		)
		RETURNING id::text
	`
	var id string
	err := r.pool.QueryRow(ctx, q,
		item.AuthorID,
		item.HolderID,
		item.Title,
		item.Description,
		item.ImageURL,
		item.Category,
		item.Unit,
		item.Quantity,
		item.IsLocked,
	).Scan(&id)
	return id, err
}

func (r *Repository) GetByID(ctx context.Context, id string) (domain.Item, error) {
	const q = `
		SELECT id::text, author_id::text, holder_id::text,
		       title, description, image_url, category, unit,
		       quantity, is_locked, created_at
		FROM items
		WHERE id = $1::uuid
	`
	var item domain.Item
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&item.ID, &item.AuthorID, &item.HolderID,
		&item.Title, &item.Description, &item.ImageURL, &item.Category, &item.Unit,
		&item.Quantity, &item.IsLocked, &item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Item{}, ErrNotFound
	}
	if err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

func (r *Repository) ListByUserID(ctx context.Context, userID string) ([]domain.Item, error) {
	const q = `
		SELECT id::text, author_id::text, holder_id::text,
		       title, description, image_url, category, unit,
		       quantity, is_locked, created_at
		FROM items
		WHERE holder_id = $1::uuid
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Item, 0)
	for rows.Next() {
		var item domain.Item
		err := rows.Scan(
			&item.ID, &item.AuthorID, &item.HolderID,
			&item.Title, &item.Description, &item.ImageURL, &item.Category, &item.Unit,
			&item.Quantity, &item.IsLocked, &item.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Update(ctx context.Context, id string, item domain.Item) error {
	const q = `
		UPDATE items SET
			title = $1,
			description = $2,
			image_url = $3,
			category = $4,
			unit = $5,
			quantity = $6
		WHERE id = $7::uuid
	`
	tag, err := r.pool.Exec(ctx, q,
		item.Title,
		item.Description,
		item.ImageURL,
		item.Category,
		item.Unit,
		item.Quantity,
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM items WHERE id = $1::uuid`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

package wish

import (
	"avitohvk/internal/domain"
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("wish not found")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, wish domain.Wish) (string, error) {
	if wish.ID != "" {
		const q = `
			INSERT INTO wishes (id, user_id, title, description)
			VALUES ($1::uuid, $2::uuid, $3, $4)
			RETURNING id::text
		`
		var id string
		err := r.pool.QueryRow(ctx, q,
			wish.ID, wish.UserID, wish.Title, wish.Description,
		).Scan(&id)
		return id, err
	}
	const q = `
		INSERT INTO wishes (user_id, title, description)
		VALUES ($1::uuid, $2, $3)
		RETURNING id::text
	`
	var id string
	err := r.pool.QueryRow(ctx, q,
		wish.UserID, wish.Title, wish.Description,
	).Scan(&id)
	return id, err
}

func (r *Repository) GetByID(ctx context.Context, id string) (domain.Wish, error) {
	const q = `
		SELECT id::text, user_id::text, title, description, created_at
		FROM wishes
		WHERE id = $1::uuid
	`
	var wish domain.Wish
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&wish.ID, &wish.UserID, &wish.Title, &wish.Description, &wish.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Wish{}, ErrNotFound
	}
	if err != nil {
		return domain.Wish{}, err
	}
	return wish, nil
}

func (r *Repository) ListByUserID(ctx context.Context, userID string) ([]domain.Wish, error) {
	const q = `
		SELECT id::text, user_id::text, title, description, created_at
		FROM wishes
		WHERE user_id = $1::uuid
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	wishes := make([]domain.Wish, 0)
	for rows.Next() {
		var wish domain.Wish
		err := rows.Scan(
			&wish.ID, &wish.UserID, &wish.Title, &wish.Description, &wish.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		wishes = append(wishes, wish)
	}
	return wishes, rows.Err()
}

func (r *Repository) ListItemsByUserID(ctx context.Context, userID string) ([]domain.Item, error) {
	const q = `
		SELECT DISTINCT
			i.id::text, i.author_id::text, i.holder_id::text,
			i.title, i.description, i.image_url, i.category, i.unit,
			i.quantity, i.is_locked, i.created_at
		FROM wishes w
		JOIN wish_items wi ON wi.wish_id = w.id
		JOIN items i ON i.id = wi.item_id
		WHERE w.user_id = $1::uuid
		ORDER BY i.created_at DESC
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

func (r *Repository) Update(ctx context.Context, id string, wish domain.Wish) error {
	const q = `
		UPDATE wishes SET
			title = $1,
			description = $2
		WHERE id = $3::uuid
	`
	tag, err := r.pool.Exec(ctx, q, wish.Title, wish.Description, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM wishes WHERE id = $1::uuid`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

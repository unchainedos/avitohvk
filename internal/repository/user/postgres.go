package user

import (
	"avitohvk/internal/domain"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrAlreadyExists = errors.New("user already exists")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}
func (r *Repository) Create(ctx context.Context, username, passwordHash string) (string, error) {
	const q = `
		INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		RETURNING id::text
	`
	var id string
	err := r.pool.QueryRow(ctx, q, username, passwordHash).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return "", ErrAlreadyExists
		}
		return "", err
	}
	return id, nil
}
func (r *Repository) GetByUsername(ctx context.Context, username string) (domain.User, error) {
	const q = `
		SELECT id::text, username, password_hash, email, tg, phone_number, is_banned, created_at
		FROM users
		WHERE username = $1
	`
	return r.scanUser(r.pool.QueryRow(ctx, q, username))
}
func (r *Repository) GetByID(ctx context.Context, id string) (domain.User, error) {
	const q = `
		SELECT id::text, username, password_hash, email, tg, phone_number, is_banned, created_at
		FROM users
		WHERE id = $1::uuid
	`
	return r.scanUser(r.pool.QueryRow(ctx, q, id))
}
func (r *Repository) Update(ctx context.Context, id string, upd domain.UserUpdate) error {
	sets := make([]string, 0, 5)
	args := make([]any, 0, 6)
	n := 1
	if upd.Username != nil {
		sets = append(sets, fmt.Sprintf("username = $%d", n))
		args = append(args, *upd.Username)
		n++
	}
	if upd.PasswordHash != nil {
		sets = append(sets, fmt.Sprintf("password_hash = $%d", n))
		args = append(args, *upd.PasswordHash)
		n++
	}
	if upd.Email != nil {
		sets = append(sets, fmt.Sprintf("email = $%d", n))
		args = append(args, *upd.Email)
		n++
	}
	if upd.TG != nil {
		sets = append(sets, fmt.Sprintf("tg = $%d", n))
		args = append(args, *upd.TG)
		n++
	}
	if upd.PhoneNumber != nil {
		sets = append(sets, fmt.Sprintf("phone_number = $%d", n))
		args = append(args, *upd.PhoneNumber)
		n++
	}
	if len(sets) == 0 {
		return nil
	}
	q := fmt.Sprintf(`UPDATE users SET %s WHERE id = $%d::uuid`, strings.Join(sets, ", "), n)
	args = append(args, id)
	tag, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyExists
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *Repository) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM users WHERE id = $1::uuid`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *Repository) scanUser(row pgx.Row) (domain.User, error) {
	var u domain.User
	err := row.Scan(
		&u.ID, &u.Username, &u.PasswordHash,
		&u.Email, &u.TG, &u.PhoneNumber,
		&u.IsBanned, &u.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	return u, nil
}

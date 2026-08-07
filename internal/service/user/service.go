package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"avitohvk/internal/domain"
	statusErrors "avitohvk/internal/errors"
	userrepo "avitohvk/internal/repository/user"
	"avitohvk/internal/transport/utilhttp"

	"golang.org/x/crypto/bcrypt"
)

type Repository interface {
	Create(ctx context.Context, username, passwordHash string) (string, error)
	GetByUsername(ctx context.Context, username string) (domain.User, error)
	GetByID(ctx context.Context, id string) (domain.User, error)
	Update(ctx context.Context, id string, upd domain.UserUpdate) error
	Delete(ctx context.Context, id string) error
}

type Service struct {
	repo      Repository
	jwtSecret []byte
	jwtTTL    time.Duration
}

func NewService(repo Repository, jwtSecret []byte, jwtTTL time.Duration) *Service {
	if jwtTTL == 0 {
		jwtTTL = 24 * time.Hour
	}
	return &Service{repo: repo, jwtSecret: jwtSecret, jwtTTL: jwtTTL}
}

func (s *Service) Register(ctx context.Context, username, password string) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return "", fmt.Errorf("%w: username and password required", statusErrors.ErrBadRequest)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	id, err := s.repo.Create(ctx, username, string(hash))
	if err != nil {
		if errors.Is(err, userrepo.ErrAlreadyExists) {
			return "", fmt.Errorf("%w: username already taken", statusErrors.ErrConflict)
		}
		return "", err
	}
	return id, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return "", fmt.Errorf("%w: username and password required", statusErrors.ErrBadRequest)
	}

	u, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, userrepo.ErrNotFound) {
			return "", statusErrors.ErrUnauthorized
		}
		return "", err
	}
	if u.IsBanned {
		return "", statusErrors.ErrUnauthorized
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", statusErrors.ErrUnauthorized
	}

	return utilhttp.GenerateToken(u.ID, s.jwtSecret, s.jwtTTL)
}

func (s *Service) Get(ctx context.Context, userID string) (domain.User, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, userrepo.ErrNotFound) {
			return domain.User{}, statusErrors.ErrNotFound
		}
		return domain.User{}, err
	}
	return u, nil
}

func (s *Service) Update(ctx context.Context, actorID, userID string, username, password, email, tg, phone *string) (domain.User, error) {
	if actorID != userID {
		return domain.User{}, statusErrors.ErrUnauthorized
	}

	upd := domain.UserUpdate{
		Username:    username,
		Email:       email,
		TG:          tg,
		PhoneNumber: phone,
	}
	if password != nil {
		if *password == "" {
			return domain.User{}, fmt.Errorf("%w: password cannot be empty", statusErrors.ErrBadRequest)
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
		if err != nil {
			return domain.User{}, err
		}
		h := string(hash)
		upd.PasswordHash = &h
	}

	if err := s.repo.Update(ctx, userID, upd); err != nil {
		if errors.Is(err, userrepo.ErrNotFound) {
			return domain.User{}, statusErrors.ErrNotFound
		}
		if errors.Is(err, userrepo.ErrAlreadyExists) {
			return domain.User{}, statusErrors.ErrConflict
		}
		return domain.User{}, err
	}
	return s.Get(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, actorID, userID string) error {
	if actorID != userID {
		return statusErrors.ErrUnauthorized
	}
	if err := s.repo.Delete(ctx, userID); err != nil {
		if errors.Is(err, userrepo.ErrNotFound) {
			return statusErrors.ErrNotFound
		}
		return err
	}
	return nil
}

package dto

import "time"

type CreateWishRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
}

type UpdateWishRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

type WishResponse struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateWishResponse struct {
	ID string `json:"id"`
}

package dto

import "time"

type CreateItemRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	ImageURL    *string `json:"image_url"`
	Category    *string `json:"category"`
	Unit        *string `json:"unit"`
	Quantity    float64 `json:"quantity"`
}

type UpdateItemRequest struct {
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	ImageURL    *string  `json:"image_url"`
	Category    *string  `json:"category"`
	Unit        *string  `json:"unit"`
	Quantity    *float64 `json:"quantity"`
}

type ItemResponse struct {
	ID          string    `json:"id"`
	AuthorID    string    `json:"author_id"`
	HolderID    string    `json:"holder_id"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	ImageURL    *string   `json:"image_url,omitempty"`
	Category    *string   `json:"category,omitempty"`
	Unit        *string   `json:"unit,omitempty"`
	Quantity    float64   `json:"quantity"`
	IsLocked    bool      `json:"is_locked"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateItemResponse struct {
	ID string `json:"id"`
}

package domain

import "time"

type Item struct {
	ID          string
	AuthorID    string
	HolderID    string
	Title       string
	Description *string
	ImageURL    *string
	Category    *string
	Unit        *string
	Quantity    float64
	IsLocked    bool
	CreatedAt   time.Time
}

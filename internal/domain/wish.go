package domain

import "time"

type Wish struct {
	ID          string
	UserID      string
	Title       string
	Description *string
	CreatedAt   time.Time
}

type WishItem struct {
	WishID string
	ItemID string
}

package domain

import "time"

type OfferedItem struct {
	ItemID   string
	Quantity float64
}

type ChownHop struct {
	ItemID   string
	Quantity float64
	ToUserID string
}

type Chown struct {
	ItemID     string
	FromUserID string
	Hops       []ChownHop
	CreatedAt  time.Time
}

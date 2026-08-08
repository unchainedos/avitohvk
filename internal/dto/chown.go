package dto

import "time"

type OfferedItem struct {
	ItemID   string  `json:"item_id"`
	Quantity float64 `json:"quantity"`
}

type ChownRequest struct {
	OfferedItems []OfferedItem `json:"offered_items"`
}

type ChownHop struct {
	ItemID   string  `json:"item_id"`
	Quantity float64 `json:"quantity"`
	ToUserID string  `json:"to_user_id"`
}

type ChownResponse struct {
	ItemID     string     `json:"item_id"`
	FromUserID string     `json:"from_user_id"`
	Hops       []ChownHop `json:"hops"`
	CreatedAt  time.Time  `json:"created_at"`
}

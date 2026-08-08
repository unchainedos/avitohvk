package domain

import "time"

type DealStatus string

const (
	DealStatusPending   DealStatus = "PENDING"
	DealStatusConfirmed DealStatus = "CONFIRMED"
	DealStatusCompleted DealStatus = "COMPLETED"
	DealStatusCancelled DealStatus = "CANCELLED"
)

type Deal struct {
	ID           string
	RootItemID   string
	CreatorID    string
	Status       DealStatus
	Participants int
	DeadlineAt   time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

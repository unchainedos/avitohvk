package domain

import "time"

type DealStatus string

const (
	DealStatusPending   DealStatus = "PENDING"
	DealStatusConfirmed DealStatus = "CONFIRMED"
	DealStatusCompleted DealStatus = "COMPLETED"
	DealStatusCanceled  DealStatus = "CANCELLED"
)

type Deal struct {
	ID                string
	RootItemID        string
	CreatorID         string
	Status            DealStatus
	NegotiationWindow time.Duration
	DeadlineAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

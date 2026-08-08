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
	ID                string
	RootItemID        string
	CreatorID         string
	Status            DealStatus
	Participants      int
	NegotiationWindow time.Duration
	// DeadlineAt is nil until the participant offering the root item joins the chain;
	// only then does the negotiation window between the creator and them start ticking.
	DeadlineAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

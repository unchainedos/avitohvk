package domain

import "time"

type ProposalStatus string

const (
	ProposalStatusPending             ProposalStatus = "PENDING"
	ProposalStatusAccepted            ProposalStatus = "ACCEPTED"
	ProposalStatusDeclined            ProposalStatus = "DECLINED"
	ProposalStatusWaitingRequiredUser ProposalStatus = "WAITING_FOR_THE_REQUIRED_USER"
)

type Proposal struct {
	DealID        string
	TransactionID string
	ParticipantID string
	ItemID        string
	ToUserID      string
	Quantity      float64
	Status        ProposalStatus
	UpdatedAt     time.Time
}

type ProposalUpdate struct {
	ItemID   *string
	Quantity *float64
}

type ItemTransfer struct {
	ItemID   string
	ToUserID string
}

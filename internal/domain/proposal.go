package domain

import "time"

type ProposalStatus string

const (
	ProposalStatusPending             ProposalStatus = "PENDING"
	ProposalStatusAccepted            ProposalStatus = "ACCEPTED"
	ProposalStatusDeclined            ProposalStatus = "DECLINED"
	ProposalStatusWaitingRequiredUser ProposalStatus = "WAITING_FOR_THE_REQUIRED_USER"
)

// Proposal is one participant's link in a chain deal (chain_deal_transactions),
// together with the underlying transfer fact it points to (transactions).
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

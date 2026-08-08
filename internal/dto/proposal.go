package dto

import "time"

type CreateDealRequest struct {
	RootItemID   string  `json:"root_item_id"`
	ItemID       string  `json:"item_id"`
	Quantity     float64 `json:"quantity"`
	Participants int     `json:"participants"`
}

type CreateProposalRequest struct {
	ItemID   string  `json:"item_id"`
	Quantity float64 `json:"quantity"`
}

type UpdateProposalRequest struct {
	ItemID   *string  `json:"item_id"`
	Quantity *float64 `json:"quantity"`
}

type ProposalResponse struct {
	DealID        string    `json:"deal_id"`
	TransactionID string    `json:"transaction_id"`
	ParticipantID string    `json:"participant_id"`
	ItemID        string    `json:"item_id"`
	ToUserID      string    `json:"to_user_id"`
	Quantity      float64   `json:"quantity"`
	Status        string    `json:"status"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ProposalListResponse struct {
	Proposals []ProposalResponse `json:"proposals"`
}

type DealResponse struct {
	ID                       string     `json:"id"`
	RootItemID               string     `json:"root_item_id"`
	CreatorID                string     `json:"creator_id"`
	Status                   string     `json:"status"`
	Participants             int        `json:"participants"`
	NegotiationWindowSeconds int64      `json:"negotiation_window_seconds"`
	DeadlineAt               *time.Time `json:"deadline_at"`
	CreatedAt                time.Time  `json:"created_at"`
}

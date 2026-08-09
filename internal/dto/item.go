package dto

import (
	"encoding/json"
	"strconv"
	"time"
)

type FlexFloat64 float64

func (f FlexFloat64) Float64() float64 { return float64(f) }

func (f *FlexFloat64) UnmarshalJSON(b []byte) error {
	if string(b) == "null" || len(b) == 0 {
		*f = 0
		return nil
	}
	var n float64
	if err := json.Unmarshal(b, &n); err == nil {
		*f = FlexFloat64(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*f = FlexFloat64(n)
	return nil
}

type CreateItemRequest struct {
	Title       string      `json:"title"`
	Description *string     `json:"description"`
	ImageURL    *string     `json:"image_url"`
	Category    *string     `json:"category"`
	Unit        *string     `json:"unit"`
	Quantity    FlexFloat64 `json:"quantity"`
}

type UpdateItemRequest struct {
	Title       *string      `json:"title"`
	Description *string      `json:"description"`
	ImageURL    *string      `json:"image_url"`
	Category    *string      `json:"category"`
	Unit        *string      `json:"unit"`
	Quantity    *FlexFloat64 `json:"quantity"`
}

type ItemResponse struct {
	ID          string    `json:"id"`
	AuthorID    string    `json:"author_id"`
	HolderID    string    `json:"holder_id"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	ImageURL    *string   `json:"image_url,omitempty"`
	Category    *string   `json:"category,omitempty"`
	Unit        *string   `json:"unit,omitempty"`
	Quantity    float64   `json:"quantity"`
	IsLocked    bool      `json:"is_locked"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateItemResponse struct {
	ID string `json:"id"`
}

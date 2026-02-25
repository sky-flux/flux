package flux

import "time"

// ReviewLog records a single review event for a card.
type ReviewLog struct {
	CardID         int64     `json:"card_id"`
	Rating         Rating    `json:"rating"`
	ReviewDatetime time.Time `json:"review_datetime"`
	ReviewDuration *int      `json:"review_duration,omitempty"` // milliseconds, optional.
}

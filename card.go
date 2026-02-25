package flux

import "time"

// Card represents a flashcard with its scheduling state.
type Card struct {
	CardID     int64      `json:"card_id"`
	State      State      `json:"state"`
	Step       *int       `json:"step"`       // nil when State=Review.
	Stability  *float64   `json:"stability"`  // nil before first review.
	Difficulty *float64   `json:"difficulty"` // nil before first review.
	Due        time.Time  `json:"due"`
	LastReview *time.Time `json:"last_review"` // nil before first review.
}

// NewCard creates a new card in the Learning state with the given ID.
// Due is set to now (immediately reviewable).
func NewCard(id int64) Card {
	step := 0
	return Card{
		CardID: id,
		State:  Learning,
		Step:   &step,
		Due:    time.Now(),
	}
}

// clone returns a deep copy of the card. Pointer fields are copied by value.
func (c Card) clone() Card {
	out := c
	if c.Step != nil {
		v := *c.Step
		out.Step = &v
	}
	if c.Stability != nil {
		v := *c.Stability
		out.Stability = &v
	}
	if c.Difficulty != nil {
		v := *c.Difficulty
		out.Difficulty = &v
	}
	if c.LastReview != nil {
		v := *c.LastReview
		out.LastReview = &v
	}
	return out
}

func (c *Card) setStability(s float64) {
	c.Stability = &s
}

func (c *Card) setDifficulty(d float64) {
	c.Difficulty = &d
}

func (c *Card) setStep(step int) {
	c.Step = &step
}

func (c *Card) clearStep() {
	c.Step = nil
}

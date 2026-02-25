package flux

import (
	"encoding"
	"encoding/json"
	"fmt"
)

// Rating represents the user's assessment of recall quality.
type Rating int

const (
	Again Rating = iota + 1 // Complete failure to recall.
	Hard                    // Recalled with significant difficulty.
	Good                    // Recalled with some effort.
	Easy                    // Recalled effortlessly.
)

var (
	ratingNames = [...]string{Again: "Again", Hard: "Hard", Good: "Good", Easy: "Easy"}
	ratingByName = map[string]Rating{
		"Again": Again,
		"Hard":  Hard,
		"Good":  Good,
		"Easy":  Easy,
	}
)

// Compile-time interface checks.
var (
	_ fmt.Stringer             = Rating(0)
	_ json.Marshaler           = Rating(0)
	_ json.Unmarshaler         = (*Rating)(nil)
	_ encoding.TextMarshaler   = Rating(0)
	_ encoding.TextUnmarshaler = (*Rating)(nil)
)

// String returns the name of the rating ("Again", "Hard", "Good", "Easy").
// For invalid values it returns "Rating(n)".
func (r Rating) String() string {
	if r.IsValid() {
		return ratingNames[r]
	}
	return fmt.Sprintf("Rating(%d)", int(r))
}

// IsValid reports whether r is a valid rating (Again through Easy).
func (r Rating) IsValid() bool {
	return r >= Again && r <= Easy
}

// MarshalText implements encoding.TextMarshaler.
func (r Rating) MarshalText() ([]byte, error) {
	if !r.IsValid() {
		return nil, fmt.Errorf("%w: %d", ErrInvalidRating, int(r))
	}
	return []byte(ratingNames[r]), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (r *Rating) UnmarshalText(text []byte) error {
	v, ok := ratingByName[string(text)]
	if !ok {
		return fmt.Errorf("%w: %q", ErrInvalidRating, text)
	}
	*r = v
	return nil
}

// MarshalJSON implements json.Marshaler. Rating serializes as a JSON string.
func (r Rating) MarshalJSON() ([]byte, error) {
	text, err := r.MarshalText()
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(text))
}

// UnmarshalJSON implements json.Unmarshaler. Expects a JSON string.
func (r *Rating) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidRating, data)
	}
	return r.UnmarshalText([]byte(s))
}

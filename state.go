package flux

import (
	"encoding"
	"encoding/json"
	"fmt"
)

// State represents the learning stage of a card.
type State int

const (
	Learning   State = iota + 1 // New card, in initial learning.
	Review                      // Entered long-term review cycle.
	Relearning                  // Forgotten, relearning.
)

var (
	stateNames = [...]string{Learning: "Learning", Review: "Review", Relearning: "Relearning"}
	stateByName = map[string]State{
		"Learning":   Learning,
		"Review":     Review,
		"Relearning": Relearning,
	}
)

// Compile-time interface checks.
var (
	_ fmt.Stringer             = State(0)
	_ json.Marshaler           = State(0)
	_ json.Unmarshaler         = (*State)(nil)
	_ encoding.TextMarshaler   = State(0)
	_ encoding.TextUnmarshaler = (*State)(nil)
)

func (s State) isValid() bool {
	return s >= Learning && s <= Relearning
}

// String returns the name of the state ("Learning", "Review", "Relearning").
// For invalid values it returns "State(n)".
func (s State) String() string {
	if s.isValid() {
		return stateNames[s]
	}
	return fmt.Sprintf("State(%d)", int(s))
}

// MarshalText implements encoding.TextMarshaler.
func (s State) MarshalText() ([]byte, error) {
	if !s.isValid() {
		return nil, fmt.Errorf("flux: invalid state: %d", int(s))
	}
	return []byte(stateNames[s]), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *State) UnmarshalText(text []byte) error {
	v, ok := stateByName[string(text)]
	if !ok {
		return fmt.Errorf("flux: invalid state: %q", text)
	}
	*s = v
	return nil
}

// MarshalJSON implements json.Marshaler. State serializes as a JSON string.
func (s State) MarshalJSON() ([]byte, error) {
	text, err := s.MarshalText()
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(text))
}

// UnmarshalJSON implements json.Unmarshaler. Expects a JSON string.
func (s *State) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("flux: invalid state: %s", data)
	}
	return s.UnmarshalText([]byte(str))
}

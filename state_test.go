package flux

import (
	"encoding/json"
	"testing"
)

func TestStateValues(t *testing.T) {
	if Learning != 1 {
		t.Errorf("Learning = %d, want 1", Learning)
	}
	if Review != 2 {
		t.Errorf("Review = %d, want 2", Review)
	}
	if Relearning != 3 {
		t.Errorf("Relearning = %d, want 3", Relearning)
	}
}

func TestStateString(t *testing.T) {
	tests := []struct {
		s    State
		want string
	}{
		{Learning, "Learning"},
		{Review, "Review"},
		{Relearning, "Relearning"},
		{State(0), "State(0)"},
		{State(4), "State(4)"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", int(tt.s), got, tt.want)
		}
	}
}

func TestStateMarshalJSON(t *testing.T) {
	tests := []struct {
		s    State
		want string
	}{
		{Learning, `"Learning"`},
		{Review, `"Review"`},
		{Relearning, `"Relearning"`},
	}
	for _, tt := range tests {
		got, err := json.Marshal(tt.s)
		if err != nil {
			t.Fatalf("json.Marshal(%v): %v", tt.s, err)
		}
		if string(got) != tt.want {
			t.Errorf("json.Marshal(%v) = %s, want %s", tt.s, got, tt.want)
		}
	}
}

func TestStateMarshalJSONInvalid(t *testing.T) {
	s := State(0)
	if _, err := json.Marshal(s); err == nil {
		t.Error("json.Marshal(State(0)) should return error")
	}
}

func TestStateUnmarshalJSON(t *testing.T) {
	tests := []struct {
		input string
		want  State
	}{
		{`"Learning"`, Learning},
		{`"Review"`, Review},
		{`"Relearning"`, Relearning},
	}
	for _, tt := range tests {
		var got State
		if err := json.Unmarshal([]byte(tt.input), &got); err != nil {
			t.Fatalf("json.Unmarshal(%s): %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("json.Unmarshal(%s) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestStateUnmarshalJSONInvalid(t *testing.T) {
	invalid := []string{`"Unknown"`, `""`, `42`, `null`}
	for _, input := range invalid {
		var s State
		if err := json.Unmarshal([]byte(input), &s); err == nil {
			t.Errorf("json.Unmarshal(%s) should return error", input)
		}
	}
}

func TestStateMarshalText(t *testing.T) {
	for _, s := range []State{Learning, Review, Relearning} {
		got, err := s.MarshalText()
		if err != nil {
			t.Fatalf("State(%d).MarshalText(): %v", int(s), err)
		}
		if string(got) != s.String() {
			t.Errorf("MarshalText() = %q, want %q", got, s.String())
		}
	}
}

func TestStateJSONRoundTrip(t *testing.T) {
	for _, s := range []State{Learning, Review, Relearning} {
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("Marshal(%v): %v", s, err)
		}
		var got State
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal(%s): %v", data, err)
		}
		if got != s {
			t.Errorf("round-trip: got %v, want %v", got, s)
		}
	}
}

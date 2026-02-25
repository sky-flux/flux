package flux

import (
	"encoding/json"
	"testing"
)

func TestRatingValues(t *testing.T) {
	if Again != 1 {
		t.Errorf("Again = %d, want 1", Again)
	}
	if Hard != 2 {
		t.Errorf("Hard = %d, want 2", Hard)
	}
	if Good != 3 {
		t.Errorf("Good = %d, want 3", Good)
	}
	if Easy != 4 {
		t.Errorf("Easy = %d, want 4", Easy)
	}
}

func TestRatingString(t *testing.T) {
	tests := []struct {
		r    Rating
		want string
	}{
		{Again, "Again"},
		{Hard, "Hard"},
		{Good, "Good"},
		{Easy, "Easy"},
		{Rating(0), "Rating(0)"},
		{Rating(5), "Rating(5)"},
	}
	for _, tt := range tests {
		if got := tt.r.String(); got != tt.want {
			t.Errorf("Rating(%d).String() = %q, want %q", int(tt.r), got, tt.want)
		}
	}
}

func TestRatingIsValid(t *testing.T) {
	valid := []Rating{Again, Hard, Good, Easy}
	for _, r := range valid {
		if !r.IsValid() {
			t.Errorf("Rating(%d).IsValid() = false, want true", int(r))
		}
	}
	invalid := []Rating{Rating(0), Rating(-1), Rating(5), Rating(100)}
	for _, r := range invalid {
		if r.IsValid() {
			t.Errorf("Rating(%d).IsValid() = true, want false", int(r))
		}
	}
}

func TestRatingMarshalJSON(t *testing.T) {
	tests := []struct {
		r    Rating
		want string
	}{
		{Again, `"Again"`},
		{Hard, `"Hard"`},
		{Good, `"Good"`},
		{Easy, `"Easy"`},
	}
	for _, tt := range tests {
		got, err := json.Marshal(tt.r)
		if err != nil {
			t.Fatalf("json.Marshal(%v): %v", tt.r, err)
		}
		if string(got) != tt.want {
			t.Errorf("json.Marshal(%v) = %s, want %s", tt.r, got, tt.want)
		}
	}
}

func TestRatingUnmarshalJSON(t *testing.T) {
	tests := []struct {
		input string
		want  Rating
	}{
		{`"Again"`, Again},
		{`"Hard"`, Hard},
		{`"Good"`, Good},
		{`"Easy"`, Easy},
	}
	for _, tt := range tests {
		var got Rating
		if err := json.Unmarshal([]byte(tt.input), &got); err != nil {
			t.Fatalf("json.Unmarshal(%s): %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("json.Unmarshal(%s) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestRatingMarshalJSONInvalid(t *testing.T) {
	r := Rating(0)
	if _, err := json.Marshal(r); err == nil {
		t.Error("json.Marshal(Rating(0)) should return error")
	}
}

func TestRatingUnmarshalJSONInvalid(t *testing.T) {
	invalid := []string{`"Unknown"`, `""`, `42`, `null`}
	for _, input := range invalid {
		var r Rating
		if err := json.Unmarshal([]byte(input), &r); err == nil {
			t.Errorf("json.Unmarshal(%s) should return error", input)
		}
	}
}

func TestRatingMarshalText(t *testing.T) {
	tests := []struct {
		r    Rating
		want string
	}{
		{Again, "Again"},
		{Hard, "Hard"},
		{Good, "Good"},
		{Easy, "Easy"},
	}
	for _, tt := range tests {
		got, err := tt.r.MarshalText()
		if err != nil {
			t.Fatalf("Rating(%d).MarshalText(): %v", int(tt.r), err)
		}
		if string(got) != tt.want {
			t.Errorf("Rating(%d).MarshalText() = %q, want %q", int(tt.r), got, tt.want)
		}
	}
}

func TestRatingMarshalTextInvalid(t *testing.T) {
	r := Rating(0)
	if _, err := r.MarshalText(); err == nil {
		t.Error("Rating(0).MarshalText() should return error")
	}
}

func TestRatingUnmarshalText(t *testing.T) {
	tests := []struct {
		input string
		want  Rating
	}{
		{"Again", Again},
		{"Hard", Hard},
		{"Good", Good},
		{"Easy", Easy},
	}
	for _, tt := range tests {
		var got Rating
		if err := got.UnmarshalText([]byte(tt.input)); err != nil {
			t.Fatalf("UnmarshalText(%q): %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("UnmarshalText(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestRatingJSONRoundTrip(t *testing.T) {
	for _, r := range []Rating{Again, Hard, Good, Easy} {
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("Marshal(%v): %v", r, err)
		}
		var got Rating
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal(%s): %v", data, err)
		}
		if got != r {
			t.Errorf("round-trip: got %v, want %v", got, r)
		}
	}
}

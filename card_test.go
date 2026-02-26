package flux

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewCard(t *testing.T) {
	c := NewCard(42)
	if c.CardID != 42 {
		t.Errorf("CardID = %d, want 42", c.CardID)
	}
	if c.State != Learning {
		t.Errorf("State = %v, want Learning", c.State)
	}
	step := 0
	if c.Step == nil || *c.Step != step {
		t.Errorf("Step = %v, want %d", c.Step, step)
	}
	if c.Stability != nil {
		t.Errorf("Stability = %v, want nil", c.Stability)
	}
	if c.Difficulty != nil {
		t.Errorf("Difficulty = %v, want nil", c.Difficulty)
	}
	if c.Due.IsZero() {
		t.Error("Due should not be zero")
	}
	if c.LastReview != nil {
		t.Errorf("LastReview = %v, want nil", c.LastReview)
	}
}

func TestCardClone(t *testing.T) {
	c := NewCard(1)
	s := 3.5
	d := 5.0
	step := 1
	now := time.Now()
	c.Stability = &s
	c.Difficulty = &d
	c.Step = &step
	c.LastReview = &now

	cloned := c.clone()

	// Values equal.
	if cloned.CardID != c.CardID {
		t.Error("clone CardID mismatch")
	}
	if *cloned.Stability != *c.Stability {
		t.Error("clone Stability value mismatch")
	}
	if *cloned.Difficulty != *c.Difficulty {
		t.Error("clone Difficulty value mismatch")
	}
	if *cloned.Step != *c.Step {
		t.Error("clone Step value mismatch")
	}
	if !cloned.LastReview.Equal(*c.LastReview) {
		t.Error("clone LastReview value mismatch")
	}

	// Pointers independent.
	*cloned.Stability = 99.0
	if *c.Stability == 99.0 {
		t.Error("clone Stability pointer not independent")
	}
	*cloned.Difficulty = 99.0
	if *c.Difficulty == 99.0 {
		t.Error("clone Difficulty pointer not independent")
	}
	*cloned.Step = 99
	if *c.Step == 99 {
		t.Error("clone Step pointer not independent")
	}
}

func TestCardCloneNilFields(t *testing.T) {
	c := NewCard(1)
	c.Stability = nil
	c.Difficulty = nil
	c.Step = nil
	c.LastReview = nil

	cloned := c.clone()
	if cloned.Stability != nil || cloned.Difficulty != nil || cloned.Step != nil || cloned.LastReview != nil {
		t.Error("clone should preserve nil fields")
	}
}

func TestCardSetStability(t *testing.T) {
	c := NewCard(1)
	c.setStability(3.5)
	if c.Stability == nil || *c.Stability != 3.5 {
		t.Errorf("Stability = %v, want 3.5", c.Stability)
	}
}

func TestCardSetDifficulty(t *testing.T) {
	c := NewCard(1)
	c.setDifficulty(5.0)
	if c.Difficulty == nil || *c.Difficulty != 5.0 {
		t.Errorf("Difficulty = %v, want 5.0", c.Difficulty)
	}
}

func TestCardSetStep(t *testing.T) {
	c := NewCard(1)
	c.setStep(2)
	if c.Step == nil || *c.Step != 2 {
		t.Errorf("Step = %v, want 2", c.Step)
	}
}

func TestCardClearStep(t *testing.T) {
	c := NewCard(1)
	c.setStep(2)
	c.clearStep()
	if c.Step != nil {
		t.Errorf("Step = %v, want nil", c.Step)
	}
}

func TestCardJSONRoundTrip(t *testing.T) {
	now := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	s := 3.5
	d := 5.0
	step := 1

	c := Card{
		CardID:     42,
		State:      Review,
		Step:       &step,
		Stability:  &s,
		Difficulty: &d,
		Due:        now,
		LastReview: &now,
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got Card
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.CardID != c.CardID {
		t.Errorf("CardID = %d, want %d", got.CardID, c.CardID)
	}
	if got.State != c.State {
		t.Errorf("State = %v, want %v", got.State, c.State)
	}
	if *got.Step != *c.Step {
		t.Errorf("Step = %d, want %d", *got.Step, *c.Step)
	}
	if *got.Stability != *c.Stability {
		t.Errorf("Stability = %f, want %f", *got.Stability, *c.Stability)
	}
	if *got.Difficulty != *c.Difficulty {
		t.Errorf("Difficulty = %f, want %f", *got.Difficulty, *c.Difficulty)
	}
	if !got.Due.Equal(c.Due) {
		t.Errorf("Due = %v, want %v", got.Due, c.Due)
	}
	if !got.LastReview.Equal(*c.LastReview) {
		t.Errorf("LastReview = %v, want %v", got.LastReview, c.LastReview)
	}
}

func TestCardJSONNilFields(t *testing.T) {
	c := Card{
		CardID: 1,
		State:  Learning,
		Due:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	s := string(data)
	// Nil pointer fields should serialize as null.
	expected := map[string]bool{
		`"stability":null`:   true,
		`"difficulty":null`:  true,
		`"last_review":null`: true,
		`"step":null`:        true,
	}
	for substr, shouldContain := range expected {
		contains := containsSubstr(s, substr)
		if shouldContain && !contains {
			t.Errorf("JSON should contain %s, got %s", substr, s)
		}
	}
}

func containsSubstr(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstr(s, substr)
}

func searchSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

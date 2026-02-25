package flux

import (
	"encoding/json"
	"testing"
	"time"
)

func TestReviewLogFields(t *testing.T) {
	now := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	dur := 1500
	rl := ReviewLog{
		CardID:         42,
		Rating:         Good,
		ReviewDatetime: now,
		ReviewDuration: &dur,
	}
	if rl.CardID != 42 {
		t.Errorf("CardID = %d, want 42", rl.CardID)
	}
	if rl.Rating != Good {
		t.Errorf("Rating = %v, want Good", rl.Rating)
	}
	if !rl.ReviewDatetime.Equal(now) {
		t.Errorf("ReviewDatetime = %v, want %v", rl.ReviewDatetime, now)
	}
	if rl.ReviewDuration == nil || *rl.ReviewDuration != 1500 {
		t.Errorf("ReviewDuration = %v, want 1500", rl.ReviewDuration)
	}
}

func TestReviewLogJSONRoundTrip(t *testing.T) {
	now := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	dur := 2500
	rl := ReviewLog{
		CardID:         7,
		Rating:         Hard,
		ReviewDatetime: now,
		ReviewDuration: &dur,
	}

	data, err := json.Marshal(rl)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got ReviewLog
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.CardID != rl.CardID || got.Rating != rl.Rating || *got.ReviewDuration != *rl.ReviewDuration {
		t.Errorf("round-trip mismatch: got %+v", got)
	}
}

func TestReviewLogJSONOmitDuration(t *testing.T) {
	rl := ReviewLog{
		CardID:         1,
		Rating:         Again,
		ReviewDatetime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(rl)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	s := string(data)
	// ReviewDuration is omitempty — nil means field absent from JSON.
	if searchSubstr(s, "review_duration") {
		t.Errorf("nil ReviewDuration should be omitted, got %s", s)
	}
}

func TestReviewLogJSONRatingAsString(t *testing.T) {
	rl := ReviewLog{
		CardID:         1,
		Rating:         Easy,
		ReviewDatetime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(rl)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// Rating should serialize as string "Easy", not number 4.
	if !searchSubstr(string(data), `"Easy"`) {
		t.Errorf("Rating should be string in JSON, got %s", data)
	}
}

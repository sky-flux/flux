package optimizer

import (
	"testing"
	"time"

	"github.com/sky-flux/flux"
)

var t0 = time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

func TestFormatRevlogsEmpty(t *testing.T) {
	got := formatRevlogs(nil)
	if len(got) != 0 {
		t.Errorf("formatRevlogs(nil) returned %d groups, want 0", len(got))
	}
}

func TestFormatRevlogsSingleCard(t *testing.T) {
	logs := []flux.ReviewLog{
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(10 * time.Minute)},
		{CardID: 1, Rating: flux.Again, ReviewDatetime: t0},
		{CardID: 1, Rating: flux.Easy, ReviewDatetime: t0.Add(24 * time.Hour)},
	}
	got := formatRevlogs(logs)

	if len(got) != 1 {
		t.Fatalf("got %d groups, want 1", len(got))
	}
	reviews := got[1]
	if len(reviews) != 3 {
		t.Fatalf("card 1 has %d reviews, want 3", len(reviews))
	}
	// Should be sorted by time.
	if reviews[0].rating != flux.Again {
		t.Errorf("first review rating = %v, want Again", reviews[0].rating)
	}
	if reviews[1].rating != flux.Good {
		t.Errorf("second review rating = %v, want Good", reviews[1].rating)
	}
	if reviews[2].rating != flux.Easy {
		t.Errorf("third review rating = %v, want Easy", reviews[2].rating)
	}
}

func TestFormatRevlogsMultiCard(t *testing.T) {
	logs := []flux.ReviewLog{
		{CardID: 2, Rating: flux.Hard, ReviewDatetime: t0},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0},
		{CardID: 2, Rating: flux.Good, ReviewDatetime: t0.Add(time.Hour)},
	}
	got := formatRevlogs(logs)

	if len(got) != 2 {
		t.Fatalf("got %d groups, want 2", len(got))
	}
	if len(got[1]) != 1 {
		t.Errorf("card 1 has %d reviews, want 1", len(got[1]))
	}
	if len(got[2]) != 2 {
		t.Errorf("card 2 has %d reviews, want 2", len(got[2]))
	}
}

func TestFormatRevlogsElapsedDays(t *testing.T) {
	logs := []flux.ReviewLog{
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(3 * 24 * time.Hour)},
		{CardID: 1, Rating: flux.Again, ReviewDatetime: t0.Add(3*24*time.Hour + time.Hour)},
	}
	got := formatRevlogs(logs)
	reviews := got[1]

	// First review: elapsed_days = 0 (no previous).
	if reviews[0].elapsedDays != 0 {
		t.Errorf("review[0].elapsedDays = %f, want 0", reviews[0].elapsedDays)
	}
	// Second review: 3 days later.
	assertFloatOpt(t, "review[1].elapsedDays", reviews[1].elapsedDays, 3.0)
	// Third review: same day as second (1 hour later).
	assertFloatOpt(t, "review[2].elapsedDays", reviews[2].elapsedDays, 1.0/24.0)
}

func TestFormatRevlogsLabel(t *testing.T) {
	logs := []flux.ReviewLog{
		{CardID: 1, Rating: flux.Again, ReviewDatetime: t0},
		{CardID: 1, Rating: flux.Hard, ReviewDatetime: t0.Add(24 * time.Hour)},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(48 * time.Hour)},
	}
	got := formatRevlogs(logs)
	reviews := got[1]

	// Again → label=0, Hard/Good/Easy → label=1.
	if reviews[0].label != 0 {
		t.Errorf("Again label = %f, want 0", reviews[0].label)
	}
	if reviews[1].label != 1 {
		t.Errorf("Hard label = %f, want 1", reviews[1].label)
	}
	if reviews[2].label != 1 {
		t.Errorf("Good label = %f, want 1", reviews[2].label)
	}
}

func TestCountCrossDayReviews(t *testing.T) {
	logs := []flux.ReviewLog{
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(3 * 24 * time.Hour)},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(3*24*time.Hour + time.Minute)},
		{CardID: 2, Rating: flux.Hard, ReviewDatetime: t0},
		{CardID: 2, Rating: flux.Easy, ReviewDatetime: t0.Add(7 * 24 * time.Hour)},
	}
	data := formatRevlogs(logs)
	got := countCrossDayReviews(data)
	// Card 1: review[0] is first (not cross-day), review[1] 3d later (cross-day),
	//          review[2] same day (not cross-day).
	// Card 2: review[0] is first, review[1] 7d later (cross-day).
	// Total: 2
	if got != 2 {
		t.Errorf("countCrossDayReviews = %d, want 2", got)
	}
}

func TestCountCrossDayReviewsEmpty(t *testing.T) {
	got := countCrossDayReviews(nil)
	if got != 0 {
		t.Errorf("countCrossDayReviews(nil) = %d, want 0", got)
	}
}

func assertFloatOpt(t *testing.T, name string, got, want float64) {
	t.Helper()
	const eps = 1e-4
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	if diff > eps {
		t.Errorf("%s = %.6f, want %.6f (diff %.6f)", name, got, want, diff)
	}
}

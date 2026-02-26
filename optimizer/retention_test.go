package optimizer

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/sky-flux/flux"
)

// --- computeProbsAndCosts ---

func TestComputeProbsAndCosts(t *testing.T) {
	dur := func(ms int) *int { return &ms }

	// Card 1: two reviews. First = Again (500ms), second = Good (800ms).
	// Card 2: two reviews. First = Good (600ms), second = Hard (700ms).
	// Card 3: one review. First = Easy (400ms).
	logs := []flux.ReviewLog{
		{CardID: 1, Rating: flux.Again, ReviewDatetime: t0, ReviewDuration: dur(500)},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(24 * time.Hour), ReviewDuration: dur(800)},
		{CardID: 2, Rating: flux.Good, ReviewDatetime: t0, ReviewDuration: dur(600)},
		{CardID: 2, Rating: flux.Hard, ReviewDatetime: t0.Add(48 * time.Hour), ReviewDuration: dur(700)},
		{CardID: 3, Rating: flux.Easy, ReviewDatetime: t0, ReviewDuration: dur(400)},
	}

	m := computeProbsAndCosts(logs)

	// First reviews: Card1=Again, Card2=Good, Card3=Easy → 3 total first reviews.
	// prob_first_again = 1/3, prob_first_good = 1/3, prob_first_easy = 1/3, prob_first_hard = 0.
	assertFloatOpt(t, "prob_first_again", m["prob_first_again"], 1.0/3.0)
	assertFloatOpt(t, "prob_first_hard", m["prob_first_hard"], 0.0)
	assertFloatOpt(t, "prob_first_good", m["prob_first_good"], 1.0/3.0)
	assertFloatOpt(t, "prob_first_easy", m["prob_first_easy"], 1.0/3.0)

	// First-review durations: Again=500, Hard=0(default), Good=600, Easy=400.
	assertFloatOpt(t, "avg_first_again_duration", m["avg_first_again_duration"], 500.0)
	assertFloatOpt(t, "avg_first_hard_duration", m["avg_first_hard_duration"], 0.0)
	assertFloatOpt(t, "avg_first_good_duration", m["avg_first_good_duration"], 600.0)
	assertFloatOpt(t, "avg_first_easy_duration", m["avg_first_easy_duration"], 400.0)

	// Non-first reviews: Card1 second=Good(800), Card2 second=Hard(700). Total=2.
	// Among these, recall ratings (not Again): Good(800), Hard(700) → both recalled.
	// prob_hard = 1/2, prob_good = 1/2, prob_easy = 0.
	assertFloatOpt(t, "prob_hard", m["prob_hard"], 1.0/2.0)
	assertFloatOpt(t, "prob_good", m["prob_good"], 1.0/2.0)
	assertFloatOpt(t, "prob_easy", m["prob_easy"], 0.0)

	// Non-first durations: Again=0(default), Hard=700, Good=800, Easy=0(default).
	assertFloatOpt(t, "avg_again_duration", m["avg_again_duration"], 0.0)
	assertFloatOpt(t, "avg_hard_duration", m["avg_hard_duration"], 700.0)
	assertFloatOpt(t, "avg_good_duration", m["avg_good_duration"], 800.0)
	assertFloatOpt(t, "avg_easy_duration", m["avg_easy_duration"], 0.0)
}

func TestComputeProbsAndCostsFirstOnly(t *testing.T) {
	dur := func(ms int) *int { return &ms }

	// All cards have only one review → all first-review stats, no non-first stats.
	logs := []flux.ReviewLog{
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0, ReviewDuration: dur(300)},
		{CardID: 2, Rating: flux.Again, ReviewDatetime: t0, ReviewDuration: dur(500)},
		{CardID: 3, Rating: flux.Good, ReviewDatetime: t0, ReviewDuration: dur(400)},
		{CardID: 4, Rating: flux.Easy, ReviewDatetime: t0, ReviewDuration: dur(200)},
	}

	m := computeProbsAndCosts(logs)

	// First reviews: 1 Again, 0 Hard, 2 Good, 1 Easy → total 4.
	assertFloatOpt(t, "prob_first_again", m["prob_first_again"], 1.0/4.0)
	assertFloatOpt(t, "prob_first_hard", m["prob_first_hard"], 0.0)
	assertFloatOpt(t, "prob_first_good", m["prob_first_good"], 2.0/4.0)
	assertFloatOpt(t, "prob_first_easy", m["prob_first_easy"], 1.0/4.0)

	// No non-first reviews → recall probs default to uniform.
	// When there are no recall reviews, prob_hard=1/3, prob_good=1/3, prob_easy=1/3.
	assertFloatOpt(t, "prob_hard", m["prob_hard"], 1.0/3.0)
	assertFloatOpt(t, "prob_good", m["prob_good"], 1.0/3.0)
	assertFloatOpt(t, "prob_easy", m["prob_easy"], 1.0/3.0)
}

// --- simulateCost ---

func TestSimulateCostInvalidParams(t *testing.T) {
	// Non-zero but out-of-bounds params → NewScheduler fails → +Inf.
	// w[4] lower bound is 1.0; set to 0.5 to trigger validation error.
	badParams := flux.DefaultParameters
	badParams[4] = 0.5
	m := defaultProbsAndCosts()
	cost := simulateCost(0.9, badParams, m)
	if !math.IsInf(cost, 1) {
		t.Errorf("simulateCost with invalid params = %f, want +Inf", cost)
	}
}

func TestSimulateCostReproducible(t *testing.T) {
	m := defaultProbsAndCosts()
	cost1 := simulateCost(0.9, flux.DefaultParameters, m)
	cost2 := simulateCost(0.9, flux.DefaultParameters, m)
	if cost1 != cost2 {
		t.Errorf("simulateCost not reproducible: %f != %f", cost1, cost2)
	}
	if cost1 <= 0 {
		t.Errorf("simulateCost = %f, want > 0", cost1)
	}
}

func TestSimulateCostHigherRetentionLowerCost(t *testing.T) {
	m := defaultProbsAndCosts()
	costLow := simulateCost(0.70, flux.DefaultParameters, m)
	costHigh := simulateCost(0.95, flux.DefaultParameters, m)
	// Higher retention → fewer lapses → generally lower cost per retained card.
	if costHigh >= costLow {
		t.Errorf("expected cost at 0.95 (%f) < cost at 0.70 (%f)", costHigh, costLow)
	}
}

// --- ComputeOptimalRetention ---

func TestComputeOptimalRetentionInsufficientLogs(t *testing.T) {
	o := NewOptimizer(OptimizerConfig{})
	logs := make([]flux.ReviewLog, 100)
	dur := 1000
	for i := range logs {
		logs[i] = flux.ReviewLog{
			CardID:         int64(i + 1),
			Rating:         flux.Good,
			ReviewDatetime: t0,
			ReviewDuration: &dur,
		}
	}
	_, err := o.ComputeOptimalRetention(context.Background(), flux.DefaultParameters, logs)
	if !errors.Is(err, ErrInsufficientLogs) {
		t.Errorf("got error %v, want ErrInsufficientLogs", err)
	}
}

func TestComputeOptimalRetentionMissingDuration(t *testing.T) {
	o := NewOptimizer(OptimizerConfig{})
	dur := 1000
	logs := make([]flux.ReviewLog, 600)
	for i := range logs {
		logs[i] = flux.ReviewLog{
			CardID:         int64(i + 1),
			Rating:         flux.Good,
			ReviewDatetime: t0,
			ReviewDuration: &dur,
		}
	}
	// Set one to nil.
	logs[300].ReviewDuration = nil

	_, err := o.ComputeOptimalRetention(context.Background(), flux.DefaultParameters, logs)
	if !errors.Is(err, ErrMissingDuration) {
		t.Errorf("got error %v, want ErrMissingDuration", err)
	}
}

func TestComputeOptimalRetentionValid(t *testing.T) {
	o := NewOptimizer(OptimizerConfig{})
	logs := generateSyntheticLogsWithDuration(200, 10, 42)

	ret, err := o.ComputeOptimalRetention(context.Background(), flux.DefaultParameters, logs)
	if err != nil {
		t.Fatalf("ComputeOptimalRetention: %v", err)
	}
	// Result must be one of the candidates.
	valid := false
	for _, c := range []float64{0.70, 0.75, 0.80, 0.85, 0.90, 0.95} {
		if ret == c {
			valid = true
			break
		}
	}
	if !valid {
		t.Errorf("retention = %f, want one of [0.70, 0.75, 0.80, 0.85, 0.90, 0.95]", ret)
	}
}

func TestComputeOptimalRetentionContextCancel(t *testing.T) {
	o := NewOptimizer(OptimizerConfig{})
	logs := generateSyntheticLogsWithDuration(200, 10, 42)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := o.ComputeOptimalRetention(ctx, flux.DefaultParameters, logs)
	if err == nil {
		t.Fatal("expected context error")
	}
}

// --- helpers ---

// defaultProbsAndCosts returns a reasonable probsAndCosts map for testing.
func defaultProbsAndCosts() map[string]float64 {
	return map[string]float64{
		"prob_first_again": 0.30,
		"prob_first_hard":  0.05,
		"prob_first_good":  0.55,
		"prob_first_easy":  0.10,

		"avg_first_again_duration": 8000,
		"avg_first_hard_duration":  6000,
		"avg_first_good_duration":  4000,
		"avg_first_easy_duration":  2000,

		"prob_hard": 0.10,
		"prob_good": 0.80,
		"prob_easy": 0.10,

		"avg_again_duration": 10000,
		"avg_hard_duration":  7000,
		"avg_good_duration":  4000,
		"avg_easy_duration":  2000,
	}
}

// generateSyntheticLogsWithDuration is like generateSyntheticLogs but sets ReviewDuration.
func generateSyntheticLogsWithDuration(numCards, reviewsPerCard int, seed int64) []flux.ReviewLog {
	logs := generateSyntheticLogs(numCards, reviewsPerCard, seed)
	dur := 5000 // 5 seconds in ms
	for i := range logs {
		logs[i].ReviewDuration = &dur
	}
	return logs
}

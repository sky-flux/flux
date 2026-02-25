package optimizer

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/sky-flux/flux"
)

var (
	// ErrInsufficientLogs is returned when fewer than 512 review logs are provided.
	ErrInsufficientLogs = errors.New("optimizer: at least 512 review logs required for optimal retention")

	// ErrMissingDuration is returned when any ReviewDuration is nil.
	ErrMissingDuration = errors.New("optimizer: ReviewDuration must not be nil for optimal retention")
)

// computeProbsAndCosts computes rating probabilities and average durations from review logs.
// "First review" = the first review of each card_id. "Non-first" = all subsequent reviews.
// For non-first recall rating probabilities, compute among only recalled reviews (not Again).
func computeProbsAndCosts(logs []flux.ReviewLog) map[string]float64 {
	// Group by card and sort by time to identify first vs non-first.
	type entry struct {
		rating   flux.Rating
		duration float64
		time     time.Time
	}
	groups := make(map[int64][]entry)
	for _, log := range logs {
		d := 0.0
		if log.ReviewDuration != nil {
			d = float64(*log.ReviewDuration)
		}
		groups[log.CardID] = append(groups[log.CardID], entry{
			rating:   log.Rating,
			duration: d,
			time:     log.ReviewDatetime,
		})
	}
	for _, g := range groups {
		sort.Slice(g, func(i, j int) bool {
			return g[i].time.Before(g[j].time)
		})
	}

	// Counters for first reviews.
	var firstTotal float64
	firstCount := map[flux.Rating]float64{}
	firstDurSum := map[flux.Rating]float64{}
	firstDurCount := map[flux.Rating]float64{}

	// Counters for non-first reviews.
	var recallTotal float64
	recallCount := map[flux.Rating]float64{}
	nonFirstDurSum := map[flux.Rating]float64{}
	nonFirstDurCount := map[flux.Rating]float64{}

	for _, g := range groups {
		for i, e := range g {
			if i == 0 {
				firstTotal++
				firstCount[e.rating]++
				firstDurSum[e.rating] += e.duration
				firstDurCount[e.rating]++
			} else {
				nonFirstDurSum[e.rating] += e.duration
				nonFirstDurCount[e.rating]++
				if e.rating != flux.Again {
					recallTotal++
					recallCount[e.rating]++
				}
			}
		}
	}

	m := make(map[string]float64)

	// First-review probabilities.
	if firstTotal > 0 {
		m["prob_first_again"] = firstCount[flux.Again] / firstTotal
		m["prob_first_hard"] = firstCount[flux.Hard] / firstTotal
		m["prob_first_good"] = firstCount[flux.Good] / firstTotal
		m["prob_first_easy"] = firstCount[flux.Easy] / firstTotal
	}

	// First-review average durations.
	for _, r := range []flux.Rating{flux.Again, flux.Hard, flux.Good, flux.Easy} {
		key := "avg_first_" + ratingLowerNames[r] + "_duration"
		if firstDurCount[r] > 0 {
			m[key] = firstDurSum[r] / firstDurCount[r]
		}
	}

	// Non-first recall probabilities (among Hard/Good/Easy only).
	if recallTotal > 0 {
		m["prob_hard"] = recallCount[flux.Hard] / recallTotal
		m["prob_good"] = recallCount[flux.Good] / recallTotal
		m["prob_easy"] = recallCount[flux.Easy] / recallTotal
	} else {
		// Default to uniform when no recall data.
		m["prob_hard"] = 1.0 / 3.0
		m["prob_good"] = 1.0 / 3.0
		m["prob_easy"] = 1.0 / 3.0
	}

	// Non-first average durations.
	for _, r := range []flux.Rating{flux.Again, flux.Hard, flux.Good, flux.Easy} {
		key := "avg_" + ratingLowerNames[r] + "_duration"
		if nonFirstDurCount[r] > 0 {
			m[key] = nonFirstDurSum[r] / nonFirstDurCount[r]
		}
	}

	return m
}

var ratingLowerNames = map[flux.Rating]string{
	flux.Again: "again",
	flux.Hard:  "hard",
	flux.Good:  "good",
	flux.Easy:  "easy",
}

// simulateCost runs a Monte Carlo simulation to estimate the cost per retained card
// for a given desired retention. It simulates 1000 cards over one year.
func simulateCost(retention float64, params [21]float64, probsAndCosts map[string]float64) float64 {
	const numCards = 1000

	s, err := flux.NewScheduler(flux.SchedulerConfig{
		Parameters:       params,
		DesiredRetention: retention,
		DisableFuzzing:   true,
	})
	if err != nil {
		return math.Inf(1)
	}

	rng := rand.New(rand.NewSource(42))

	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Extract probabilities and costs.
	pfAgain := probsAndCosts["prob_first_again"]
	pfHard := probsAndCosts["prob_first_hard"]
	pfGood := probsAndCosts["prob_first_good"]
	// pfEasy is remainder

	dFirstAgain := probsAndCosts["avg_first_again_duration"]
	dFirstHard := probsAndCosts["avg_first_hard_duration"]
	dFirstGood := probsAndCosts["avg_first_good_duration"]
	dFirstEasy := probsAndCosts["avg_first_easy_duration"]

	pHard := probsAndCosts["prob_hard"]
	pGood := probsAndCosts["prob_good"]
	// pEasy is remainder

	dAgain := probsAndCosts["avg_again_duration"]
	dHard := probsAndCosts["avg_hard_duration"]
	dGood := probsAndCosts["avg_good_duration"]
	dEasy := probsAndCosts["avg_easy_duration"]

	var totalDuration float64

	for i := 0; i < numCards; i++ {
		card := flux.NewCard(int64(i + 1))
		card.Due = startDate
		now := startDate
		isFirst := true

		for !now.After(endDate) {
			var rating flux.Rating
			var dur float64

			if isFirst {
				// Choose rating from first-review probabilities.
				p := rng.Float64()
				switch {
				case p < pfAgain:
					rating = flux.Again
					dur = dFirstAgain
				case p < pfAgain+pfHard:
					rating = flux.Hard
					dur = dFirstHard
				case p < pfAgain+pfHard+pfGood:
					rating = flux.Good
					dur = dFirstGood
				default:
					rating = flux.Easy
					dur = dFirstEasy
				}
				isFirst = false
			} else {
				// Non-first: with probability=retention → recall, else → Again.
				if rng.Float64() < retention {
					// Recalled: choose among Hard/Good/Easy.
					p := rng.Float64()
					switch {
					case p < pHard:
						rating = flux.Hard
						dur = dHard
					case p < pHard+pGood:
						rating = flux.Good
						dur = dGood
					default:
						rating = flux.Easy
						dur = dEasy
					}
				} else {
					rating = flux.Again
					dur = dAgain
				}
			}

			totalDuration += dur
			card, _ = s.ReviewCard(card, rating, now)
			now = card.Due
		}
	}

	return totalDuration / (retention * numCards)
}

// ComputeOptimalRetention finds the retention value (from candidates) with minimal
// simulated cost. It validates inputs and checks for context cancellation.
func (o *Optimizer) ComputeOptimalRetention(ctx context.Context, params [21]float64, logs []flux.ReviewLog) (float64, error) {
	if len(logs) < 512 {
		return 0, ErrInsufficientLogs
	}
	for _, log := range logs {
		if log.ReviewDuration == nil {
			return 0, ErrMissingDuration
		}
	}

	probsAndCosts := computeProbsAndCosts(logs)
	candidates := []float64{0.70, 0.75, 0.80, 0.85, 0.90, 0.95}

	bestRetention := candidates[0]
	bestCost := math.Inf(1)

	for _, c := range candidates {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		cost := simulateCost(c, params, probsAndCosts)
		if cost < bestCost {
			bestCost = cost
			bestRetention = c
		}
	}

	return bestRetention, nil
}

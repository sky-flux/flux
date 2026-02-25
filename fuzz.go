package flux

import (
	"math"
	"math/rand"
)

type fuzzEntry struct {
	start, end float64
	factor     float64
}

var fuzzRanges = []fuzzEntry{
	{2.5, 7.0, 0.15},
	{7.0, 20.0, 0.10},
	{20.0, math.Inf(1), 0.05},
}

// fuzzDelta computes the fuzz range delta for a given interval.
// delta = 1.0 + Σ(factor * max(min(interval, end) - start, 0))
func fuzzDelta(interval float64) float64 {
	delta := 1.0
	for _, r := range fuzzRanges {
		delta += r.factor * math.Max(math.Min(interval, r.end)-r.start, 0)
	}
	return delta
}

// applyFuzz randomizes the interval to prevent review clustering.
// Returns the original interval unchanged if < 2.5 days.
func applyFuzz(interval, maxIvl int, rng *rand.Rand) int {
	if float64(interval) < 2.5 {
		return interval
	}

	ivl := float64(interval)
	delta := fuzzDelta(ivl)

	minIvl := max(2, int(math.Round(ivl-delta)))
	maxFuzzIvl := min(int(math.Round(ivl+delta)), maxIvl)
	minIvl = min(minIvl, maxFuzzIvl)

	fuzzed := int(math.Round(rng.Float64()*float64(maxFuzzIvl-minIvl+1))) + minIvl
	fuzzed = min(fuzzed, maxIvl)
	return fuzzed
}

package flux

import (
	"math/rand"
	"testing"
)

func TestFuzzDeltaSingleBand(t *testing.T) {
	// interval=3 → only [2.5, 7) band: factor=0.15
	// delta = 1.0 + 0.15 * (min(3, 7) - 2.5) = 1.0 + 0.15*0.5 = 1.075
	got := fuzzDelta(3.0)
	assertFloat(t, "fuzzDelta(3)", got, 1.075)
}

func TestFuzzDeltaTwoBands(t *testing.T) {
	// interval=10 → [2.5,7) full + [7,20) partial
	// band1: 0.15 * (7 - 2.5) = 0.15 * 4.5 = 0.675
	// band2: 0.10 * (10 - 7) = 0.10 * 3.0 = 0.3
	// delta = 1.0 + 0.675 + 0.3 = 1.975
	got := fuzzDelta(10.0)
	assertFloat(t, "fuzzDelta(10)", got, 1.975)
}

func TestFuzzDeltaThreeBands(t *testing.T) {
	// interval=50 → all three bands
	// band1: 0.15 * (7 - 2.5) = 0.675
	// band2: 0.10 * (20 - 7) = 1.3
	// band3: 0.05 * (50 - 20) = 1.5
	// delta = 1.0 + 0.675 + 1.3 + 1.5 = 4.475
	got := fuzzDelta(50.0)
	assertFloat(t, "fuzzDelta(50)", got, 4.475)
}

func TestApplyFuzzNoFuzzSmallInterval(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	// interval < 2.5 → no fuzz, return as-is
	if got := applyFuzz(1, 36500, rng); got != 1 {
		t.Errorf("applyFuzz(1) = %d, want 1", got)
	}
	if got := applyFuzz(2, 36500, rng); got != 2 {
		t.Errorf("applyFuzz(2) = %d, want 2", got)
	}
}

func TestApplyFuzzWithinBounds(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	// interval=10, delta=1.975
	// min_ivl = max(2, round(10-1.975)) = 8
	// max_fuzz_ivl = min(round(10+1.975), 36500) = 12
	// formula: round(rand()*(12-8+1)+8) can produce up to 13
	// final: min(13, 36500) = 13
	for i := 0; i < 100; i++ {
		got := applyFuzz(10, 36500, rng)
		if got < 8 || got > 13 {
			t.Errorf("applyFuzz(10) = %d, expected [8, 13]", got)
		}
	}
}

func TestApplyFuzzMaxIvlClamp(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	// interval=50, maxIvl=48
	// delta=4.475
	// min_ivl = max(2, round(50-4.475)) = 46
	// max_ivl = min(round(50+4.475), 48) = 48
	// min_ivl = min(46, 48) = 46
	// result should be in [46, 48]
	for i := 0; i < 100; i++ {
		got := applyFuzz(50, 48, rng)
		if got < 46 || got > 48 {
			t.Errorf("applyFuzz(50, maxIvl=48) = %d, expected [46, 48]", got)
		}
	}
}

func TestApplyFuzzReproducible(t *testing.T) {
	// Same seed → same results
	rng1 := rand.New(rand.NewSource(123))
	rng2 := rand.New(rand.NewSource(123))
	for i := 0; i < 20; i++ {
		a := applyFuzz(15, 36500, rng1)
		b := applyFuzz(15, 36500, rng2)
		if a != b {
			t.Errorf("iteration %d: %d != %d with same seed", i, a, b)
		}
	}
}

func TestApplyFuzzInterval3(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	// interval=3, delta=1.075
	// min_ivl = max(2, round(1.925)) = 2
	// max_fuzz_ivl = min(round(4.075), 36500) = 4
	// formula: round(rand()*(4-2+1)+2) can produce up to 5
	// final: min(5, 36500) = 5
	for i := 0; i < 100; i++ {
		got := applyFuzz(3, 36500, rng)
		if got < 2 || got > 5 {
			t.Errorf("applyFuzz(3) = %d, expected [2, 5]", got)
		}
	}
}

func TestApplyFuzzNeverExceedsMaxIvl(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	maxIvl := 10
	for i := 0; i < 200; i++ {
		got := applyFuzz(8, maxIvl, rng)
		if got > maxIvl {
			t.Errorf("applyFuzz(8, max=%d) = %d, exceeds max", maxIvl, got)
		}
		if got < 1 {
			t.Errorf("applyFuzz(8, max=%d) = %d, below 1", maxIvl, got)
		}
	}
}

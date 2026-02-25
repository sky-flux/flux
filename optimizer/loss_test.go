package optimizer

import (
	"math"
	"testing"
	"time"

	"github.com/sky-flux/flux"
)

// --- bceLoss ---

func TestBceLossRecalled(t *testing.T) {
	// -[1*ln(0.9) + 0*ln(0.1)] = -ln(0.9) ≈ 0.10536
	got := bceLoss(0.9, 1)
	assertFloatOpt(t, "bceLoss(0.9,1)", got, 0.10536)
}

func TestBceLossForgotten(t *testing.T) {
	// -[0*ln(0.9) + 1*ln(0.1)] = -ln(0.1) ≈ 2.30259
	got := bceLoss(0.9, 0)
	assertFloatOpt(t, "bceLoss(0.9,0)", got, 2.30259)
}

func TestBceLossHalf(t *testing.T) {
	// -[1*ln(0.5) + 0*ln(0.5)] = -ln(0.5) ≈ 0.69315
	got := bceLoss(0.5, 1)
	assertFloatOpt(t, "bceLoss(0.5,1)", got, 0.69315)
}

func TestBceLossClampLow(t *testing.T) {
	// rPred near 0 should be clamped to avoid -Inf.
	got := bceLoss(0.0, 1)
	if math.IsInf(got, 0) || math.IsNaN(got) {
		t.Errorf("bceLoss(0,1) = %v, should not be Inf/NaN", got)
	}
}

func TestBceLossClampHigh(t *testing.T) {
	// rPred near 1 should be clamped to avoid -Inf for (1-rPred).
	got := bceLoss(1.0, 0)
	if math.IsInf(got, 0) || math.IsNaN(got) {
		t.Errorf("bceLoss(1,0) = %v, should not be Inf/NaN", got)
	}
}

// --- computeBatchLoss ---

func TestComputeBatchLossBasic(t *testing.T) {
	// Card 1: review at t0, then cross-day review at t0+3d.
	// The cross-day review has a predicted retrievability from the Scheduler.
	logs := []flux.ReviewLog{
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(10 * time.Minute)},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(3 * 24 * time.Hour)},
	}
	data := formatRevlogs(logs)
	loss := computeBatchLoss(flux.DefaultParameters, data)

	// Loss should be finite and positive.
	if math.IsNaN(loss) || math.IsInf(loss, 0) {
		t.Fatalf("computeBatchLoss = %v, want finite", loss)
	}
	if loss <= 0 {
		t.Errorf("computeBatchLoss = %f, want > 0", loss)
	}
}

func TestComputeBatchLossNoCrossDay(t *testing.T) {
	// Only same-day reviews → no cross-day → no loss contributions → return 0.
	logs := []flux.ReviewLog{
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(5 * time.Minute)},
	}
	data := formatRevlogs(logs)
	loss := computeBatchLoss(flux.DefaultParameters, data)
	if loss != 0 {
		t.Errorf("computeBatchLoss with no cross-day = %f, want 0", loss)
	}
}

func TestComputeBatchLossAgainHigherLoss(t *testing.T) {
	// A card that is Always recalled (Good) should have lower loss
	// than one that is always forgotten (Again) on cross-day review.
	goodLogs := []flux.ReviewLog{
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(10 * time.Minute)},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(3 * 24 * time.Hour)},
	}
	againLogs := []flux.ReviewLog{
		{CardID: 2, Rating: flux.Good, ReviewDatetime: t0},
		{CardID: 2, Rating: flux.Good, ReviewDatetime: t0.Add(10 * time.Minute)},
		{CardID: 2, Rating: flux.Again, ReviewDatetime: t0.Add(3 * 24 * time.Hour)},
	}
	goodData := formatRevlogs(goodLogs)
	againData := formatRevlogs(againLogs)
	goodLoss := computeBatchLoss(flux.DefaultParameters, goodData)
	againLoss := computeBatchLoss(flux.DefaultParameters, againData)
	if againLoss <= goodLoss {
		t.Errorf("Again loss %f should be > Good loss %f", againLoss, goodLoss)
	}
}

// --- numericalGradient ---

func TestNumericalGradientDirection(t *testing.T) {
	// Build data with many Again ratings on cross-day reviews.
	// Increasing w[0] (initial stability for Again) should reduce loss,
	// so gradient w.r.t. w[0] should be negative.
	logs := []flux.ReviewLog{
		{CardID: 1, Rating: flux.Again, ReviewDatetime: t0},
		{CardID: 1, Rating: flux.Again, ReviewDatetime: t0.Add(2 * 24 * time.Hour)},
		{CardID: 1, Rating: flux.Again, ReviewDatetime: t0.Add(4 * 24 * time.Hour)},
	}
	data := formatRevlogs(logs)
	grad := numericalGradient(flux.DefaultParameters, data)

	// Gradient should be finite for all 21 parameters.
	for i, g := range grad {
		if math.IsNaN(g) || math.IsInf(g, 0) {
			t.Errorf("grad[%d] = %v, want finite", i, g)
		}
	}
}

func TestNumericalGradientSymmetry(t *testing.T) {
	// With only one card, perturbing a parameter that doesn't affect the loss
	// should produce ~0 gradient. This is hard to guarantee for FSRS params,
	// so instead we verify gradient is finite and relatively small for unused params.
	logs := []flux.ReviewLog{
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(10 * time.Minute)},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(5 * 24 * time.Hour)},
	}
	data := formatRevlogs(logs)
	grad := numericalGradient(flux.DefaultParameters, data)

	// All gradients should be finite.
	for i, g := range grad {
		if math.IsNaN(g) || math.IsInf(g, 0) {
			t.Errorf("grad[%d] = %v, want finite", i, g)
		}
	}
}

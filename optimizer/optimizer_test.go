package optimizer

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/sky-flux/flux"
)

// generateSyntheticLogs creates review logs by simulating with DefaultParameters.
// Cards are reviewed at their scheduled due time with stochastic ratings based
// on predicted retrievability.
func generateSyntheticLogs(numCards, reviewsPerCard int, seed int64) []flux.ReviewLog {
	rng := rand.New(rand.NewSource(seed))
	s, _ := flux.NewScheduler(flux.SchedulerConfig{
		Parameters:     flux.DefaultParameters,
		DisableFuzzing: true,
	})

	baseTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	var logs []flux.ReviewLog

	for i := 0; i < numCards; i++ {
		cardID := int64(i + 1)
		card := flux.NewCard(cardID)
		card.Due = baseTime
		now := baseTime

		for j := 0; j < reviewsPerCard; j++ {
			r := s.Retrievability(card, now)
			var rating flux.Rating
			if rng.Float64() > r {
				rating = flux.Again
			} else {
				p := rng.Float64()
				switch {
				case p < 0.05:
					rating = flux.Hard
				case p < 0.85:
					rating = flux.Good
				default:
					rating = flux.Easy
				}
			}

			logs = append(logs, flux.ReviewLog{
				CardID:         cardID,
				Rating:         rating,
				ReviewDatetime: now,
			})

			card, _ = s.ReviewCard(card, rating, now)
			now = card.Due
		}
	}

	return logs
}

// --- NewOptimizer ---

func TestNewOptimizerDefaults(t *testing.T) {
	o := NewOptimizer(OptimizerConfig{})
	if o.epochs != 5 {
		t.Errorf("epochs = %d, want 5", o.epochs)
	}
	if o.miniBatchSize != 512 {
		t.Errorf("miniBatchSize = %d, want 512", o.miniBatchSize)
	}
	if o.learningRate != 0.04 {
		t.Errorf("learningRate = %f, want 0.04", o.learningRate)
	}
	if o.maxSeqLen != 64 {
		t.Errorf("maxSeqLen = %d, want 64", o.maxSeqLen)
	}
}

func TestNewOptimizerCustom(t *testing.T) {
	o := NewOptimizer(OptimizerConfig{
		Epochs:        10,
		MiniBatchSize: 256,
		LearningRate:  0.01,
		MaxSeqLen:     32,
	})
	if o.epochs != 10 {
		t.Errorf("epochs = %d, want 10", o.epochs)
	}
	if o.miniBatchSize != 256 {
		t.Errorf("miniBatchSize = %d, want 256", o.miniBatchSize)
	}
	if o.learningRate != 0.01 {
		t.Errorf("learningRate = %f, want 0.01", o.learningRate)
	}
	if o.maxSeqLen != 32 {
		t.Errorf("maxSeqLen = %d, want 32", o.maxSeqLen)
	}
}

// --- ComputeOptimalParameters ---

func TestOptimizerEmptyLogs(t *testing.T) {
	o := NewOptimizer(OptimizerConfig{})
	_, err := o.ComputeOptimalParameters(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty logs")
	}
}

func TestOptimizerInsufficientData(t *testing.T) {
	o := NewOptimizer(OptimizerConfig{})
	// Only 1 cross-day review, well below MiniBatchSize=512.
	logs := []flux.ReviewLog{
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(10 * time.Minute)},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(3 * 24 * time.Hour)},
	}
	params, err := o.ComputeOptimalParameters(context.Background(), logs)
	if err == nil {
		t.Fatal("expected ErrInsufficientData")
	}
	if params != flux.DefaultParameters {
		t.Error("expected DefaultParameters for insufficient data")
	}
}

func TestOptimizerLossDecreases(t *testing.T) {
	logs := generateSyntheticLogs(300, 10, 42)
	o := NewOptimizer(OptimizerConfig{Epochs: 3})

	data := formatRevlogs(logs)
	initialLoss := computeBatchLoss(flux.DefaultParameters, data)

	optimized, err := o.ComputeOptimalParameters(context.Background(), logs)
	if err != nil {
		t.Fatalf("ComputeOptimalParameters: %v", err)
	}

	optimizedLoss := computeBatchLoss(optimized, data)
	// Optimized loss should not be significantly worse than initial.
	if optimizedLoss > initialLoss*1.01 {
		t.Errorf("optimized loss %f > initial loss %f * 1.01", optimizedLoss, initialLoss)
	}
}

func TestOptimizerParamsInBounds(t *testing.T) {
	logs := generateSyntheticLogs(300, 10, 42)
	o := NewOptimizer(OptimizerConfig{Epochs: 2})

	optimized, err := o.ComputeOptimalParameters(context.Background(), logs)
	if err != nil {
		t.Fatalf("ComputeOptimalParameters: %v", err)
	}

	for i := 0; i < 21; i++ {
		if optimized[i] < flux.LowerBounds[i] || optimized[i] > flux.UpperBounds[i] {
			t.Errorf("w[%d] = %f, out of bounds [%f, %f]",
				i, optimized[i], flux.LowerBounds[i], flux.UpperBounds[i])
		}
	}
}

func TestOptimizerContextCancel(t *testing.T) {
	logs := generateSyntheticLogs(300, 10, 42)
	o := NewOptimizer(OptimizerConfig{Epochs: 100})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := o.ComputeOptimalParameters(ctx, logs)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestOptimizerMaxSeqLen(t *testing.T) {
	// Generate data with many reviews per card, use MaxSeqLen=5 to truncate.
	// With 10 reviews per card truncated to 5, cross-day reviews still exceed MiniBatchSize.
	logs := generateSyntheticLogs(500, 10, 42)
	o := NewOptimizer(OptimizerConfig{Epochs: 1, MaxSeqLen: 5, MiniBatchSize: 64})

	_, err := o.ComputeOptimalParameters(context.Background(), logs)
	// With MaxSeqLen=3, each card has at most 2 cross-day reviews.
	// 300 cards × ~2 cross-day = ~600, should be enough for mini-batch.
	if err != nil {
		t.Fatalf("ComputeOptimalParameters with MaxSeqLen=3: %v", err)
	}
}

// --- ComputeBatchLoss (public) ---

func TestComputeBatchLossPublic(t *testing.T) {
	o := NewOptimizer(OptimizerConfig{})
	logs := []flux.ReviewLog{
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(10 * time.Minute)},
		{CardID: 1, Rating: flux.Good, ReviewDatetime: t0.Add(3 * 24 * time.Hour)},
	}
	loss := o.ComputeBatchLoss(flux.DefaultParameters, logs)
	if loss <= 0 {
		t.Errorf("ComputeBatchLoss = %f, want > 0", loss)
	}
}

func TestComputeBatchLossPublicEmpty(t *testing.T) {
	o := NewOptimizer(OptimizerConfig{})
	loss := o.ComputeBatchLoss(flux.DefaultParameters, nil)
	if loss != 0 {
		t.Errorf("ComputeBatchLoss(nil) = %f, want 0", loss)
	}
}

// --- clampParams ---

func TestClampParams(t *testing.T) {
	// Params well below lower bounds should be clamped up.
	var params [21]float64 // all zeros
	clamped := clampParams(params)
	for i := 0; i < 21; i++ {
		if clamped[i] != flux.LowerBounds[i] {
			t.Errorf("clamped[%d] = %f, want %f", i, clamped[i], flux.LowerBounds[i])
		}
	}

	// Params above upper bounds should be clamped down.
	var high [21]float64
	for i := range high {
		high[i] = 999.0
	}
	clamped = clampParams(high)
	for i := 0; i < 21; i++ {
		if clamped[i] != flux.UpperBounds[i] {
			t.Errorf("clamped[%d] = %f, want %f", i, clamped[i], flux.UpperBounds[i])
		}
	}
}

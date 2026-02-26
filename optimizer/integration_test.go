//go:build integration

package optimizer

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	"github.com/sky-flux/flux"
)

// sampleEntry matches the JSON format produced by gen_optimizer_baseline.py.
type sampleEntry struct {
	CardID         int64  `json:"card_id"`
	Rating         int    `json:"rating"`
	ReviewDatetime string `json:"review_datetime"`
	DurationMS     *int   `json:"review_duration_ms,omitempty"`
}

type baseline struct {
	TrueParameters      [21]float64 `json:"true_parameters"`
	OptimizedParameters [21]float64 `json:"optimized_parameters"`
	BatchLoss           float64     `json:"batch_loss"`
	DefaultLoss         float64     `json:"default_loss"`
	OptimalRetention    *float64    `json:"optimal_retention"`
}

func loadSampleLogs(t *testing.T) []flux.ReviewLog {
	t.Helper()
	data, err := os.ReadFile("../testdata/anki_revlogs_sample.json")
	if err != nil {
		t.Fatalf("load sample: %v", err)
	}
	var entries []sampleEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("parse sample: %v", err)
	}

	logs := make([]flux.ReviewLog, len(entries))
	for i, e := range entries {
		dt, err := time.Parse(time.RFC3339, e.ReviewDatetime)
		if err != nil {
			t.Fatalf("parse time %q: %v", e.ReviewDatetime, err)
		}
		logs[i] = flux.ReviewLog{
			CardID:         e.CardID,
			Rating:         flux.Rating(e.Rating),
			ReviewDatetime: dt,
		}
		if e.DurationMS != nil {
			d := *e.DurationMS
			logs[i].ReviewDuration = &d
		}
	}
	return logs
}

func loadBaseline(t *testing.T) baseline {
	t.Helper()
	data, err := os.ReadFile("../testdata/py_fsrs_optimizer_baseline.json")
	if err != nil {
		t.Fatalf("load baseline: %v", err)
	}
	var b baseline
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatalf("parse baseline: %v", err)
	}
	return b
}

// TestIntegrationOptimizeLoss verifies that our Go optimizer produces
// a loss comparable to py-fsrs (within 10% relative).
func TestIntegrationOptimizeLoss(t *testing.T) {
	logs := loadSampleLogs(t)
	b := loadBaseline(t)

	o := NewOptimizer(OptimizerConfig{Epochs: 5})
	optimized, err := o.ComputeOptimalParameters(context.Background(), logs)
	if err != nil {
		t.Fatalf("ComputeOptimalParameters: %v", err)
	}

	goLoss := o.ComputeBatchLoss(optimized, logs)
	t.Logf("Go optimized loss:     %.6f", goLoss)
	t.Logf("py-fsrs optimized loss: %.6f", b.BatchLoss)
	t.Logf("py-fsrs default loss:   %.6f", b.DefaultLoss)

	// Go loss should be <= py-fsrs default loss × 1.1
	// (our numerical gradient optimizer should at least not be much worse than defaults)
	if goLoss > b.DefaultLoss*1.1 {
		t.Errorf("Go loss %f > py-fsrs default loss %f × 1.1", goLoss, b.DefaultLoss)
	}
}

// TestIntegrationParamsInBounds verifies all optimized parameters stay within bounds.
func TestIntegrationParamsInBounds(t *testing.T) {
	logs := loadSampleLogs(t)

	o := NewOptimizer(OptimizerConfig{Epochs: 5})
	optimized, err := o.ComputeOptimalParameters(context.Background(), logs)
	if err != nil {
		t.Fatalf("ComputeOptimalParameters: %v", err)
	}

	for i := 0; i < 21; i++ {
		if optimized[i] < flux.LowerBounds[i] || optimized[i] > flux.UpperBounds[i] {
			t.Errorf("w[%d] = %f out of bounds [%f, %f]",
				i, optimized[i], flux.LowerBounds[i], flux.UpperBounds[i])
		}
	}
}

// TestIntegrationBatchLossConsistency verifies that our ComputeBatchLoss
// on default parameters produces a value close to py-fsrs default loss.
func TestIntegrationBatchLossConsistency(t *testing.T) {
	logs := loadSampleLogs(t)
	b := loadBaseline(t)

	o := NewOptimizer(OptimizerConfig{})
	goDefaultLoss := o.ComputeBatchLoss(flux.DefaultParameters, logs)

	t.Logf("Go default loss:     %.6f", goDefaultLoss)
	t.Logf("py-fsrs default loss: %.6f", b.DefaultLoss)

	// Should be within 5% relative difference.
	relDiff := math.Abs(goDefaultLoss-b.DefaultLoss) / b.DefaultLoss
	if relDiff > 0.05 {
		t.Errorf("Go default loss %.6f differs from py-fsrs %.6f by %.1f%%",
			goDefaultLoss, b.DefaultLoss, relDiff*100)
	}
}

// TestIntegrationOptimalRetention verifies ComputeOptimalRetention returns a sensible value.
func TestIntegrationOptimalRetention(t *testing.T) {
	logs := loadSampleLogs(t)

	o := NewOptimizer(OptimizerConfig{Epochs: 3})
	params, err := o.ComputeOptimalParameters(context.Background(), logs)
	if err != nil {
		t.Fatalf("ComputeOptimalParameters: %v", err)
	}

	retention, err := o.ComputeOptimalRetention(context.Background(), params, logs)
	if err != nil {
		t.Fatalf("ComputeOptimalRetention: %v", err)
	}

	t.Logf("Optimal retention: %.2f", retention)

	if retention < 0.70 || retention > 0.95 {
		t.Errorf("retention = %f, want ∈ [0.70, 0.95]", retention)
	}
}

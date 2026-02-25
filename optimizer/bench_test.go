package optimizer

import (
	"context"
	"testing"
)

// BenchmarkOptimize1000 measures optimization of 1000 cards × 10 reviews.
// Target: < 2s.
func BenchmarkOptimize1000(b *testing.B) {
	logs := generateSyntheticLogs(1000, 10, 42)
	o := NewOptimizer(OptimizerConfig{Epochs: 5})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := o.ComputeOptimalParameters(context.Background(), logs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOptimize10000 measures optimization of 10000 cards × 10 reviews.
// Target: < 15s.
func BenchmarkOptimize10000(b *testing.B) {
	logs := generateSyntheticLogs(10000, 10, 42)
	o := NewOptimizer(OptimizerConfig{Epochs: 5})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := o.ComputeOptimalParameters(context.Background(), logs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

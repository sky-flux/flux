package flux_test

import (
	"testing"
	"time"

	"github.com/sky-flux/flux"
)

// BenchmarkReviewCard measures the time to process a single review.
// Target: < 500ns/op.
func BenchmarkReviewCard(b *testing.B) {
	s, err := flux.NewScheduler(flux.SchedulerConfig{DisableFuzzing: true})
	if err != nil {
		b.Fatal(err)
	}
	card := flux.NewCard(1)
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Prime the card with one review so it has stability/difficulty.
	card, _ = s.ReviewCard(card, flux.Good, now)
	now = now.Add(24 * time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		card, _ = s.ReviewCard(card, flux.Good, now)
		now = now.Add(24 * time.Hour)
	}
}

// BenchmarkGetRetrievability measures the time to compute retrievability.
// Target: < 100ns/op.
func BenchmarkGetRetrievability(b *testing.B) {
	s, err := flux.NewScheduler(flux.SchedulerConfig{DisableFuzzing: true})
	if err != nil {
		b.Fatal(err)
	}
	card := flux.NewCard(1)
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	card, _ = s.ReviewCard(card, flux.Good, now)
	queryTime := now.Add(5 * 24 * time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Retrievability(card, queryTime)
	}
}

// BenchmarkPreviewCard measures the time to preview all four ratings.
// Target: < 2μs/op.
func BenchmarkPreviewCard(b *testing.B) {
	s, err := flux.NewScheduler(flux.SchedulerConfig{DisableFuzzing: true})
	if err != nil {
		b.Fatal(err)
	}
	card := flux.NewCard(1)
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	card, _ = s.ReviewCard(card, flux.Good, now)
	now = now.Add(24 * time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.PreviewCard(card, now)
	}
}

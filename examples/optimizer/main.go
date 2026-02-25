// Command optimizer demonstrates optimizing FSRS parameters from review logs
// and computing the optimal retention rate.
package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/sky-flux/flux"
	"github.com/sky-flux/flux/optimizer"
)

func main() {
	// Generate synthetic review logs for demonstration.
	logs := generateDemoLogs(200, 10, 42)
	fmt.Printf("Generated %d review logs from %d cards\n\n", len(logs), 200)

	// Optimize parameters.
	opt := optimizer.NewOptimizer(optimizer.OptimizerConfig{Epochs: 3})

	fmt.Println("Optimizing parameters...")
	params, err := opt.ComputeOptimalParameters(context.Background(), logs)
	if err != nil {
		panic(err)
	}

	// Compare loss: default vs optimized.
	defaultLoss := opt.ComputeBatchLoss(flux.DefaultParameters, logs)
	optimizedLoss := opt.ComputeBatchLoss(params, logs)
	fmt.Printf("Default parameters loss:   %.6f\n", defaultLoss)
	fmt.Printf("Optimized parameters loss: %.6f\n", optimizedLoss)
	fmt.Printf("Improvement:               %.2f%%\n\n", (defaultLoss-optimizedLoss)/defaultLoss*100)

	// Show a few optimized parameters.
	fmt.Println("Optimized parameters (first 4 = initial stability):")
	for i := 0; i < 4; i++ {
		fmt.Printf("  w[%d]: %.4f (default: %.4f)\n", i, params[i], flux.DefaultParameters[i])
	}

	// Compute optimal retention.
	fmt.Println("\nComputing optimal retention...")
	// Add durations for retention computation.
	dur := 5000
	for i := range logs {
		logs[i].ReviewDuration = &dur
	}

	retention, err := opt.ComputeOptimalRetention(context.Background(), params, logs)
	if err != nil {
		fmt.Printf("Optimal retention: skipped (%v)\n", err)
		return
	}
	fmt.Printf("Optimal retention: %.0f%%\n", retention*100)
}

// generateDemoLogs creates synthetic review logs using the default scheduler.
func generateDemoLogs(numCards, reviewsPerCard int, seed int64) []flux.ReviewLog {
	s, _ := flux.NewScheduler(flux.SchedulerConfig{DisableFuzzing: true})
	rng := rand.New(rand.NewSource(seed))
	base := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	var logs []flux.ReviewLog
	for i := 0; i < numCards; i++ {
		card := flux.NewCard(int64(i + 1))
		card.Due = base
		now := base

		for j := 0; j < reviewsPerCard; j++ {
			r := s.Retrievability(card, now)
			var rating flux.Rating
			if rng.Float64() >= r {
				rating = flux.Again
			} else {
				p := rng.Float64()
				switch {
				case p < 0.1:
					rating = flux.Hard
				case p < 0.85:
					rating = flux.Good
				default:
					rating = flux.Easy
				}
			}

			logs = append(logs, flux.ReviewLog{
				CardID:         card.CardID,
				Rating:         rating,
				ReviewDatetime: now,
			})
			card, _ = s.ReviewCard(card, rating, now)
			now = card.Due
		}
	}
	return logs
}

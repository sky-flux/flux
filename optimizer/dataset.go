package optimizer

import (
	"sort"
	"time"

	"github.com/sky-flux/flux"
)

// review is an internal representation of a single review event for training.
type review struct {
	rating      flux.Rating
	elapsedDays float64   // days since previous review (0 for first)
	label       float64   // 0 if Again, 1 otherwise
	reviewTime  time.Time // original review timestamp (for Scheduler replay)
}

// formatRevlogs groups review logs by card ID and sorts each group by time.
// Each review computes elapsed_days from the previous review and a binary label.
func formatRevlogs(logs []flux.ReviewLog) map[int64][]review {
	if len(logs) == 0 {
		return nil
	}

	// Group by card ID.
	groups := make(map[int64][]flux.ReviewLog)
	for _, log := range logs {
		groups[log.CardID] = append(groups[log.CardID], log)
	}

	result := make(map[int64][]review, len(groups))
	for cardID, cardLogs := range groups {
		// Sort by review time.
		sort.Slice(cardLogs, func(i, j int) bool {
			return cardLogs[i].ReviewDatetime.Before(cardLogs[j].ReviewDatetime)
		})

		reviews := make([]review, len(cardLogs))
		for i, log := range cardLogs {
			var elapsed float64
			if i > 0 {
				elapsed = log.ReviewDatetime.Sub(cardLogs[i-1].ReviewDatetime).Hours() / 24.0
			}

			label := 1.0
			if log.Rating == flux.Again {
				label = 0.0
			}

			reviews[i] = review{
				rating:      log.Rating,
				elapsedDays: elapsed,
				label:       label,
				reviewTime:  log.ReviewDatetime,
			}
		}
		result[cardID] = reviews
	}

	return result
}

// countCrossDayReviews counts reviews where elapsed_days >= 1 (cross-day reviews).
// The first review of each card is never cross-day (elapsed_days = 0).
func countCrossDayReviews(data map[int64][]review) int {
	count := 0
	for _, reviews := range data {
		for _, r := range reviews {
			if r.elapsedDays >= 1.0 {
				count++
			}
		}
	}
	return count
}

package optimizer

import (
	"math"

	"github.com/sky-flux/flux"
)

const bceClamp = 1e-7

// bceLoss computes the binary cross-entropy loss: -[y*ln(p) + (1-y)*ln(1-p)].
// rPred is clamped to [bceClamp, 1-bceClamp] to avoid log(0).
func bceLoss(rPred, y float64) float64 {
	p := math.Max(bceClamp, math.Min(rPred, 1-bceClamp))
	return -(y*math.Log(p) + (1-y)*math.Log(1-p))
}

// computeBatchLoss computes the average BCE loss over all cross-day reviews.
// It creates a Scheduler from params and replays each card's review history.
// Returns 0 if there are no cross-day reviews.
func computeBatchLoss(params [21]float64, data map[int64][]review) float64 {
	s, err := flux.NewScheduler(flux.SchedulerConfig{
		Parameters:     params,
		DisableFuzzing: true,
	})
	if err != nil {
		return 0
	}

	var totalLoss float64
	var count int

	for cardID, reviews := range data {
		card := flux.NewCard(cardID)
		card.Due = reviews[0].reviewTime

		for _, rev := range reviews {
			// Compute retrievability BEFORE this review.
			rPred := s.Retrievability(card, rev.reviewTime)

			// Only cross-day reviews contribute to loss.
			if card.LastReview != nil && rev.elapsedDays >= 1.0 {
				totalLoss += bceLoss(rPred, rev.label)
				count++
			}

			// Update card state.
			card, _ = s.ReviewCard(card, rev.rating, rev.reviewTime)
		}
	}

	if count == 0 {
		return 0
	}
	return totalLoss / float64(count)
}

const gradEps = 1e-5

// numericalGradient computes the gradient of the batch loss w.r.t. each parameter
// using central differences: dL/dw[i] ≈ (L(w[i]+ε) - L(w[i]-ε)) / (2ε).
func numericalGradient(params [21]float64, data map[int64][]review) [21]float64 {
	var grad [21]float64
	for i := 0; i < 21; i++ {
		pPlus := params
		pPlus[i] += gradEps
		pMinus := params
		pMinus[i] -= gradEps

		lPlus := computeBatchLoss(pPlus, data)
		lMinus := computeBatchLoss(pMinus, data)

		grad[i] = (lPlus - lMinus) / (2 * gradEps)
	}
	return grad
}

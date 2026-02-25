package optimizer

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"sort"

	"github.com/sky-flux/flux"
)

var (
	// ErrEmptyLogs is returned when no review logs are provided.
	ErrEmptyLogs = errors.New("optimizer: no review logs provided")

	// ErrInsufficientData is returned when cross-day reviews are fewer than MiniBatchSize.
	ErrInsufficientData = errors.New("optimizer: insufficient cross-day reviews for optimization")
)

// OptimizerConfig configures the training process.
// Zero values are replaced with sensible defaults.
type OptimizerConfig struct {
	Epochs        int     `json:"epochs"`          // default 5
	MiniBatchSize int     `json:"mini_batch_size"` // default 512
	LearningRate  float64 `json:"learning_rate"`   // default 0.04
	MaxSeqLen     int     `json:"max_seq_len"`     // default 64
}

// Optimizer trains FSRS parameters from review logs using mini-batch
// gradient descent with Adam and cosine annealing learning rate.
type Optimizer struct {
	epochs        int
	miniBatchSize int
	learningRate  float64
	maxSeqLen     int
}

// NewOptimizer creates an Optimizer with the given config.
// Zero-valued fields receive defaults: Epochs=5, MiniBatchSize=512,
// LearningRate=0.04, MaxSeqLen=64.
func NewOptimizer(cfg OptimizerConfig) *Optimizer {
	o := &Optimizer{
		epochs:        cfg.Epochs,
		miniBatchSize: cfg.MiniBatchSize,
		learningRate:  cfg.LearningRate,
		maxSeqLen:     cfg.MaxSeqLen,
	}
	if o.epochs == 0 {
		o.epochs = 5
	}
	if o.miniBatchSize == 0 {
		o.miniBatchSize = 512
	}
	if o.learningRate == 0 {
		o.learningRate = 0.04
	}
	if o.maxSeqLen == 0 {
		o.maxSeqLen = 64
	}
	return o
}

// ComputeOptimalParameters optimizes FSRS parameters from review logs.
// It starts from DefaultParameters and uses mini-batch gradient descent
// (numerical central differences) with Adam optimizer and cosine annealing LR.
//
// Returns ErrEmptyLogs if logs is empty, or ErrInsufficientData (along with
// DefaultParameters) if cross-day reviews are fewer than MiniBatchSize.
// The context can be used to cancel long-running optimization.
func (o *Optimizer) ComputeOptimalParameters(ctx context.Context, logs []flux.ReviewLog) ([21]float64, error) {
	if len(logs) == 0 {
		return [21]float64{}, ErrEmptyLogs
	}

	data := formatRevlogs(logs)

	// Truncate each card's reviews to maxSeqLen.
	for cardID, reviews := range data {
		if len(reviews) > o.maxSeqLen {
			data[cardID] = reviews[:o.maxSeqLen]
		}
	}

	numReviews := countCrossDayReviews(data)
	if numReviews < o.miniBatchSize {
		return flux.DefaultParameters, ErrInsufficientData
	}

	params := flux.DefaultParameters
	tMax := int(math.Ceil(float64(numReviews)/float64(o.miniBatchSize))) * o.epochs
	adam := NewAdam(o.learningRate)
	ca := NewCosineAnnealing(o.learningRate, tMax)
	rng := rand.New(rand.NewSource(42))

	// Sorted card IDs for deterministic shuffle.
	cardIDs := make([]int64, 0, len(data))
	for id := range data {
		cardIDs = append(cardIDs, id)
	}
	sort.Slice(cardIDs, func(i, j int) bool { return cardIDs[i] < cardIDs[j] })

	bestParams := params
	bestLoss := math.Inf(1)

	for epoch := 0; epoch < o.epochs; epoch++ {
		if err := ctx.Err(); err != nil {
			return bestParams, err
		}

		rng.Shuffle(len(cardIDs), func(i, j int) {
			cardIDs[i], cardIDs[j] = cardIDs[j], cardIDs[i]
		})

		batchData := make(map[int64][]review)
		crossDayCount := 0

		for _, cardID := range cardIDs {
			reviews := data[cardID]
			batchData[cardID] = reviews

			for _, r := range reviews {
				if r.elapsedDays >= 1.0 {
					crossDayCount++
				}
			}

			if crossDayCount >= o.miniBatchSize {
				grad := numericalGradient(params, batchData)
				adam.SetLR(ca.LR())
				params = adam.Update(params, grad)
				params = clampParams(params)
				ca.Step()

				batchData = make(map[int64][]review)
				crossDayCount = 0
			}
		}

		// Handle remaining reviews at end of epoch.
		if crossDayCount > 0 {
			grad := numericalGradient(params, batchData)
			adam.SetLR(ca.LR())
			params = adam.Update(params, grad)
			params = clampParams(params)
			ca.Step()
		}

		// Track best parameters by epoch loss.
		epochLoss := computeBatchLoss(params, data)
		if epochLoss < bestLoss {
			bestLoss = epochLoss
			bestParams = params
		}
	}

	return bestParams, nil
}

// ComputeBatchLoss computes the average BCE loss over all cross-day reviews.
// This is a convenience wrapper that preprocesses the review logs.
func (o *Optimizer) ComputeBatchLoss(params [21]float64, logs []flux.ReviewLog) float64 {
	data := formatRevlogs(logs)
	return computeBatchLoss(params, data)
}

// clampParams constrains each parameter to [LowerBounds, UpperBounds].
func clampParams(params [21]float64) [21]float64 {
	for i := 0; i < 21; i++ {
		if params[i] < flux.LowerBounds[i] {
			params[i] = flux.LowerBounds[i]
		}
		if params[i] > flux.UpperBounds[i] {
			params[i] = flux.UpperBounds[i]
		}
	}
	return params
}

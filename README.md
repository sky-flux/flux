# flux

[![CI](https://img.shields.io/github/actions/workflow/status/sky-flux/flux/ci.yml?branch=main&label=CI)](https://github.com/sky-flux/flux/actions)
[![codecov](https://codecov.io/github/sky-flux/flux/graph/badge.svg?token=YT941R23LJ)](https://codecov.io/github/sky-flux/flux)
[![Go Report Card](https://goreportcard.com/badge/github.com/sky-flux/flux)](https://goreportcard.com/report/github.com/sky-flux/flux)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Pure Go implementation of the **FSRS v6** spaced repetition algorithm. Zero dependencies outside the standard library.

flux provides a complete scheduling engine for flashcard applications: review scheduling, memory state tracking, parameter optimization from review history, and optimal retention computation via Monte Carlo simulation.

## Features

- **FSRS v6 algorithm** -- full implementation of the latest Free Spaced Repetition Scheduler with 21 trainable parameters
- **Zero dependencies** -- only the Go standard library
- **Card lifecycle management** -- Learning, Review, and Relearning states with configurable step durations
- **Parameter optimizer** -- mini-batch gradient descent with Adam and cosine annealing to train parameters from review logs
- **Optimal retention** -- Monte Carlo simulation to find the retention target that minimizes total review cost
- **Retrievability** -- compute recall probability for any card at any point in time
- **Preview & reschedule** -- preview all rating outcomes before committing, or replay review logs to rebuild card state
- **Interval fuzzing** -- optional randomization to spread reviews and avoid clustering
- **JSON serialization** -- Card, Rating, State, Scheduler, and ReviewLog all implement JSON marshaling
- **Deterministic & testable** -- fuzzing can be disabled for reproducible tests

## Quick Start

```bash
go get github.com/sky-flux/flux
```

Create a card, review it multiple times, and watch the scheduling adapt:

```go
s, _ := flux.NewScheduler(flux.SchedulerConfig{DesiredRetention: 0.9})
card := flux.NewCard(1)
now := time.Now()

// First review — card moves through Learning steps
card, _ = s.ReviewCard(card, flux.Good, now)
fmt.Println(card.State) // Learning (one more step)

// Second review — graduates to Review
card, _ = s.ReviewCard(card, flux.Good, card.Due)
fmt.Println(card.State) // Review
fmt.Println(card.Due)   // ~2 days from now

// Third review — interval grows with each successful recall
card, _ = s.ReviewCard(card, flux.Good, card.Due)
fmt.Println(card.Due)   // ~10 days from now

// Check recall probability at any point
r := s.Retrievability(card, card.Due)
fmt.Printf("%.0f%%\n", r*100) // ~90% (matches DesiredRetention)

// Preview all four rating outcomes before committing
preview := s.PreviewCard(card, card.Due)
for _, rating := range []flux.Rating{flux.Again, flux.Hard, flux.Good, flux.Easy} {
    fmt.Printf("%s → %s\n", rating, preview[rating].Due)
}
```

See [`examples/`](examples/) for complete runnable programs covering the basic lifecycle, parameter optimization, and review log rescheduling.

## API Overview

### Core Types

```go
// Card holds the scheduling state for a single flashcard.
type Card struct {
    CardID     int64
    State      State      // Learning, Review, or Relearning
    Step       *int       // current learning/relearning step (nil in Review)
    Stability  *float64   // memory stability in days (nil before first review)
    Difficulty *float64   // item difficulty (nil before first review)
    Due        time.Time
    LastReview *time.Time
}

// Rating represents the user's recall assessment.
type Rating int // Again=1, Hard=2, Good=3, Easy=4

// State represents the learning stage of a card.
type State int // Learning=1, Review=2, Relearning=3

// ReviewLog records a single review event.
type ReviewLog struct {
    CardID         int64
    Rating         Rating
    ReviewDatetime time.Time
    ReviewDuration *int // milliseconds, optional
}
```

### Scheduler

```go
func NewScheduler(cfg SchedulerConfig) (*Scheduler, error)
```

| Method | Description |
|--------|-------------|
| `ReviewCard(card Card, rating Rating, now time.Time) (Card, ReviewLog)` | Process a review and return the updated card and log |
| `PreviewCard(card Card, now time.Time) map[Rating]Card` | Preview outcomes for all four ratings |
| `RescheduleCard(card Card, logs []ReviewLog) (Card, error)` | Replay review logs to rebuild card state |
| `Retrievability(card Card, now time.Time) float64` | Compute recall probability at a given time |

### SchedulerConfig

```go
type SchedulerConfig struct {
    Parameters       [21]float64     // zero -> DefaultParameters
    DesiredRetention float64         // zero -> 0.9
    LearningSteps    []time.Duration // nil -> [1m, 10m]
    RelearningSteps  []time.Duration // nil -> [10m]
    MaximumInterval  int             // zero -> 36500 days
    DisableFuzzing   bool            // zero -> false (fuzzing enabled)
}
```

### Parameters

```go
var DefaultParameters [21]float64 // FSRS v6 defaults from py-fsrs
var LowerBounds [21]float64
var UpperBounds [21]float64

func ValidateParameters(p [21]float64) error
```

## Optimizer

The `optimizer` sub-package trains FSRS parameters from real review history and computes optimal retention targets.

```go
import "github.com/sky-flux/flux/optimizer"

// Collect review logs from your application (e.g. from a database).
// Each log records which card was reviewed, the rating, and when.
logs := []flux.ReviewLog{
    {CardID: 1, Rating: flux.Good, ReviewDatetime: day1},
    {CardID: 1, Rating: flux.Good, ReviewDatetime: day3},
    // ... hundreds or thousands of real reviews
}

opt := optimizer.NewOptimizer(optimizer.OptimizerConfig{})

// Train personalized parameters from review history
params, err := opt.ComputeOptimalParameters(ctx, logs)

// Use the optimized parameters in a new scheduler
s, _ := flux.NewScheduler(flux.SchedulerConfig{Parameters: params})

// Optionally: find the retention target that minimizes total review cost.
// Requires ReviewDuration to be set on each log.
retention, err := opt.ComputeOptimalRetention(ctx, params, logs)
```

### OptimizerConfig

| Field | Default | Description |
|-------|---------|-------------|
| `Epochs` | 5 | Training epochs |
| `MiniBatchSize` | 512 | Reviews per mini-batch |
| `LearningRate` | 0.04 | Initial Adam learning rate |
| `MaxSeqLen` | 64 | Max reviews per card |

## Performance

Benchmarks run on Apple M-series silicon. All targets met.

| Benchmark | Result | Target |
|-----------|--------|--------|
| ReviewCard | 182 ns/op | < 500 ns |
| GetRetrievability | 26 ns/op | < 100 ns |
| PreviewCard | 814 ns/op | < 2 us |
| Optimize1000 | 0.50 s | < 2 s |
| Optimize10000 | 4.61 s | < 15 s |

## Alignment with py-fsrs

flux is a line-by-line port of the reference [py-fsrs](https://github.com/open-spaced-repetition/py-fsrs) Python implementation. All 21 FSRS v6 parameters, the memory state equations, the stability/difficulty update formulas, and the interval calculation logic match the Python reference. The test suite validates output parity against py-fsrs for the same inputs and parameter sets.

## Examples

The [`examples/`](examples/) directory contains complete runnable programs:

| Example | Description | Run |
|---------|-------------|-----|
| [`basic`](examples/basic/) | Card creation, review loop, preview | `go run ./examples/basic/` |
| [`optimizer`](examples/optimizer/) | Parameter training, optimal retention | `go run ./examples/optimizer/` |
| [`reschedule`](examples/reschedule/) | Replay review logs to rebuild state | `go run ./examples/reschedule/` |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to contribute to this project.

## License

[MIT](LICENSE)

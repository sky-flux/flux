# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [v0.8.0] - 2026-02-25

### Added

- README.md with full project documentation
- CONTRIBUTING.md with development guidelines
- CHANGELOG.md

## [v0.7.0] - 2026-02-25

### Added

- `bench_test.go` — Scheduler benchmarks (ReviewCard 182ns, Retrievability 26ns, PreviewCard 814ns)
- `optimizer/bench_test.go` — Optimizer benchmarks (1000 cards 0.5s, 10000 cards 4.6s)
- `examples/basic/main.go` — Card lifecycle demo
- `examples/optimizer/main.go` — Parameter optimization demo
- `examples/reschedule/main.go` — Review log replay demo

## [v0.6.0] - 2026-02-25

### Added

- `optimizer/retention.go` — ComputeOptimalRetention via Monte Carlo simulation
- `optimizer/integration_test.go` — Cross-validation with py-fsrs optimizer baseline
- `scripts/gen_optimizer_baseline.py` — Synthetic data generation for integration tests
- ReviewDuration field on ReviewLog

## [v0.5.0] - 2026-02-25

### Added

- `optimizer/optimizer.go` — ComputeOptimalParameters training pipeline
- Mini-batch gradient descent with Adam optimizer and cosine annealing LR
- ComputeBatchLoss public API

## [v0.4.0] - 2026-02-25

### Added

- `optimizer/dataset.go` — Review log preprocessing (formatRevlogs, cross-day review extraction)
- `optimizer/loss.go` — BCE loss computation and numerical gradient
- `optimizer/adam.go` — Adam optimizer with bias correction and cosine annealing LR scheduler

## [v0.3.0] - 2026-02-25

### Added

- py-fsrs alignment test scenarios (cross-validation with Python FSRS reference implementation)
- Design specification document (flux.md)

## [v0.2.0] - 2026-02-25

### Added

- `Scheduler` with full FSRS v6 state machine (Learning -> Review -> Relearning)
- `ReviewCard`, `PreviewCard`, `RescheduleCard`, `Retrievability` methods
- Interval fuzzing with three-band algorithm
- JSON serialization for Scheduler and Card
- Configurable learning/relearning steps, desired retention, maximum interval

## [v0.1.0] - 2026-02-25

### Added

- FSRS v6 algorithm kernel (21-parameter model)
- `Card` and `ReviewLog` types
- `Rating` (Again/Hard/Good/Easy) and `State` (Learning/Review/Relearning) enums
- `DefaultParameters`, `LowerBounds`, `UpperBounds`, `ValidateParameters`
- Initial stability, difficulty, recall/forget stability computations
- Parameter validation with bounds checking

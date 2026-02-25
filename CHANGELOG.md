# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [v1.0.2] - 2026-02-25

### Changed

- CI: upgrade golangci-lint-action v6 → v9 (golangci-lint v1 → v2, Go 1.26 compatible)

## [v1.0.1] - 2026-02-25

### Fixed

- CI: temporarily disable golangci-lint (v1.64 built with Go 1.24, incompatible with Go 1.26)
- CI: add `fail-fast: false` to prevent matrix cancellation on partial failure

## [v1.0.0] - 2026-02-25

### Summary

First stable release. API frozen, production ready.

- Pure Go FSRS v6 implementation with 21 trainable parameters
- Scheduler with full state machine (Learning, Review, Relearning)
- Parameter optimizer with Adam and cosine annealing LR
- Optimal retention via Monte Carlo simulation
- 100% test coverage, all benchmarks within targets
- Cross-validated against py-fsrs reference implementation
- Zero external dependencies

## [v0.9.2] - 2026-02-25

### Changed

- Expanded README Quick Start with full card lifecycle (multi-review, state transitions, retrievability, preview)
- Enhanced Optimizer section with ReviewLog construction guidance and end-to-end workflow
- Added Examples section linking to all three runnable programs

## [v0.9.1] - 2026-02-25

### Fixed

- Added missing v0.9.0 entry to CHANGELOG
- Updated flux.md specification checklist to reflect completed work
- Fixed testdata file format references (parquet → json) in flux.md
- Updated dependency description (removed gonum, stdlib only) in flux.md

### Deferred

- `CODE_OF_CONDUCT.md` and `SECURITY.md` deferred to v1.0.0 — project is pre-release with a single maintainer; community governance docs will be added when the project opens for external contributions

## [v0.9.0] - 2026-02-25

### Added

- `.github/workflows/ci.yml` — CI with Go 1.26/stable matrix, vet, lint, race detection, 100% coverage gate, integration tests
- `.github/workflows/release.yml` — Tag-triggered release workflow
- `.github/ISSUE_TEMPLATE/bug_report.md` and `feature_request.md` — Issue templates
- `.github/PULL_REQUEST_TEMPLATE.md` — PR template with testing checklist
- `Makefile` with test, cover, lint, bench, vet, examples targets

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

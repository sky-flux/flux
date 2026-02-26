// Package flux implements the FSRS v6 spaced repetition scheduling algorithm.
//
// flux is a pure-Go, zero-dependency implementation of the Free Spaced
// Repetition Scheduler (FSRS) version 6 — the algorithm used by Anki
// (via fsrs4anki) and SiYuan. It provides a complete scheduling engine
// for flashcard applications with a 21-parameter trainable model.
//
// # Core Concepts
//
//   - [Card] holds the scheduling state: stability, difficulty, due date, and learning step.
//   - [Scheduler] applies the FSRS v6 state machine (Learning → Review → Relearning)
//     to compute review intervals.
//   - [Rating] (Again, Hard, Good, Easy) is the user's recall assessment.
//   - [ReviewLog] records each review event for later optimization.
//
// # Basic Usage
//
//	s, err := flux.NewScheduler(flux.SchedulerConfig{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	card := flux.NewCard(1)
//	card, reviewLog := s.ReviewCard(card, flux.Good, time.Now())
//
//	// Preview all possible outcomes before the user answers.
//	preview := s.PreviewCard(card, time.Now())
//
//	// Rebuild card state from historical review logs.
//	card, err = s.RescheduleCard(flux.NewCard(1), logs)
//
//	// Check recall probability at any point in time.
//	r := s.Retrievability(card, time.Now())
//
// # Optimizer
//
// The [flux/optimizer] subpackage trains optimal parameters from review
// history using mini-batch gradient descent with Adam and cosine annealing.
// See [optimizer.Optimizer] for details.
//
// # Cross-Validation
//
// All outputs are cross-validated against the py-fsrs reference implementation
// to ensure algorithmic correctness.
package flux

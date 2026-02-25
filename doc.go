// Package flux implements the FSRS v6 spaced repetition scheduling algorithm.
//
// flux provides a pure-Go Scheduler for computing optimal review intervals
// and an Optimizer (in the flux/optimizer subpackage) for training FSRS
// parameters from historical review logs.
//
// Basic usage:
//
//	s, err := flux.NewScheduler(flux.SchedulerConfig{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	card := flux.NewCard(1)
//	card, log := s.ReviewCard(card, flux.Good, time.Now())
package flux

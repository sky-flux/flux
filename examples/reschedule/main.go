// Command reschedule demonstrates replaying review logs to rebuild a card's scheduling state.
package main

import (
	"fmt"
	"time"

	"github.com/sky-flux/flux"
)

func main() {
	// Imagine these review logs were exported from an SRS application.
	logs := []flux.ReviewLog{
		{CardID: 42, Rating: flux.Good, ReviewDatetime: date(2024, 1, 1)},
		{CardID: 42, Rating: flux.Good, ReviewDatetime: date(2024, 1, 1).Add(10 * time.Minute)},
		{CardID: 42, Rating: flux.Good, ReviewDatetime: date(2024, 1, 3)},
		{CardID: 42, Rating: flux.Hard, ReviewDatetime: date(2024, 1, 10)},
		{CardID: 42, Rating: flux.Good, ReviewDatetime: date(2024, 1, 20)},
		{CardID: 42, Rating: flux.Easy, ReviewDatetime: date(2024, 3, 1)},
	}

	// Create a scheduler with custom parameters (or use defaults).
	s, err := flux.NewScheduler(flux.SchedulerConfig{
		DesiredRetention: 0.9,
		DisableFuzzing:   true,
	})
	if err != nil {
		panic(err)
	}

	// Reschedule: replay all logs to rebuild card state.
	card := flux.NewCard(42)
	card, err = s.RescheduleCard(card, logs)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== Rescheduled Card ===")
	fmt.Printf("Card ID:    %d\n", card.CardID)
	fmt.Printf("State:      %s\n", card.State)
	fmt.Printf("Due:        %s\n", card.Due.Format(time.DateOnly))
	if card.Stability != nil {
		fmt.Printf("Stability:  %.2f days\n", *card.Stability)
	}
	if card.Difficulty != nil {
		fmt.Printf("Difficulty: %.2f\n", *card.Difficulty)
	}

	// Check retrievability at a specific point.
	checkDate := date(2024, 3, 15)
	r := s.Retrievability(card, checkDate)
	fmt.Printf("\nRetrievability on %s: %.1f%%\n", checkDate.Format(time.DateOnly), r*100)
}

func date(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 10, 0, 0, 0, time.UTC)
}

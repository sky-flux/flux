// Command basic demonstrates creating a card, reviewing it, and checking the due date.
package main

import (
	"fmt"
	"time"

	"github.com/sky-flux/flux"
)

func main() {
	s, err := flux.NewScheduler(flux.SchedulerConfig{
		DesiredRetention: 0.9,
	})
	if err != nil {
		panic(err)
	}

	card := flux.NewCard(1)
	now := time.Now().Truncate(time.Second)

	fmt.Println("=== New Card ===")
	fmt.Printf("State: %s, Due: %s\n\n", card.State, card.Due.Format(time.DateTime))

	// Simulate a sequence of reviews.
	ratings := []flux.Rating{flux.Good, flux.Good, flux.Good, flux.Easy}
	for i, rating := range ratings {
		card, _ = s.ReviewCard(card, rating, now)

		fmt.Printf("Review %d: rated %s\n", i+1, rating)
		fmt.Printf("  State:      %s\n", card.State)
		fmt.Printf("  Due:        %s\n", card.Due.Format(time.DateTime))
		if card.Stability != nil {
			fmt.Printf("  Stability:  %.2f days\n", *card.Stability)
		}
		if card.Difficulty != nil {
			fmt.Printf("  Difficulty: %.2f\n", *card.Difficulty)
		}

		r := s.Retrievability(card, now)
		fmt.Printf("  Recall %%:   %.1f%%\n\n", r*100)

		now = card.Due
	}

	// Preview what would happen with each rating.
	fmt.Println("=== Preview Next Review ===")
	preview := s.PreviewCard(card, now)
	for _, r := range []flux.Rating{flux.Again, flux.Hard, flux.Good, flux.Easy} {
		c := preview[r]
		fmt.Printf("  %s → Due: %s\n", r, c.Due.Format(time.DateTime))
	}
}

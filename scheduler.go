package flux

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// SchedulerConfig configures a Scheduler.
// Zero values produce sensible defaults; see field comments.
type SchedulerConfig struct {
	Parameters       [21]float64     `json:"parameters"`        // zero → DefaultParameters
	DesiredRetention float64         `json:"desired_retention"` // zero → 0.9
	LearningSteps    []time.Duration `json:"learning_steps"`    // nil → [1m, 10m]; empty → no steps
	RelearningSteps  []time.Duration `json:"relearning_steps"`  // nil → [10m]; empty → no steps
	MaximumInterval  int             `json:"maximum_interval"`  // zero → 36500
	DisableFuzzing   bool            `json:"disable_fuzzing"`   // zero false → fuzz enabled
}

// Scheduler schedules card reviews using the FSRS v6 algorithm.
type Scheduler struct {
	algo             algo
	desiredRetention float64
	learningSteps    []time.Duration
	relearningSteps  []time.Duration
	maximumInterval  int
	disableFuzzing   bool
	rng              *rand.Rand
}

// NewScheduler creates a Scheduler from the given config.
// Zero-value fields are filled with defaults; invalid values return an error.
func NewScheduler(cfg SchedulerConfig) (*Scheduler, error) {
	// Parameters: zero array → defaults.
	params := cfg.Parameters
	if params == [21]float64{} {
		params = DefaultParameters
	}
	if err := ValidateParameters(params); err != nil {
		return nil, err
	}

	// DesiredRetention: zero → 0.9.
	dr := cfg.DesiredRetention
	if dr == 0 {
		dr = 0.9
	}
	if dr < 0 || dr > 1 {
		return nil, fmt.Errorf("flux: desired retention %f out of range (0, 1]", dr)
	}

	// MaximumInterval: zero → 36500.
	maxIvl := cfg.MaximumInterval
	if maxIvl == 0 {
		maxIvl = 36500
	}
	if maxIvl < 0 {
		return nil, fmt.Errorf("flux: maximum interval %d must be positive", maxIvl)
	}

	// LearningSteps: nil → defaults.
	ls := cfg.LearningSteps
	if ls == nil {
		ls = []time.Duration{time.Minute, 10 * time.Minute}
	}

	// RelearningSteps: nil → defaults.
	rs := cfg.RelearningSteps
	if rs == nil {
		rs = []time.Duration{10 * time.Minute}
	}

	return &Scheduler{
		algo:             newAlgo(params),
		desiredRetention: dr,
		learningSteps:    ls,
		relearningSteps:  rs,
		maximumInterval:  maxIvl,
		disableFuzzing:   cfg.DisableFuzzing,
		rng:              rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

// ReviewCard processes a review of the card at the given time.
// It returns the updated card and a review log. The input card is not mutated.
func (s *Scheduler) ReviewCard(card Card, rating Rating, now time.Time) (Card, ReviewLog) {
	c := card.clone()

	// Compute elapsed days since last review.
	var elapsedDays float64
	if c.LastReview != nil {
		elapsedDays = now.Sub(*c.LastReview).Hours() / 24.0
	}

	// Update stability and difficulty.
	s.updateMemory(&c, rating, elapsedDays)

	// Determine steps for current state.
	steps := s.stepsForState(c.State)

	// State transition and interval.
	interval := s.transition(&c, rating, steps)

	// Apply fuzz if enabled and final state is Review.
	if !s.disableFuzzing && c.State == Review {
		days := int(interval.Hours() / 24.0)
		if days > 0 {
			fuzzed := applyFuzz(days, s.maximumInterval, s.rng)
			interval = time.Duration(fuzzed) * 24 * time.Hour
		}
	}

	c.Due = now.Add(interval)
	c.LastReview = &now

	log := ReviewLog{
		CardID:         c.CardID,
		Rating:         rating,
		ReviewDatetime: now,
	}

	return c, log
}

// PreviewCard returns the result of reviewing the card with each possible rating.
func (s *Scheduler) PreviewCard(card Card, now time.Time) map[Rating]Card {
	result := make(map[Rating]Card, 4)
	for _, r := range []Rating{Again, Hard, Good, Easy} {
		c, _ := s.ReviewCard(card, r, now)
		result[r] = c
	}
	return result
}

// RescheduleCard replays the given review logs to rebuild the card's scheduling state.
// Returns ErrCardIDMismatch if any log's CardID does not match the card's CardID.
func (s *Scheduler) RescheduleCard(card Card, logs []ReviewLog) (Card, error) {
	c := card.clone()
	for _, log := range logs {
		if log.CardID != c.CardID {
			return Card{}, fmt.Errorf("%w: card %d, log %d", ErrCardIDMismatch, c.CardID, log.CardID)
		}
		c, _ = s.ReviewCard(c, log.Rating, log.ReviewDatetime)
	}
	return c, nil
}

// schedulerJSON is the serialized form of a Scheduler.
type schedulerJSON struct {
	Parameters       [21]float64 `json:"parameters"`
	DesiredRetention float64     `json:"desired_retention"`
	LearningSteps    []int64     `json:"learning_steps"`    // nanoseconds
	RelearningSteps  []int64     `json:"relearning_steps"`  // nanoseconds
	MaximumInterval  int         `json:"maximum_interval"`
	DisableFuzzing   bool        `json:"disable_fuzzing"`
}

// MarshalJSON implements json.Marshaler.
func (s *Scheduler) MarshalJSON() ([]byte, error) {
	j := schedulerJSON{
		Parameters:       s.algo.w,
		DesiredRetention: s.desiredRetention,
		MaximumInterval:  s.maximumInterval,
		DisableFuzzing:   s.disableFuzzing,
	}
	j.LearningSteps = durationsToNanos(s.learningSteps)
	j.RelearningSteps = durationsToNanos(s.relearningSteps)
	return json.Marshal(j)
}

// UnmarshalJSON implements json.Unmarshaler.
// It rebuilds the internal precomputed state from the serialized config.
func (s *Scheduler) UnmarshalJSON(data []byte) error {
	var j schedulerJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	cfg := SchedulerConfig{
		Parameters:       j.Parameters,
		DesiredRetention: j.DesiredRetention,
		MaximumInterval:  j.MaximumInterval,
		DisableFuzzing:   j.DisableFuzzing,
		LearningSteps:    nanosToDurations(j.LearningSteps),
		RelearningSteps:  nanosToDurations(j.RelearningSteps),
	}
	rebuilt, err := NewScheduler(cfg)
	if err != nil {
		return err
	}
	*s = *rebuilt
	return nil
}

func durationsToNanos(ds []time.Duration) []int64 {
	ns := make([]int64, len(ds))
	for i, d := range ds {
		ns[i] = int64(d)
	}
	return ns
}

func nanosToDurations(ns []int64) []time.Duration {
	if ns == nil {
		return nil
	}
	ds := make([]time.Duration, len(ns))
	for i, n := range ns {
		ds[i] = time.Duration(n)
	}
	return ds
}

// Retrievability returns the probability of recall for the card at the given time.
// Returns 0 if the card has never been reviewed or has no stability.
func (s *Scheduler) Retrievability(card Card, now time.Time) float64 {
	if card.LastReview == nil || card.Stability == nil {
		return 0
	}
	elapsed := now.Sub(*card.LastReview).Hours() / 24.0
	return s.algo.retrievability(elapsed, *card.Stability)
}

// updateMemory updates the card's stability and difficulty based on the review.
func (s *Scheduler) updateMemory(c *Card, rating Rating, elapsedDays float64) {
	if c.Stability == nil {
		// First review: initialize S and D.
		c.setStability(s.algo.initStability(rating))
		c.setDifficulty(s.algo.initDifficulty(rating, true))
		return
	}

	stability := *c.Stability
	difficulty := *c.Difficulty

	if elapsedDays < 1 {
		// Same-day review.
		c.setStability(s.algo.shortTermStability(stability, rating))
	} else {
		// Cross-day review.
		r := s.algo.retrievability(elapsedDays, stability)
		c.setStability(s.algo.nextStability(difficulty, stability, r, rating))
	}
	c.setDifficulty(s.algo.nextDifficulty(difficulty, rating))
}

// stepsForState returns the step durations for the given state.
func (s *Scheduler) stepsForState(state State) []time.Duration {
	switch state {
	case Learning:
		return s.learningSteps
	case Relearning:
		return s.relearningSteps
	default:
		return nil
	}
}

// transition applies the state machine logic and returns the scheduling interval.
func (s *Scheduler) transition(c *Card, rating Rating, steps []time.Duration) time.Duration {
	switch c.State {
	case Learning, Relearning:
		return s.transitionLearning(c, rating, steps)
	default:
		return s.transitionReview(c, rating)
	}
}

// transitionLearning handles Learning and Relearning state transitions.
func (s *Scheduler) transitionLearning(c *Card, rating Rating, steps []time.Duration) time.Duration {
	step := 0
	if c.Step != nil {
		step = *c.Step
	}

	// Empty steps or step overflow → graduate to Review.
	if len(steps) == 0 || (step >= len(steps) && rating != Again) {
		return s.graduateToReview(c)
	}

	switch rating {
	case Again:
		c.setStep(0)
		return steps[0]

	case Hard:
		if step == 0 && len(steps) == 1 {
			return time.Duration(float64(steps[0]) * 1.5)
		}
		if step == 0 && len(steps) >= 2 {
			return (steps[0] + steps[1]) / 2
		}
		return steps[step]

	case Good:
		nextStep := step + 1
		if nextStep >= len(steps) {
			// Last step → graduate.
			return s.graduateToReview(c)
		}
		c.setStep(nextStep)
		return steps[nextStep]

	default:
		return s.graduateToReview(c)
	}
}

// transitionReview handles Review state transitions.
func (s *Scheduler) transitionReview(c *Card, rating Rating) time.Duration {
	if rating == Again {
		if len(s.relearningSteps) > 0 {
			c.State = Relearning
			c.setStep(0)
			return s.relearningSteps[0]
		}
		// Empty relearning steps → stay Review with nextInterval.
	}

	// Hard, Good, Easy, or Again with empty relearning steps.
	c.clearStep()
	days := s.algo.nextInterval(*c.Stability, s.desiredRetention, s.maximumInterval)
	return time.Duration(days) * 24 * time.Hour
}

// graduateToReview transitions a card from Learning/Relearning to Review.
func (s *Scheduler) graduateToReview(c *Card) time.Duration {
	c.State = Review
	c.clearStep()
	days := s.algo.nextInterval(*c.Stability, s.desiredRetention, s.maximumInterval)
	return time.Duration(days) * 24 * time.Hour
}

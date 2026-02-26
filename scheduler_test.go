package flux

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
	"time"
)

var t0 = time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

func mustScheduler(t *testing.T, cfg SchedulerConfig) *Scheduler {
	t.Helper()
	s, err := NewScheduler(cfg)
	if err != nil {
		t.Fatalf("NewScheduler: %v", err)
	}
	return s
}

func noFuzzCfg() SchedulerConfig {
	return SchedulerConfig{DisableFuzzing: true}
}

// --- NewScheduler ---

func TestNewSchedulerDefault(t *testing.T) {
	s := mustScheduler(t, SchedulerConfig{})
	if s == nil {
		t.Fatal("NewScheduler returned nil")
	}
}

func TestNewSchedulerInvalidParams(t *testing.T) {
	cfg := SchedulerConfig{}
	cfg.Parameters = DefaultParameters
	cfg.Parameters[0] = -1.0 // below lower bound
	_, err := NewScheduler(cfg)
	if err == nil {
		t.Error("NewScheduler should reject invalid parameters")
	}
}

func TestNewSchedulerInvalidRetention(t *testing.T) {
	cfg := SchedulerConfig{DesiredRetention: 1.5}
	_, err := NewScheduler(cfg)
	if err == nil {
		t.Error("NewScheduler should reject retention > 1")
	}
	cfg2 := SchedulerConfig{DesiredRetention: -0.1}
	_, err2 := NewScheduler(cfg2)
	if err2 == nil {
		t.Error("NewScheduler should reject retention < 0")
	}
}

func TestNewSchedulerInvalidMaxInterval(t *testing.T) {
	cfg := SchedulerConfig{MaximumInterval: -1}
	_, err := NewScheduler(cfg)
	if err == nil {
		t.Error("NewScheduler should reject negative max interval")
	}
}

// --- Learning: first review ---

func TestLearningFirstAgain(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := NewCard(1)
	c, _ := s.ReviewCard(card, Again, t0)

	if c.State != Learning {
		t.Errorf("State = %v, want Learning", c.State)
	}
	if c.Step == nil || *c.Step != 0 {
		t.Errorf("Step = %v, want 0", c.Step)
	}
	// S = S₀(Again), D = D₀(Again)
	assertFloat(t, "Stability", *c.Stability, s.algo.initStability(Again))
	assertFloat(t, "Difficulty", *c.Difficulty, s.algo.initDifficulty(Again, true))
	// interval = learning_steps[0] = 1m
	wantDue := t0.Add(time.Minute)
	if !c.Due.Equal(wantDue) {
		t.Errorf("Due = %v, want %v", c.Due, wantDue)
	}
}

func TestLearningFirstHard(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := NewCard(1)
	c, _ := s.ReviewCard(card, Hard, t0)

	if c.State != Learning {
		t.Errorf("State = %v, want Learning", c.State)
	}
	// Hard at step=0, len=2 → interval = (1m + 10m) / 2 = 5.5m
	wantDue := t0.Add((time.Minute + 10*time.Minute) / 2)
	if !c.Due.Equal(wantDue) {
		t.Errorf("Due = %v, want %v", c.Due, wantDue)
	}
}

func TestLearningFirstGood(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := NewCard(1)
	c, _ := s.ReviewCard(card, Good, t0)

	if c.State != Learning {
		t.Errorf("State = %v, want Learning", c.State)
	}
	if c.Step == nil || *c.Step != 1 {
		t.Errorf("Step = %v, want 1", c.Step)
	}
	// interval = learning_steps[1] = 10m
	wantDue := t0.Add(10 * time.Minute)
	if !c.Due.Equal(wantDue) {
		t.Errorf("Due = %v, want %v", c.Due, wantDue)
	}
}

func TestLearningFirstEasy(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := NewCard(1)
	c, _ := s.ReviewCard(card, Easy, t0)

	if c.State != Review {
		t.Errorf("State = %v, want Review", c.State)
	}
	if c.Step != nil {
		t.Errorf("Step = %v, want nil", c.Step)
	}
	// interval = nextInterval(S₀(Easy))
	days := s.algo.nextInterval(*c.Stability, 0.9, 36500)
	wantDue := t0.Add(time.Duration(days) * 24 * time.Hour)
	if !c.Due.Equal(wantDue) {
		t.Errorf("Due = %v, want %v", c.Due, wantDue)
	}
}

// --- Learning: Good at last step → Review ---

func TestLearningGoodLastStep(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := NewCard(1)
	// First Good → step=1 (last step in [1m, 10m])
	c, _ := s.ReviewCard(card, Good, t0)
	// Second Good at step=1 → Review
	c, _ = s.ReviewCard(c, Good, t0.Add(10*time.Minute))

	if c.State != Review {
		t.Errorf("State = %v, want Review", c.State)
	}
	if c.Step != nil {
		t.Errorf("Step = %v, want nil", c.Step)
	}
}

// --- Learning: same-day review → shortTermStability ---

func TestLearningSameDay(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := NewCard(1)
	// First review sets S and D.
	c, _ := s.ReviewCard(card, Again, t0)
	sBefore := *c.Stability
	dBefore := *c.Difficulty

	// Same-day review (5 min later).
	c, _ = s.ReviewCard(c, Good, t0.Add(5*time.Minute))

	// S should be updated via shortTermStability.
	sExpected := s.algo.shortTermStability(sBefore, Good)
	assertFloat(t, "Stability after same-day", *c.Stability, sExpected)
	// D should be updated via nextDifficulty.
	dExpected := s.algo.nextDifficulty(dBefore, Good)
	assertFloat(t, "Difficulty after same-day", *c.Difficulty, dExpected)
}

// --- Learning: cross-day review → nextStability ---

func TestLearningCrossDay(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := NewCard(1)
	c, _ := s.ReviewCard(card, Again, t0)
	sBefore := *c.Stability
	dBefore := *c.Difficulty

	// Cross-day review (2 days later).
	t1 := t0.Add(48 * time.Hour)
	elapsed := t1.Sub(t0).Hours() / 24.0
	r := s.algo.retrievability(elapsed, sBefore)
	c, _ = s.ReviewCard(c, Good, t1)

	sExpected := s.algo.nextStability(dBefore, sBefore, r, Good)
	assertFloat(t, "Stability after cross-day", *c.Stability, sExpected)
}

// --- Learning: Hard step=0 len=1 → 1.5x ---

func TestLearningHardSingleStep(t *testing.T) {
	cfg := noFuzzCfg()
	cfg.LearningSteps = []time.Duration{5 * time.Minute}
	s := mustScheduler(t, cfg)
	card := NewCard(1)
	c, _ := s.ReviewCard(card, Hard, t0)

	// Hard at step=0, len=1 → interval = 5m * 1.5 = 7.5m
	wantDue := t0.Add(time.Duration(float64(5*time.Minute) * 1.5))
	if !c.Due.Equal(wantDue) {
		t.Errorf("Due = %v, want %v", c.Due, wantDue)
	}
}

// --- Learning: Hard step>0 → learning_steps[step] ---

func TestLearningHardMidStep(t *testing.T) {
	cfg := noFuzzCfg()
	cfg.LearningSteps = []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute}
	s := mustScheduler(t, cfg)
	card := NewCard(1)
	card.setStep(1)
	card.setStability(2.0)
	card.setDifficulty(5.0)
	now := t0
	card.LastReview = &now

	c, _ := s.ReviewCard(card, Hard, t0.Add(time.Minute))

	// Hard at step=1, len=3 → interval = learning_steps[1] = 5m
	wantDue := t0.Add(time.Minute).Add(5 * time.Minute)
	if !c.Due.Equal(wantDue) {
		t.Errorf("Due = %v, want %v", c.Due, wantDue)
	}
	if c.Step == nil || *c.Step != 1 {
		t.Errorf("Step = %v, want 1", c.Step)
	}
}

// --- Learning: empty steps → directly Review ---

func TestLearningEmptySteps(t *testing.T) {
	cfg := noFuzzCfg()
	cfg.LearningSteps = []time.Duration{}
	s := mustScheduler(t, cfg)
	card := NewCard(1)
	c, _ := s.ReviewCard(card, Hard, t0)

	if c.State != Review {
		t.Errorf("State = %v, want Review", c.State)
	}
	if c.Step != nil {
		t.Errorf("Step = %v, want nil", c.Step)
	}
}

// --- Learning: step >= len → directly Review ---

func TestLearningStepOverflow(t *testing.T) {
	cfg := noFuzzCfg()
	cfg.LearningSteps = []time.Duration{time.Minute}
	s := mustScheduler(t, cfg)
	card := NewCard(1)
	card.setStep(5) // artificially set step beyond len
	card.setStability(2.0)
	card.setDifficulty(5.0)
	now := t0
	card.LastReview = &now

	c, _ := s.ReviewCard(card, Good, t0.Add(time.Minute))

	if c.State != Review {
		t.Errorf("State = %v, want Review", c.State)
	}
}

// --- Review: cross-day Hard/Good/Easy ---

func reviewCard(t *testing.T) Card {
	t.Helper()
	return Card{
		CardID:     1,
		State:      Review,
		Stability:  ptrF(5.0),
		Difficulty: ptrF(5.0),
		Due:        t0,
		LastReview: ptrT(t0),
	}
}

func ptrF(f float64) *float64     { return &f }
func ptrT(t time.Time) *time.Time { return &t }

func TestReviewCrossDayGood(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := reviewCard(t)
	t1 := t0.Add(5 * 24 * time.Hour) // 5 days later
	c, _ := s.ReviewCard(card, Good, t1)

	if c.State != Review {
		t.Errorf("State = %v, want Review", c.State)
	}
	if c.Step != nil {
		t.Errorf("Step = %v, want nil", c.Step)
	}
	// interval should be > 5 days (stability grew)
	daysDue := c.Due.Sub(t1).Hours() / 24.0
	if daysDue < 5 {
		t.Errorf("interval = %.1f days, want > 5", daysDue)
	}
}

func TestReviewCrossDayHardPenalty(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := reviewCard(t)
	t1 := t0.Add(5 * 24 * time.Hour)
	cGood, _ := s.ReviewCard(card, Good, t1)
	cHard, _ := s.ReviewCard(card, Hard, t1)

	// Hard should give shorter interval than Good.
	ivlGood := cGood.Due.Sub(t1)
	ivlHard := cHard.Due.Sub(t1)
	if ivlHard >= ivlGood {
		t.Errorf("Hard interval %v should be < Good interval %v", ivlHard, ivlGood)
	}
}

func TestReviewCrossDayEasyBonus(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := reviewCard(t)
	t1 := t0.Add(5 * 24 * time.Hour)
	cGood, _ := s.ReviewCard(card, Good, t1)
	cEasy, _ := s.ReviewCard(card, Easy, t1)

	// Easy should give longer interval than Good.
	ivlGood := cGood.Due.Sub(t1)
	ivlEasy := cEasy.Due.Sub(t1)
	if ivlEasy <= ivlGood {
		t.Errorf("Easy interval %v should be > Good interval %v", ivlEasy, ivlGood)
	}
}

// --- Review: same-day → shortTermStability ---

func TestReviewSameDay(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := reviewCard(t)
	// Same-day review (6 hours later).
	t1 := t0.Add(6 * time.Hour)
	c, _ := s.ReviewCard(card, Good, t1)

	// Stability should be updated via shortTermStability, not nextStability.
	sExpected := s.algo.shortTermStability(*card.Stability, Good)
	assertFloat(t, "Stability after same-day Review", *c.Stability, sExpected)
}

// --- Review: Again → Relearning ---

func TestReviewAgainRelearning(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := reviewCard(t)
	t1 := t0.Add(5 * 24 * time.Hour)
	c, _ := s.ReviewCard(card, Again, t1)

	if c.State != Relearning {
		t.Errorf("State = %v, want Relearning", c.State)
	}
	if c.Step == nil || *c.Step != 0 {
		t.Errorf("Step = %v, want 0", c.Step)
	}
	// interval = relearning_steps[0] = 10m
	wantDue := t1.Add(10 * time.Minute)
	if !c.Due.Equal(wantDue) {
		t.Errorf("Due = %v, want %v", c.Due, wantDue)
	}
}

// --- Review: Again + empty relearning_steps → stay Review ---

func TestReviewAgainEmptyRelearningSteps(t *testing.T) {
	cfg := noFuzzCfg()
	cfg.RelearningSteps = []time.Duration{}
	s := mustScheduler(t, cfg)
	card := reviewCard(t)
	t1 := t0.Add(5 * 24 * time.Hour)
	c, _ := s.ReviewCard(card, Again, t1)

	if c.State != Review {
		t.Errorf("State = %v, want Review", c.State)
	}
	// Should have an interval from nextInterval.
	daysDue := c.Due.Sub(t1).Hours() / 24.0
	if daysDue < 0.5 {
		t.Errorf("interval = %.2f days, want >= 0.5", daysDue)
	}
}

// --- Relearning: symmetric with Learning ---

func TestRelearningAgain(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := Card{
		CardID:     1,
		State:      Relearning,
		Step:       ptrI(0),
		Stability:  ptrF(3.0),
		Difficulty: ptrF(5.0),
		Due:        t0,
		LastReview: ptrT(t0),
	}
	c, _ := s.ReviewCard(card, Again, t0.Add(5*time.Minute))

	if c.State != Relearning {
		t.Errorf("State = %v, want Relearning", c.State)
	}
	if c.Step == nil || *c.Step != 0 {
		t.Errorf("Step = %v, want 0", c.Step)
	}
}

func ptrI(i int) *int { return &i }

func TestRelearningGoodGraduate(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	// Default relearning_steps = [10m], so Good at step=0 (last step) → Review.
	card := Card{
		CardID:     1,
		State:      Relearning,
		Step:       ptrI(0),
		Stability:  ptrF(3.0),
		Difficulty: ptrF(5.0),
		Due:        t0,
		LastReview: ptrT(t0),
	}
	c, _ := s.ReviewCard(card, Good, t0.Add(10*time.Minute))

	if c.State != Review {
		t.Errorf("State = %v, want Review", c.State)
	}
	if c.Step != nil {
		t.Errorf("Step = %v, want nil", c.Step)
	}
}

// --- Fuzz ---

func TestFuzzEnabledChangesInterval(t *testing.T) {
	cfg := SchedulerConfig{} // fuzz enabled by default
	s := mustScheduler(t, cfg)
	card := reviewCard(t)
	t1 := t0.Add(10 * 24 * time.Hour)

	// Run multiple times; with fuzz, intervals should vary.
	intervals := make(map[int]bool)
	for i := 0; i < 50; i++ {
		c, _ := s.ReviewCard(card, Good, t1)
		days := int(math.Round(c.Due.Sub(t1).Hours() / 24.0))
		intervals[days] = true
	}
	if len(intervals) < 2 {
		t.Errorf("fuzz should produce varied intervals, got %d unique values", len(intervals))
	}
}

func TestFuzzDisabledStableInterval(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := reviewCard(t)
	t1 := t0.Add(10 * 24 * time.Hour)

	c1, _ := s.ReviewCard(card, Good, t1)
	c2, _ := s.ReviewCard(card, Good, t1)
	if !c1.Due.Equal(c2.Due) {
		t.Errorf("without fuzz, intervals should be identical: %v vs %v", c1.Due, c2.Due)
	}
}

// --- Retrievability ---

func TestRetrievabilityNilLastReview(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := NewCard(1)
	got := s.Retrievability(card, t0)
	if got != 0 {
		t.Errorf("Retrievability with nil LastReview = %f, want 0", got)
	}
}

func TestRetrievabilityNormal(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := reviewCard(t)
	// 5 days later, S=5 → R ≈ 0.9
	t1 := t0.Add(5 * 24 * time.Hour)
	got := s.Retrievability(card, t1)
	assertFloat(t, "Retrievability at S days", got, 0.9)
}

// --- ReviewLog ---

func TestReviewCardReturnsLog(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := NewCard(42)
	_, log := s.ReviewCard(card, Good, t0)

	if log.CardID != 42 {
		t.Errorf("log.CardID = %d, want 42", log.CardID)
	}
	if log.Rating != Good {
		t.Errorf("log.Rating = %v, want Good", log.Rating)
	}
	if !log.ReviewDatetime.Equal(t0) {
		t.Errorf("log.ReviewDatetime = %v, want %v", log.ReviewDatetime, t0)
	}
}

// --- LastReview is set ---

func TestReviewCardSetsLastReview(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := NewCard(1)
	c, _ := s.ReviewCard(card, Good, t0)
	if c.LastReview == nil || !c.LastReview.Equal(t0) {
		t.Errorf("LastReview = %v, want %v", c.LastReview, t0)
	}
}

// --- Input card not mutated ---

func TestReviewCardDoesNotMutateInput(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := NewCard(1)
	original := card
	s.ReviewCard(card, Good, t0)
	if card.State != original.State {
		t.Error("ReviewCard mutated input card State")
	}
	if card.Stability != original.Stability {
		t.Error("ReviewCard mutated input card Stability")
	}
}

// --- PreviewCard ---

func TestPreviewCardReturnsFourRatings(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := NewCard(1)
	previews := s.PreviewCard(card, t0)

	if len(previews) != 4 {
		t.Fatalf("PreviewCard returned %d entries, want 4", len(previews))
	}
	for _, r := range []Rating{Again, Hard, Good, Easy} {
		if _, ok := previews[r]; !ok {
			t.Errorf("missing key %v", r)
		}
	}
}

func TestPreviewCardMatchesReviewCard(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := NewCard(1)
	previews := s.PreviewCard(card, t0)

	for _, r := range []Rating{Again, Hard, Good, Easy} {
		reviewed, _ := s.ReviewCard(card, r, t0)
		preview := previews[r]
		if preview.State != reviewed.State {
			t.Errorf("rating %v: State = %v, want %v", r, preview.State, reviewed.State)
		}
		if !preview.Due.Equal(reviewed.Due) {
			t.Errorf("rating %v: Due = %v, want %v", r, preview.Due, reviewed.Due)
		}
		if (preview.Stability == nil) != (reviewed.Stability == nil) {
			t.Errorf("rating %v: Stability nil mismatch", r)
		} else if preview.Stability != nil {
			assertFloat(t, "Stability", *preview.Stability, *reviewed.Stability)
		}
	}
}

func TestPreviewCardDoesNotMutateInput(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := reviewCard(t)
	original := card
	s.PreviewCard(card, t0)
	if card.State != original.State {
		t.Error("PreviewCard mutated input card State")
	}
}

// --- RescheduleCard ---

func TestRescheduleCardReplay(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := NewCard(1)

	// Manually build a 3-step sequence.
	c1, log1 := s.ReviewCard(card, Good, t0)
	t1 := t0.Add(10 * time.Minute)
	c2, log2 := s.ReviewCard(c1, Good, t1)
	t2 := t1.Add(5 * 24 * time.Hour)
	c3, log3 := s.ReviewCard(c2, Good, t2)

	// RescheduleCard should reproduce the same final state.
	got, err := s.RescheduleCard(NewCard(1), []ReviewLog{log1, log2, log3})
	if err != nil {
		t.Fatalf("RescheduleCard: %v", err)
	}
	if got.State != c3.State {
		t.Errorf("State = %v, want %v", got.State, c3.State)
	}
	assertFloat(t, "Stability", *got.Stability, *c3.Stability)
	assertFloat(t, "Difficulty", *got.Difficulty, *c3.Difficulty)
}

func TestRescheduleCardIDMismatch(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := NewCard(1)
	logs := []ReviewLog{
		{CardID: 999, Rating: Good, ReviewDatetime: t0},
	}
	_, err := s.RescheduleCard(card, logs)
	if err == nil {
		t.Error("RescheduleCard should return error for CardID mismatch")
	}
	if !errors.Is(err, ErrCardIDMismatch) {
		t.Errorf("error = %v, want ErrCardIDMismatch", err)
	}
}

func TestRescheduleCardEmptyLogs(t *testing.T) {
	s := mustScheduler(t, noFuzzCfg())
	card := NewCard(1)
	got, err := s.RescheduleCard(card, nil)
	if err != nil {
		t.Fatalf("RescheduleCard: %v", err)
	}
	// No logs → card returned as-is.
	if got.State != card.State {
		t.Errorf("State = %v, want %v", got.State, card.State)
	}
}

// --- Scheduler JSON ---

func TestSchedulerJSONRoundTrip(t *testing.T) {
	cfg := SchedulerConfig{
		DesiredRetention: 0.85,
		MaximumInterval:  180,
		DisableFuzzing:   true,
		LearningSteps:    []time.Duration{2 * time.Minute, 15 * time.Minute},
		RelearningSteps:  []time.Duration{5 * time.Minute},
	}
	s := mustScheduler(t, cfg)

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var s2 Scheduler
	if err := json.Unmarshal(data, &s2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Verify round-trip produces identical scheduling results.
	card := NewCard(1)
	c1, _ := s.ReviewCard(card, Good, t0)
	c2, _ := s2.ReviewCard(card, Good, t0)

	if c1.State != c2.State {
		t.Errorf("State mismatch: %v vs %v", c1.State, c2.State)
	}
	if !c1.Due.Equal(c2.Due) {
		t.Errorf("Due mismatch: %v vs %v", c1.Due, c2.Due)
	}
	assertFloat(t, "Stability", *c1.Stability, *c2.Stability)
	assertFloat(t, "Difficulty", *c1.Difficulty, *c2.Difficulty)
}

func TestSchedulerJSONDefaultConfig(t *testing.T) {
	s := mustScheduler(t, SchedulerConfig{})
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var s2 Scheduler
	if err := json.Unmarshal(data, &s2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Should produce same results with default config.
	card := reviewCard(t)
	t1 := t0.Add(5 * 24 * time.Hour)
	c1, _ := s.ReviewCard(card, Hard, t1)
	c2, _ := s2.ReviewCard(card, Hard, t1)
	assertFloat(t, "Stability", *c1.Stability, *c2.Stability)
}

func TestSchedulerJSONMalformed(t *testing.T) {
	var s Scheduler
	// Valid JSON but wrong structure for schedulerJSON.
	if err := json.Unmarshal([]byte(`{"parameters":"not_an_array"}`), &s); err == nil {
		t.Error("Unmarshal should reject malformed scheduler JSON")
	}
}

func TestSchedulerJSONEmptySteps(t *testing.T) {
	// Empty steps (not nil) should survive round-trip.
	cfg := SchedulerConfig{
		LearningSteps:   []time.Duration{},
		RelearningSteps: []time.Duration{},
		DisableFuzzing:  true,
	}
	s := mustScheduler(t, cfg)

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var s2 Scheduler
	if err := json.Unmarshal(data, &s2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Empty steps → any rating graduates immediately to Review.
	card := NewCard(1)
	c, _ := s2.ReviewCard(card, Hard, t0)
	if c.State != Review {
		t.Errorf("State = %v, want Review (empty steps)", c.State)
	}
}

func TestSchedulerJSONInvalidParams(t *testing.T) {
	// Craft JSON with invalid parameters to trigger NewScheduler error.
	bad := `{"parameters":[999,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"desired_retention":0.9,"learning_steps":null,"relearning_steps":null,"maximum_interval":36500,"disable_fuzzing":false}`
	var s Scheduler
	if err := json.Unmarshal([]byte(bad), &s); err == nil {
		t.Error("Unmarshal should reject invalid parameters")
	}
}

func TestSchedulerJSONNullSteps(t *testing.T) {
	// JSON with null steps → NewScheduler fills defaults.
	raw := `{"parameters":[0.212,1.2931,2.3065,8.2956,6.4133,0.8334,3.0194,0.001,1.8722,0.1666,0.796,1.4835,0.0614,0.2629,1.6483,0.6014,1.8729,0.5425,0.0912,0.0658,0.1542],"desired_retention":0.9,"learning_steps":null,"relearning_steps":null,"maximum_interval":36500,"disable_fuzzing":true}`
	var s Scheduler
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Default steps should be restored from null.
	card := NewCard(1)
	c, _ := s.ReviewCard(card, Good, t0)
	if c.State != Learning {
		t.Errorf("State = %v, want Learning (default steps from null)", c.State)
	}
}

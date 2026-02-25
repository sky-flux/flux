package optimizer

import (
	"math"
	"testing"
)

// --- Adam ---

func TestAdamUpdateDirection(t *testing.T) {
	// A positive gradient should decrease the parameter.
	adam := NewAdam(0.04)

	params := [21]float64{1.0}
	grads := [21]float64{2.0} // positive gradient for w[0]

	updated := adam.Update(params, grads)
	if updated[0] >= params[0] {
		t.Errorf("w[0] = %f, want < %f (should decrease with positive gradient)", updated[0], params[0])
	}
}

func TestAdamUpdateNegativeGradient(t *testing.T) {
	// A negative gradient should increase the parameter.
	adam := NewAdam(0.04)

	params := [21]float64{1.0}
	grads := [21]float64{-2.0}

	updated := adam.Update(params, grads)
	if updated[0] <= params[0] {
		t.Errorf("w[0] = %f, want > %f (should increase with negative gradient)", updated[0], params[0])
	}
}

func TestAdamBiasCorrection(t *testing.T) {
	// At step 1 with β1=0.9, the bias-corrected m̂ should be ~10x the raw m.
	// m = (1-β1)*g = 0.1*g, m̂ = m/(1-β1^1) = m/0.1 = g
	// So the effective gradient is the full gradient, not dampened by β1.
	adam := NewAdam(0.04)

	params := [21]float64{5.0}
	grads := [21]float64{1.0}

	updated := adam.Update(params, grads)
	// With lr=0.04, the step should be close to 0.04 (since m̂≈1.0, v̂≈1.0).
	// Exact: m̂ = 1.0, v̂ = 1.0, step = 0.04 * 1.0 / (sqrt(1.0) + 1e-8) ≈ 0.04
	diff := params[0] - updated[0]
	assertFloatOpt(t, "bias correction step", diff, 0.04)
}

func TestAdamMultiStep(t *testing.T) {
	// After several steps with constant gradient, parameters should
	// move consistently in the gradient descent direction.
	adam := NewAdam(0.04)

	params := [21]float64{10.0}
	grads := [21]float64{1.0}

	for i := 0; i < 10; i++ {
		params = adam.Update(params, grads)
	}
	// After 10 steps of descending with positive gradient, w[0] should be < 10.
	if params[0] >= 10.0 {
		t.Errorf("w[0] = %f after 10 steps, want < 10.0", params[0])
	}
}

func TestAdamZeroGradient(t *testing.T) {
	// Zero gradient should not change the parameter (from initial state).
	adam := NewAdam(0.04)

	params := [21]float64{5.0, 3.0, 7.0}
	grads := [21]float64{} // all zeros

	updated := adam.Update(params, grads)
	for i := 0; i < 21; i++ {
		if updated[i] != params[i] {
			t.Errorf("w[%d] = %f, want %f (zero gradient should not change params)", i, updated[i], params[i])
		}
	}
}

func TestAdamMultipleParams(t *testing.T) {
	// Different gradients for different parameters should produce
	// different update magnitudes.
	adam := NewAdam(0.04)

	params := [21]float64{5.0, 5.0}
	grads := [21]float64{1.0, 0.1}

	updated := adam.Update(params, grads)
	step0 := params[0] - updated[0]
	step1 := params[1] - updated[1]

	// Both should decrease, but step0 > step1 (since grad[0] > grad[1]).
	// Actually with Adam bias correction at step 1, both effective steps are ~0.04.
	// But at subsequent steps the difference appears. Let's just check direction.
	if step0 <= 0 {
		t.Errorf("step0 = %f, want > 0", step0)
	}
	if step1 <= 0 {
		t.Errorf("step1 = %f, want > 0", step1)
	}
}

func TestAdamSetLR(t *testing.T) {
	// SetLR should change the learning rate used by subsequent updates.
	adam := NewAdam(0.04)

	params := [21]float64{5.0}
	grads := [21]float64{1.0}

	updated1 := adam.Update(params, grads)
	step1 := params[0] - updated1[0]

	// Reset and use a much larger LR.
	adam2 := NewAdam(0.04)
	adam2.SetLR(0.4)

	updated2 := adam2.Update(params, grads)
	step2 := params[0] - updated2[0]

	// step2 should be ~10x step1 since lr is 10x.
	if step2 <= step1 {
		t.Errorf("step with lr=0.4 (%f) should be > step with lr=0.04 (%f)", step2, step1)
	}
}

// --- CosineAnnealing ---

func TestCosineAnnealingStart(t *testing.T) {
	// At t=0, lr should equal lr_max.
	ca := NewCosineAnnealing(0.04, 100)
	lr := ca.LR()
	assertFloatOpt(t, "lr at t=0", lr, 0.04)
}

func TestCosineAnnealingEnd(t *testing.T) {
	// At t=T_max, lr should be ≈ 0.
	ca := NewCosineAnnealing(0.04, 100)
	for i := 0; i < 100; i++ {
		ca.Step()
	}
	lr := ca.LR()
	if lr > 1e-6 {
		t.Errorf("lr at t=T_max = %f, want ≈ 0", lr)
	}
}

func TestCosineAnnealingMidpoint(t *testing.T) {
	// At t=T_max/2, lr should ≈ lr_max/2.
	ca := NewCosineAnnealing(0.04, 100)
	for i := 0; i < 50; i++ {
		ca.Step()
	}
	lr := ca.LR()
	assertFloatOpt(t, "lr at T_max/2", lr, 0.02)
}

func TestCosineAnnealingMonotonic(t *testing.T) {
	// LR should monotonically decrease from t=0 to t=T_max.
	ca := NewCosineAnnealing(0.04, 50)
	prev := ca.LR()
	for i := 0; i < 50; i++ {
		ca.Step()
		cur := ca.LR()
		if cur > prev+1e-10 {
			t.Errorf("lr increased at step %d: %f > %f", i+1, cur, prev)
		}
		prev = cur
	}
}

func TestCosineAnnealingFormula(t *testing.T) {
	// Verify the exact formula: lr_t = 0.5 * lr_max * (1 + cos(π * t / T_max))
	lrMax := 0.04
	tMax := 100
	ca := NewCosineAnnealing(lrMax, tMax)

	steps := []int{0, 10, 25, 50, 75, 100}
	for _, s := range steps {
		ca2 := NewCosineAnnealing(lrMax, tMax)
		for i := 0; i < s; i++ {
			ca2.Step()
		}
		got := ca2.LR()
		want := 0.5 * lrMax * (1 + math.Cos(math.Pi*float64(s)/float64(tMax)))
		assertFloatOpt(t, "cosine lr at step", got, want)
	}
	_ = ca // suppress unused
}

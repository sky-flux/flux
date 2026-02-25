package flux

import (
	"math"
	"testing"
)

const epsilon = 1e-4

func assertFloat(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > epsilon {
		t.Errorf("%s = %.6f, want %.6f (diff %.6f)", name, got, want, math.Abs(got-want))
	}
}

func TestNewAlgo(t *testing.T) {
	a := newAlgo(DefaultParameters)
	// DECAY = -w[20] = -0.1542
	assertFloat(t, "decay", a.decay, -0.1542)
	// FACTOR = 0.9^(1/DECAY) - 1
	wantFactor := math.Pow(0.9, 1.0/a.decay) - 1.0
	assertFloat(t, "factor", a.factor, wantFactor)
}

// --- retrievability ---

func TestRetrievabilityAtZero(t *testing.T) {
	a := newAlgo(DefaultParameters)
	// R(0, S) = (1 + FACTOR * 0 / S) ^ DECAY = 1.0
	got := a.retrievability(0, 5.0)
	assertFloat(t, "R(0, 5)", got, 1.0)
}

func TestRetrievabilityAtStability(t *testing.T) {
	a := newAlgo(DefaultParameters)
	// R(S, S) should be 0.9 by definition of stability.
	got := a.retrievability(5.0, 5.0)
	assertFloat(t, "R(S, S)", got, 0.9)
}

func TestRetrievabilityDecay(t *testing.T) {
	a := newAlgo(DefaultParameters)
	// R(t, S) decreases as t increases.
	r1 := a.retrievability(1.0, 5.0)
	r2 := a.retrievability(10.0, 5.0)
	if r1 <= r2 {
		t.Errorf("R(1, 5) = %.4f should be > R(10, 5) = %.4f", r1, r2)
	}
}

func TestRetrievabilitySmallS(t *testing.T) {
	a := newAlgo(DefaultParameters)
	// With minimal S, R drops fast.
	got := a.retrievability(1.0, 0.001)
	if got >= 0.5 {
		t.Errorf("R(1, 0.001) = %.4f, expected < 0.5", got)
	}
}

// --- initStability ---

func TestInitStability(t *testing.T) {
	a := newAlgo(DefaultParameters)
	// S₀(G) = clamp_s(w[G-1])
	tests := []struct {
		r    Rating
		want float64
	}{
		{Again, DefaultParameters[0]}, // 0.212
		{Hard, DefaultParameters[1]},  // 1.2931
		{Good, DefaultParameters[2]},  // 2.3065
		{Easy, DefaultParameters[3]},  // 8.2956
	}
	for _, tt := range tests {
		got := a.initStability(tt.r)
		want := math.Max(tt.want, 0.001)
		assertFloat(t, "S0("+tt.r.String()+")", got, want)
	}
}

// --- initDifficulty ---

func TestInitDifficulty(t *testing.T) {
	a := newAlgo(DefaultParameters)
	// D₀(G) = w[4] - e^(w[5]*(G-1)) + 1, clamped to [1, 10]
	tests := []struct {
		r     Rating
		clamp bool
	}{
		{Again, true},
		{Hard, true},
		{Good, true},
		{Easy, true},
	}
	for _, tt := range tests {
		got := a.initDifficulty(tt.r, tt.clamp)
		// D₀(G) = w[4] - exp(w[5] * (G - 1)) + 1
		raw := DefaultParameters[4] - math.Exp(DefaultParameters[5]*float64(tt.r-1)) + 1
		want := raw
		if tt.clamp {
			want = math.Min(math.Max(want, 1), 10)
		}
		assertFloat(t, "D0("+tt.r.String()+")", got, want)
	}
}

func TestInitDifficultyNoClamp(t *testing.T) {
	a := newAlgo(DefaultParameters)
	// When clamp=false, result can be outside [1, 10].
	// Used for mean reversion target.
	got := a.initDifficulty(Easy, false)
	raw := DefaultParameters[4] - math.Exp(DefaultParameters[5]*float64(Easy-1)) + 1
	assertFloat(t, "D0(Easy, no clamp)", got, raw)
}

// --- nextInterval ---

func TestNextInterval(t *testing.T) {
	a := newAlgo(DefaultParameters)
	// I(r, S) = round((S / FACTOR) * (r^(1/DECAY) - 1)), clamped to [1, maxIvl]
	got := a.nextInterval(5.0, 0.9, 36500)
	// When r=0.9 and S=5: interval should be 5 (since R(S,S)=0.9 by definition).
	if got != 5 {
		t.Errorf("nextInterval(5.0, 0.9, 36500) = %d, want 5", got)
	}
}

func TestNextIntervalClampMin(t *testing.T) {
	a := newAlgo(DefaultParameters)
	// Very small S → interval clamps to 1.
	got := a.nextInterval(0.001, 0.9, 36500)
	if got < 1 {
		t.Errorf("nextInterval should be >= 1, got %d", got)
	}
}

func TestNextIntervalClampMax(t *testing.T) {
	a := newAlgo(DefaultParameters)
	// Very large S → clamp to maxIvl.
	got := a.nextInterval(100000.0, 0.9, 365)
	if got != 365 {
		t.Errorf("nextInterval should clamp to maxIvl 365, got %d", got)
	}
}

func TestNextIntervalLowRetention(t *testing.T) {
	a := newAlgo(DefaultParameters)
	// Lower retention → longer interval.
	ivl90 := a.nextInterval(10.0, 0.9, 36500)
	ivl80 := a.nextInterval(10.0, 0.8, 36500)
	if ivl80 <= ivl90 {
		t.Errorf("lower retention should give longer interval: ivl80=%d, ivl90=%d", ivl80, ivl90)
	}
}

// --- shortTermStability ---

func TestShortTermStability(t *testing.T) {
	a := newAlgo(DefaultParameters)
	// SInc = exp(w[17] * (G - 3 + w[18])) * S^(-w[19])
	// For Good (G=3): SInc = exp(w[17] * w[18]) * S^(-w[19])
	// If G ∈ {Good, Easy}: SInc = max(SInc, 1.0)
	// S' = clamp_s(S * SInc)

	tests := []struct {
		name string
		s    float64
		r    Rating
	}{
		{"Again S=5", 5.0, Again},
		{"Hard S=5", 5.0, Hard},
		{"Good S=5", 5.0, Good},
		{"Easy S=5", 5.0, Easy},
	}
	for _, tt := range tests {
		got := a.shortTermStability(tt.s, tt.r)

		sInc := math.Exp(DefaultParameters[17]*(float64(tt.r)-3+DefaultParameters[18])) * math.Pow(tt.s, -DefaultParameters[19])
		if tt.r == Good || tt.r == Easy {
			sInc = math.Max(sInc, 1.0)
		}
		want := math.Max(tt.s*sInc, 0.001)

		assertFloat(t, tt.name, got, want)
	}
}

func TestShortTermStabilityGoodNoDecrease(t *testing.T) {
	a := newAlgo(DefaultParameters)
	// For Good/Easy, SInc ≥ 1.0 → stability never decreases.
	s := 5.0
	got := a.shortTermStability(s, Good)
	if got < s {
		t.Errorf("Good shortTerm should not decrease: got %.4f < %.4f", got, s)
	}
}

// --- nextDifficulty ---

func TestNextDifficulty(t *testing.T) {
	a := newAlgo(DefaultParameters)

	tests := []struct {
		name string
		d    float64
		r    Rating
	}{
		{"Again D=5", 5.0, Again},
		{"Good D=5", 5.0, Good},
		{"Easy D=5", 5.0, Easy},
		{"Again D=1 boundary", 1.0, Again},
		{"Easy D=10 boundary", 10.0, Easy},
	}
	for _, tt := range tests {
		got := a.nextDifficulty(tt.d, tt.r)

		deltaD := -DefaultParameters[6] * (float64(tt.r) - 3)
		dPrime := tt.d + (10-tt.d)*deltaD/9
		d0Easy := DefaultParameters[4] - math.Exp(DefaultParameters[5]*float64(Easy-1)) + 1
		dDoublePrime := DefaultParameters[7]*d0Easy + (1-DefaultParameters[7])*dPrime
		want := math.Min(math.Max(dDoublePrime, 1), 10)

		assertFloat(t, tt.name, got, want)
	}
}

func TestNextDifficultyAgainIncreases(t *testing.T) {
	a := newAlgo(DefaultParameters)
	d := 5.0
	got := a.nextDifficulty(d, Again)
	if got <= d {
		t.Errorf("Again should increase difficulty: got %.4f <= %.4f", got, d)
	}
}

func TestNextDifficultyEasyDecreases(t *testing.T) {
	a := newAlgo(DefaultParameters)
	d := 5.0
	got := a.nextDifficulty(d, Easy)
	if got >= d {
		t.Errorf("Easy should decrease difficulty: got %.4f >= %.4f", got, d)
	}
}

// --- nextRecallStability ---

func TestNextRecallStability(t *testing.T) {
	a := newAlgo(DefaultParameters)

	tests := []struct {
		name string
		d    float64
		s    float64
		r    float64
		g    Rating
	}{
		{"Good D=5 S=5 R=0.9", 5.0, 5.0, 0.9, Good},
		{"Hard D=5 S=5 R=0.9", 5.0, 5.0, 0.9, Hard},
		{"Easy D=5 S=5 R=0.9", 5.0, 5.0, 0.9, Easy},
		{"Good D=5 S=5 R=0.5", 5.0, 5.0, 0.5, Good},
		{"Good D=1 S=1 R=0.9", 1.0, 1.0, 0.9, Good},
	}
	for _, tt := range tests {
		got := a.nextRecallStability(tt.d, tt.s, tt.r, tt.g)

		hardPenalty := 1.0
		if tt.g == Hard {
			hardPenalty = DefaultParameters[15]
		}
		easyBonus := 1.0
		if tt.g == Easy {
			easyBonus = DefaultParameters[16]
		}
		want := tt.s * (1 + math.Exp(DefaultParameters[8])*
			(11-tt.d)*
			math.Pow(tt.s, -DefaultParameters[9])*
			(math.Exp((1-tt.r)*DefaultParameters[10])-1)*
			hardPenalty*easyBonus)

		assertFloat(t, tt.name, got, want)
	}
}

func TestNextRecallStabilityGrowth(t *testing.T) {
	a := newAlgo(DefaultParameters)
	// Recall stability should always increase for Good/Easy.
	s := 5.0
	got := a.nextRecallStability(5.0, s, 0.9, Good)
	if got <= s {
		t.Errorf("recall stability should grow: got %.4f <= %.4f", got, s)
	}
}

// --- nextForgetStability ---

func TestNextForgetStability(t *testing.T) {
	a := newAlgo(DefaultParameters)

	tests := []struct {
		name string
		d    float64
		s    float64
		r    float64
	}{
		{"D=5 S=5 R=0.9", 5.0, 5.0, 0.9},
		{"D=5 S=5 R=0.5", 5.0, 5.0, 0.5},
		{"D=1 S=1 R=0.9", 1.0, 1.0, 0.9},
		{"D=10 S=50 R=0.9", 10.0, 50.0, 0.9},
	}
	for _, tt := range tests {
		got := a.nextForgetStability(tt.d, tt.s, tt.r)

		long := DefaultParameters[11] *
			math.Pow(tt.d, -DefaultParameters[12]) *
			(math.Pow(tt.s+1, DefaultParameters[13]) - 1) *
			math.Exp((1-tt.r)*DefaultParameters[14])
		short := tt.s / math.Exp(DefaultParameters[17]*DefaultParameters[18])
		want := math.Min(long, short)

		assertFloat(t, tt.name, got, want)
	}
}

func TestNextForgetStabilityLessThanS(t *testing.T) {
	a := newAlgo(DefaultParameters)
	// Forget stability should be less than current stability.
	s := 5.0
	got := a.nextForgetStability(5.0, s, 0.9)
	if got >= s {
		t.Errorf("forget stability should be < S: got %.4f >= %.4f", got, s)
	}
}

// --- nextStability (dispatch) ---

func TestNextStability(t *testing.T) {
	a := newAlgo(DefaultParameters)
	d, s, r := 5.0, 5.0, 0.9

	// Again → nextForgetStability
	gotAgain := a.nextStability(d, s, r, Again)
	wantAgain := a.nextForgetStability(d, s, r)
	assertFloat(t, "nextStability Again", gotAgain, wantAgain)

	// Hard → nextRecallStability
	gotHard := a.nextStability(d, s, r, Hard)
	wantHard := a.nextRecallStability(d, s, r, Hard)
	assertFloat(t, "nextStability Hard", gotHard, wantHard)

	// Good → nextRecallStability
	gotGood := a.nextStability(d, s, r, Good)
	wantGood := a.nextRecallStability(d, s, r, Good)
	assertFloat(t, "nextStability Good", gotGood, wantGood)

	// Easy → nextRecallStability
	gotEasy := a.nextStability(d, s, r, Easy)
	wantEasy := a.nextRecallStability(d, s, r, Easy)
	assertFloat(t, "nextStability Easy", gotEasy, wantEasy)
}

// --- clamp helpers ---

func TestClampS(t *testing.T) {
	tests := []struct {
		in, want float64
	}{
		{5.0, 5.0},
		{0.001, 0.001},
		{0.0, 0.001},
		{-1.0, 0.001},
	}
	for _, tt := range tests {
		got := clampS(tt.in)
		assertFloat(t, "clampS", got, tt.want)
	}
}

func TestClampD(t *testing.T) {
	tests := []struct {
		in, want float64
	}{
		{5.0, 5.0},
		{1.0, 1.0},
		{10.0, 10.0},
		{0.5, 1.0},
		{11.0, 10.0},
	}
	for _, tt := range tests {
		got := clampD(tt.in)
		assertFloat(t, "clampD", got, tt.want)
	}
}

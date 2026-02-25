package flux

import "math"

// algo holds precomputed constants derived from the 21 FSRS parameters.
type algo struct {
	w      [21]float64
	decay  float64 // -w[20]
	factor float64 // 0.9^(1/decay) - 1
}

// newAlgo creates an algo with precomputed decay and factor.
func newAlgo(p [21]float64) algo {
	decay := -p[20]
	factor := math.Pow(0.9, 1.0/decay) - 1.0
	return algo{w: p, decay: decay, factor: factor}
}

// retrievability computes R(t, S) = (1 + FACTOR * t / S) ^ DECAY.
func (a *algo) retrievability(elapsedDays, stability float64) float64 {
	return math.Pow(1+a.factor*elapsedDays/stability, a.decay)
}

// initStability returns the initial stability S₀(G) = clamp_s(w[G-1]).
func (a *algo) initStability(r Rating) float64 {
	return clampS(a.w[r-1])
}

// initDifficulty returns the initial difficulty D₀(G).
// D₀(G) = w[4] - e^(w[5] * (G - 1)) + 1
// When clamp is true, the result is clamped to [1, 10].
func (a *algo) initDifficulty(r Rating, clamp bool) float64 {
	d := a.w[4] - math.Exp(a.w[5]*float64(r-1)) + 1
	if clamp {
		return clampD(d)
	}
	return d
}

// nextInterval computes the next review interval in days.
// I(r, S) = round((S / FACTOR) * (r^(1/DECAY) - 1)), clamped to [1, maxIvl].
func (a *algo) nextInterval(stability, desiredRetention float64, maxIvl int) int {
	ivl := stability / a.factor * (math.Pow(desiredRetention, 1.0/a.decay) - 1)
	rounded := int(math.Round(ivl))
	if rounded < 1 {
		rounded = 1
	}
	if rounded > maxIvl {
		rounded = maxIvl
	}
	return rounded
}

// shortTermStability computes the same-day review stability.
// SInc = e^(w[17] * (G - 3 + w[18])) * S^(-w[19])
// If G ∈ {Good, Easy}: SInc = max(SInc, 1.0)
// S' = clamp_s(S * SInc)
func (a *algo) shortTermStability(stability float64, r Rating) float64 {
	sInc := math.Exp(a.w[17]*(float64(r)-3+a.w[18])) * math.Pow(stability, -a.w[19])
	if r == Good || r == Easy {
		sInc = math.Max(sInc, 1.0)
	}
	return clampS(stability * sInc)
}

// nextDifficulty computes the updated difficulty after a review.
// ΔD = -w[6] * (G - 3)
// D' = D + (10 - D) * ΔD / 9     (linear damping)
// D'' = w[7]*D₀(Easy) + (1-w[7])*D'  (mean reversion)
// D'' = clamp_d(D'')
func (a *algo) nextDifficulty(difficulty float64, r Rating) float64 {
	deltaD := -a.w[6] * (float64(r) - 3)
	dPrime := difficulty + (10-difficulty)*deltaD/9
	d0Easy := a.initDifficulty(Easy, false) // mean reversion target, unclamped
	dDoublePrime := a.w[7]*d0Easy + (1-a.w[7])*dPrime
	return clampD(dDoublePrime)
}

// nextStability dispatches to nextRecallStability or nextForgetStability.
func (a *algo) nextStability(d, s, r float64, rating Rating) float64 {
	if rating == Again {
		return a.nextForgetStability(d, s, r)
	}
	return a.nextRecallStability(d, s, r, rating)
}

// nextRecallStability computes stability after a successful recall (Hard/Good/Easy).
// S'_r = S * (1 + e^w[8] * (11-D) * S^(-w[9]) * (e^((1-R)*w[10]) - 1) * hardPenalty * easyBonus)
func (a *algo) nextRecallStability(d, s, r float64, rating Rating) float64 {
	hardPenalty := 1.0
	if rating == Hard {
		hardPenalty = a.w[15]
	}
	easyBonus := 1.0
	if rating == Easy {
		easyBonus = a.w[16]
	}
	return s * (1 + math.Exp(a.w[8])*
		(11-d)*
		math.Pow(s, -a.w[9])*
		(math.Exp((1-r)*a.w[10])-1)*
		hardPenalty*easyBonus)
}

// nextForgetStability computes stability after forgetting (Again).
// S'_f = min(long, short)
// long = w[11] * D^(-w[12]) * ((S+1)^w[13] - 1) * e^((1-R)*w[14])
// short = S / e^(w[17] * w[18])
func (a *algo) nextForgetStability(d, s, r float64) float64 {
	long := a.w[11] *
		math.Pow(d, -a.w[12]) *
		(math.Pow(s+1, a.w[13]) - 1) *
		math.Exp((1-r)*a.w[14])
	short := s / math.Exp(a.w[17]*a.w[18])
	return math.Min(long, short)
}

// clampS clamps stability to a minimum of 0.001.
func clampS(s float64) float64 {
	return math.Max(s, 0.001)
}

// clampD clamps difficulty to [1, 10].
func clampD(d float64) float64 {
	return math.Min(math.Max(d, 1), 10)
}

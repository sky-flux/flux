package optimizer

import "math"

// Adam implements the Adam optimizer with bias correction.
//
// Update rule:
//
//	m[i] = β1·m[i] + (1-β1)·g[i]
//	v[i] = β2·v[i] + (1-β2)·g[i]²
//	m̂[i] = m[i] / (1 - β1^t)
//	v̂[i] = v[i] / (1 - β2^t)
//	w[i] = w[i] - lr · m̂[i] / (√v̂[i] + ε)
type Adam struct {
	lr   float64
	beta1, beta2 float64
	eps  float64
	m, v [21]float64
	step int
}

// NewAdam creates an Adam optimizer with the given learning rate.
// Uses standard defaults: β1=0.9, β2=0.999, ε=1e-8.
func NewAdam(lr float64) *Adam {
	return &Adam{
		lr:    lr,
		beta1: 0.9,
		beta2: 0.999,
		eps:   1e-8,
	}
}

// Update applies one Adam step and returns the updated parameters.
func (a *Adam) Update(params, grads [21]float64) [21]float64 {
	a.step++

	for i := 0; i < 21; i++ {
		g := grads[i]
		if g == 0 {
			continue
		}

		a.m[i] = a.beta1*a.m[i] + (1-a.beta1)*g
		a.v[i] = a.beta2*a.v[i] + (1-a.beta2)*g*g

		mHat := a.m[i] / (1 - math.Pow(a.beta1, float64(a.step)))
		vHat := a.v[i] / (1 - math.Pow(a.beta2, float64(a.step)))

		params[i] -= a.lr * mHat / (math.Sqrt(vHat) + a.eps)
	}

	return params
}

// SetLR updates the learning rate (used by CosineAnnealing).
func (a *Adam) SetLR(lr float64) {
	a.lr = lr
}

// CosineAnnealing implements the cosine annealing learning rate schedule.
//
//	lr_t = 0.5 * lr_max * (1 + cos(π * t / T_max))
type CosineAnnealing struct {
	lrMax float64
	tMax  int
	t     int
}

// NewCosineAnnealing creates a cosine annealing scheduler.
func NewCosineAnnealing(lrMax float64, tMax int) *CosineAnnealing {
	return &CosineAnnealing{
		lrMax: lrMax,
		tMax:  tMax,
	}
}

// LR returns the current learning rate.
func (ca *CosineAnnealing) LR() float64 {
	return 0.5 * ca.lrMax * (1 + math.Cos(math.Pi*float64(ca.t)/float64(ca.tMax)))
}

// Step advances the schedule by one step and returns the new learning rate.
func (ca *CosineAnnealing) Step() float64 {
	ca.t++
	return ca.LR()
}

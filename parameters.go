package flux

import "fmt"

// DefaultParameters are the FSRS v6 default parameter values
// from py-fsrs / fsrs4anki Wiki FSRS-6.
var DefaultParameters = [21]float64{
	0.212, 1.2931, 2.3065, 8.2956, // w[0..3]  initial stability S₀(G)
	6.4133, 0.8334, 3.0194, 0.001, // w[4..7]  difficulty params
	1.8722, 0.1666, 0.796, 1.4835, // w[8..11] recall stability params
	0.0614, 0.2629, 1.6483, 0.6014, // w[12..15] forget stability params
	1.8729, 0.5425, 0.0912, 0.0658, // w[16..19] easy/short-term params
	0.1542, // w[20] decay exponent (v6 trainable)
}

// LowerBounds defines the minimum allowed value for each parameter.
var LowerBounds = [21]float64{
	0.001, 0.001, 0.001, 0.001,
	1.0, 0.001, 0.001, 0.001,
	0.0, 0.0, 0.001, 0.001,
	0.001, 0.001, 0.0, 0.0,
	1.0, 0.0, 0.0, 0.0,
	0.1,
}

// UpperBounds defines the maximum allowed value for each parameter.
var UpperBounds = [21]float64{
	100.0, 100.0, 100.0, 100.0,
	10.0, 4.0, 4.0, 0.75,
	4.5, 0.8, 3.5, 5.0,
	0.25, 0.9, 4.0, 1.0,
	6.0, 2.0, 2.0, 0.8,
	0.8,
}

// ValidateParameters checks that all 21 parameters are within [LowerBounds, UpperBounds].
func ValidateParameters(p [21]float64) error {
	for i := 0; i < 21; i++ {
		if p[i] < LowerBounds[i] || p[i] > UpperBounds[i] {
			return fmt.Errorf("%w: w[%d] = %f, bounds [%f, %f]",
				ErrInvalidParameters, i, p[i], LowerBounds[i], UpperBounds[i])
		}
	}
	return nil
}

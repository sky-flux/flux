package flux

import "errors"

// Sentinel errors for the flux package.
// Use errors.Is to check: errors.Is(err, flux.ErrInvalidRating)
var (
	ErrInvalidRating     = errors.New("flux: invalid rating")
	ErrInvalidParameters = errors.New("flux: parameters out of bounds")
	ErrCardIDMismatch    = errors.New("flux: card ID mismatch in review log")
	ErrInsufficientData  = errors.New("flux: insufficient review data for optimization")
)

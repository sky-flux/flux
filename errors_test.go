package flux

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	sentinels := []error{
		ErrInvalidRating,
		ErrInvalidParameters,
		ErrCardIDMismatch,
		ErrInsufficientData,
	}
	for _, err := range sentinels {
		if err == nil {
			t.Error("sentinel error is nil")
		}
	}
}

func TestSentinelErrorsIsCheck(t *testing.T) {
	// Wrapping with fmt.Errorf %w preserves errors.Is chain.
	wrapped := fmt.Errorf("context: %w", ErrInvalidRating)
	if !errors.Is(wrapped, ErrInvalidRating) {
		t.Error("errors.Is(wrapped, ErrInvalidRating) = false, want true")
	}
	if errors.Is(wrapped, ErrInvalidParameters) {
		t.Error("errors.Is(wrapped, ErrInvalidParameters) = true, want false")
	}
}

func TestSentinelErrorPrefix(t *testing.T) {
	tests := []struct {
		err    error
		prefix string
	}{
		{ErrInvalidRating, "flux: "},
		{ErrInvalidParameters, "flux: "},
		{ErrCardIDMismatch, "flux: "},
		{ErrInsufficientData, "flux: "},
	}
	for _, tt := range tests {
		msg := tt.err.Error()
		if len(msg) < len(tt.prefix) || msg[:len(tt.prefix)] != tt.prefix {
			t.Errorf("%v should start with %q, got %q", tt.err, tt.prefix, msg)
		}
	}
}

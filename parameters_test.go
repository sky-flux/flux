package flux

import (
	"errors"
	"testing"
)

func TestDefaultParametersLength(t *testing.T) {
	if len(DefaultParameters) != 21 {
		t.Errorf("len(DefaultParameters) = %d, want 21", len(DefaultParameters))
	}
}

func TestBoundsLength(t *testing.T) {
	if len(LowerBounds) != 21 {
		t.Errorf("len(LowerBounds) = %d, want 21", len(LowerBounds))
	}
	if len(UpperBounds) != 21 {
		t.Errorf("len(UpperBounds) = %d, want 21", len(UpperBounds))
	}
}

func TestDefaultParametersWithinBounds(t *testing.T) {
	for i := 0; i < 21; i++ {
		if DefaultParameters[i] < LowerBounds[i] || DefaultParameters[i] > UpperBounds[i] {
			t.Errorf("DefaultParameters[%d] = %f, out of [%f, %f]",
				i, DefaultParameters[i], LowerBounds[i], UpperBounds[i])
		}
	}
}

func TestLowerBoundsLessThanUpper(t *testing.T) {
	for i := 0; i < 21; i++ {
		if LowerBounds[i] > UpperBounds[i] {
			t.Errorf("LowerBounds[%d] = %f > UpperBounds[%d] = %f",
				i, LowerBounds[i], i, UpperBounds[i])
		}
	}
}

func TestValidateParametersValid(t *testing.T) {
	if err := ValidateParameters(DefaultParameters); err != nil {
		t.Errorf("ValidateParameters(DefaultParameters) = %v, want nil", err)
	}
}

func TestValidateParametersBelowLower(t *testing.T) {
	p := DefaultParameters
	p[0] = LowerBounds[0] - 1.0
	err := ValidateParameters(p)
	if err == nil {
		t.Fatal("ValidateParameters should fail for below-lower")
	}
	if !errors.Is(err, ErrInvalidParameters) {
		t.Errorf("error should wrap ErrInvalidParameters, got %v", err)
	}
}

func TestValidateParametersAboveUpper(t *testing.T) {
	p := DefaultParameters
	p[20] = UpperBounds[20] + 1.0
	err := ValidateParameters(p)
	if err == nil {
		t.Fatal("ValidateParameters should fail for above-upper")
	}
	if !errors.Is(err, ErrInvalidParameters) {
		t.Errorf("error should wrap ErrInvalidParameters, got %v", err)
	}
}

func TestValidateParametersExactBounds(t *testing.T) {
	// Parameters at exact lower bounds should be valid.
	if err := ValidateParameters(LowerBounds); err != nil {
		t.Errorf("ValidateParameters(LowerBounds) = %v, want nil", err)
	}
	// Parameters at exact upper bounds should be valid.
	if err := ValidateParameters(UpperBounds); err != nil {
		t.Errorf("ValidateParameters(UpperBounds) = %v, want nil", err)
	}
}

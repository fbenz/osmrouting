
package umath

import (
	"math"
	"math/rand"
	"testing"
)


// limits, but note that f > HalfMaxFloat does not necessarily imply that
// Float32ToHalf(f) == Infinity, as we might be rounding down.
const MaxHalf half = 0x7bff
const HalfMaxFloat = 65504.0 
const HalfMinFloat = 1.0 / float32(1 << 14)

// half has 10 bits mantissa, so with proper rounding we should have a
// relative error of at most 2^-11
const RelativePrecision = 1.0 / float64(1 << 11)

func TestPrecision(t *testing.T) {
	// Test the limits first
	halfMax := Float32ToHalf(HalfMaxFloat)
	if IsInfHalf(halfMax) {
		t.Error("Unable to represent HalfMax.")
	}
	
	halfMin := Float32ToHalf(HalfMinFloat)
	if HalfToFloat32(halfMin) != HalfMinFloat {
		t.Error("Unable to represent HalfMin.")
	}
	
	// Now do some random test. test/quick doesn't really work here, for
	// some reason so we do it manually.
	for i := 0; i < 1000; i++ {
		a := rand.Float32() * 2 * HalfMaxFloat
		h := Float32ToHalf(a)
		if math.Abs(float64(a)) > float64(HalfMaxFloat) {
			if !IsInfHalf(h) && h != MaxHalf {
				t.Errorf("%v did not overflow", a)
			}
		} else if math.Abs(float64(a)) < float64(HalfMinFloat) {
			if HalfToFloat32(h) != 0.0 {
				t.Errorf("%v did not underflow", a)
			}
		} else {
			b := HalfToFloat32(h)
			err := math.Abs(float64(a - b)) / float64(a)
			if err > RelativePrecision {
				t.Errorf("Relative error of %v on input %a (conversion: %v)", err, a, b)
			}
		}
	}
}

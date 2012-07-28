
package alg

import (
	"math"
	"math/rand"
	"testing"
)

// half has 10 bits mantissa, so with proper rounding we should have a
// relative error of at most 2^-11
const RelativePrecision = 1.0 / float64(1 << 11)

func TestPrecision(t *testing.T) {
	// Test the limits first
	halfMax := Float32ToHalf(MaxHalfFloat)
	if IsInfHalf(halfMax) {
		t.Error("Unable to represent HalfMax.")
	}
	
	halfMin := Float32ToHalf(MinHalfFloat)
	if HalfToFloat32(halfMin) != HalfMinFloat {
		t.Error("Unable to represent HalfMin.")
	}
	
	// Now do some random test. test/quick doesn't really work here, for
	// some reason so we do it manually.
	for i := 0; i < 1000; i++ {
		a := rand.Float32() * 2 * MaxHalfFloat
		h := Float32ToHalf(a)
		if math.Abs(float64(a)) > float64(MaxHalfFloat) {
			if !IsInfHalf(h) && h != MaxHalf {
				t.Errorf("%v did not overflow", a)
			}
		} else if math.Abs(float64(a)) < float64(MinHalfFloat) {
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

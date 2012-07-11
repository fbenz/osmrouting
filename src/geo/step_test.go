
package geo

import (
	"testing"
	"testing/quick"
)

func TestStepEncodeDecode(t *testing.T) {
	// The Generate method is defined in coordinate_test.go
	embedProject := func(start Coordinate, step []Coordinate) bool {
		// Encode it and Decode it again
		encoding := EncodeStep(start, step)
		decoded  := DecodeStep(start, encoding)
		for i, _ := range step {
			if !step[i].Equal(decoded[i]) {
				return false
			}
		}
		return true
	}
	
	if err := quick.Check(embedProject, nil); err != nil {
		t.Error(err)
	}
}

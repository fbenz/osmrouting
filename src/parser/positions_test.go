
package main

import (
	"geo"
	"testing"
	"testing/quick"
)

func TestPositions(t *testing.T) {
	// The Generate method is defined in coordinate_test.go
	positions := NewPositions(64)
	embedProject := func(i int64, c geo.Coordinate) bool {
		positions.Set(i, c)
		return c.Equal(positions.Get(i))
	}
	
	if err := quick.Check(embedProject, nil); err != nil {
		t.Error(err)
	}
}

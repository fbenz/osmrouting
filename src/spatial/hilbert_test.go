package spatial

import (
	"geo"
	"testing"
	"testing/quick"
)

func TestHilbertQuadrant(t *testing.T) {
	q0 := quadrant(0, 0, 1)
	q1 := quadrant(0, 1, 1)
	q2 := quadrant(1, 1, 1)
	q3 := quadrant(1, 0, 1)
	if q0 != 0 {
		t.Errorf("(0,0) ->! 0")
	}
	if q1 != 1 {
		t.Errorf("(0,1) ->! 1")
	}
	if q2 != 2 {
		t.Errorf("(1,1) ->! 2")
	}
	if q3 != 3 {
		t.Errorf("(1,0) ->! 3")
	}
}

func TestHilbertIrreflexive(t *testing.T) {
	irreflexive := func(a geo.Coordinate) bool {
		return !HilbertLess(a, a)
	}
	if err := quick.Check(irreflexive, nil); err != nil {
		t.Error(err)
	}
}

func TestHilbertAntisymmetry(t *testing.T) {
	antisym := func(a, b geo.Coordinate) bool {
		return !(HilbertLess(a, b) && HilbertLess(b, a))
	}
	if err := quick.Check(antisym, nil); err != nil {
		t.Error(err)
	}
}

func TestHilbertTransitivity(t *testing.T) {
	transitive := func(a, b, c geo.Coordinate) bool {
		if HilbertLess(a, b) && HilbertLess(b, c) {
			return HilbertLess(a, c)
		} else if HilbertLess(a, c) && HilbertLess(c, b) {
			return HilbertLess(a, b)
		} else if HilbertLess(b, a) && HilbertLess(a, c) {
			return HilbertLess(b, c)
		}
		return true
	}
	if err := quick.Check(transitive, nil); err != nil {
		t.Error(err)
	}
}

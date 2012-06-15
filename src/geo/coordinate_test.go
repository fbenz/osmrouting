package geo

import (
	"math"
	"math/rand"
	"testing"
	"testing/quick"
)

// Generate random coordinates.
func Generate(rand *rand.Rand, _ int) Coordinate {
	lng := rand.Float64() * 360.0 - 180.0
	lat := rand.Float64() * 180.0 - 90.0
	return Coordinate{lat, lng}
}

// Random tests for Encode/Decode.
func TestEncodeDecode(t *testing.T) {
	// Test that encode/decode is the identity on osm values.
	embeddingProjection := func(a Coordinate) bool {
		osm := Round(a)
		lat, lng := Encode(osm)
		return Decode(lat, lng) == osm
	}
	if err := quick.Check(embeddingProjection, nil); err != nil {
		t.Error(err)
	}
	
	// Otherwise the result should still be within epsilon from the source.
	decodeTolerance := func(a Coordinate) bool {
		lat, lng := Encode(a)
		return Equal(Decode(lat, lng), a)
	}
	if err := quick.Check(decodeTolerance, nil); err != nil {
		t.Error(err)
	}
}

// Test that rounding works correctly.
func TestEqualTolerance(t *testing.T) {
	roundEqual := func(a Coordinate) bool {
		return Equal(a, Round(a))
	}
	if err := quick.Check(roundEqual, nil); err != nil {
		t.Error(err)
	}
}

const (
	DeltaTolerance    = 1e-7
	DistanceTolerance = 0.002
)

type DistanceTestData struct {
	From, To Coordinate
	DeltaLat float64
	DeltaLng float64
	Distance float64
}

var distanceTests = [...]DistanceTestData {
	// Somewhere in Greenwhich
	{
		Coordinate{51.4809180306, -0.00529407598709},
		Coordinate{51.4799156000,  0.00561100000000},
		0.0010024306, 0.01090507598709, 763.4,
	},
	// Coordinates which do interesting things to mapping software.
	{
		Coordinate{-16.1562805556, -179.999658333},
		Coordinate{-16.1568638889,  179.999686111},
		0.0005833333, 0.000655556, 95.44,
	},
	// The same point twice, with different coordinates.
	{
		Coordinate{90.0, 130.0},
		Coordinate{90.0, -20.0},
		0.0, 150.0, 0.0,
	},
	// Random Tests from Ellipsoid
	{
		Coordinate{-38.369163,        10.874558},
		Coordinate{-38.3656166574817, 10.880662670944},
		0.0035463425182982178, 0.0061046709439995794, 663.027183,
	},
	{
		Coordinate{-1.549886,         156.466532},
		Coordinate{-1.56800166865689, 156.48838856866},
		0.01811566865688996, 0.021856568660012954, 3150.908018,
	},
}

// Test that Distance is accurate for small distances and that we
// handle all the corner cases for coordinates.
func TestDistance(t *testing.T) {
	for _, test := range distanceTests {
		deltaLat, deltaLng := Delta(test.From, test.To)
		distance := Distance(test.From, test.To)
		err := false
		if math.Abs(deltaLat - test.DeltaLat) > DeltaTolerance {
			t.Errorf("Wrong value for deltaLat: %.7f instead of %.7f",
				deltaLat, test.DeltaLat)
			err = true
		}
		if math.Abs(deltaLng - test.DeltaLng) > DeltaTolerance {
			t.Errorf("Wrong value for deltaLng: %.7f instead of %.7f",
				deltaLng, test.DeltaLng)
			err = true
		}
		if math.Abs(distance - test.Distance) > DistanceTolerance * test.Distance {
			t.Errorf("Wrong value for distance: %.2f instead of %.2f",
				distance, test.Distance)
			err = true
		}
		if err {
			t.Errorf("There were errors in test %v\n", test)
		}
	}
}

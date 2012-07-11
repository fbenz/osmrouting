
package geo

// About OpenStreemaps coordinates:
// Coordinates are stored as lattitude/longitude pairs referencing the
// WGS84 geodetic datum. All coordinates are stored with 7 decimal digits
// of precision.
// Since the coordinates are stored in degrees between [-180, 180] for
// longitude and [-90, 90] for lattitude this means that a coordinate
// could be encoded in 63 bits total. Simply store longitude as a value
// in [0, 3.6e9] and lattitude similarly as a value in [0, 1.8e9].
// Interestingly both are actually stored as int64 values, so I might be
// missing something. In any case, int64's are what you want to use
// for exact computations to avoid overflow.
// Anyway, since the values fit into 32 bits we can just do all
// computations using double precision floating point values (which
// have a 52 bits mantissa) without loosing precision.

import (
	"fmt"
	"math"
)

const (
	// Average "Great-Circle" radius of the earth in meter.
	GreatCircleRadius = 6372797.0
	// OsmEpsilon is the smallest difference between two osm coordinates.
	OsmEpsilon = 1e-7
	// The inverse of OsmEpsilon
	OsmPrecision = 1e7
)

// Coordinates are represented as a pair of double precision floating
// point numbers, for reasons explained above.
type Coordinate struct {
	Lat float64
	Lng float64
}

// Decode a coordinate given its fixed point representation.
func DecodeCoordinate(lat, lng int32) Coordinate {
	return Coordinate{
		Lat: float64(lat) / OsmPrecision,
		Lng: float64(lng) / OsmPrecision,
	}
}

// Represent a coordinate in fixed point format.
func (a Coordinate) Encode() (lat, lng int32) {
	lat = int32(math.Floor(a.Lat * OsmPrecision + 0.5))
	lng = int32(math.Floor(a.Lng * OsmPrecision + 0.5))
	return
}

// Round a Coordindate to osm precision.
func (a Coordinate) Round() Coordinate {
	return Coordinate{
		Lat: math.Floor(a.Lat * OsmPrecision + 0.5) / OsmPrecision,
		Lng: math.Floor(a.Lng * OsmPrecision + 0.5) / OsmPrecision,
	}
}

// Implement the fmt.Stringer interface for coordinates to make %v or
// %s work in format strings.
func (c Coordinate) String() string {
	return fmt.Sprintf("(%.7f, %.7f)", c.Lat, c.Lng)
}

// Test if two components differ by less than epsilon.
func EqualTolerance(a, b float64) bool {
	return math.Abs(a - b) <= OsmEpsilon / 2.0
}

// Test if two Coordinates differ by less than epsilon.
func (a Coordinate) Equal(b Coordinate) bool {
	return EqualTolerance(a.Lat, b.Lat) && EqualTolerance(a.Lng, b.Lng)
}

// Compute the difference in latitude, longitude for a and b,
// handling wrap around as appropriate.
func (a Coordinate) Delta(b Coordinate) (lat, lng float64) {
	lat = math.Abs(a.Lat - b.Lat)
	lng = math.Mod(math.Abs(a.Lng - b.Lng), 360.0)
	return lat, math.Min(lng, 360 - lng)
}

// Compute an approximate distance between a and b in meter.
// Note that this is an euclidean approximation. If a and b are far
// apart the result is meaningless.
// On the other hand this is fast and stable for small differences
// in the coordinates.
func (a Coordinate) Distance(b Coordinate) float64 {
	deltaLat, deltaLng := a.Delta(b)
	// Convert to Radians
	deltaLat = (deltaLat * math.Pi) / 180.0
	deltaLng = (deltaLng * math.Pi) / 180.0
	aLat := a.Lat * math.Pi / 180.0
	bLat := b.Lat * math.Pi / 180.0
	// Euclidean distance
	cos2      := (1.0 + math.Cos(aLat + bLat)) / 2.0
	deltaLat2 := deltaLat * deltaLat
	deltaLng2 := deltaLng * deltaLng
	return GreatCircleRadius * math.Sqrt(deltaLat2 + cos2 * deltaLng2)
}

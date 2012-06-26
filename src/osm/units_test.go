package osm

import (
	"math"
	"testing"
)

var boolTable = [...]struct {
	string
	bool
}{
	{"yes", true}, {"Yes", true},
	{"no", false}, {"No", false},
	{"true", true}, {"false", false},
	{"1", true}, {"0", false},
}

func TestParseBool(t *testing.T) {
	for _, test := range boolTable {
		r := ParseBool(test.string)
		if r != test.bool {
			t.Errorf("ParseBool(%s) = %v (should be %v)\n", test.string, r, test.bool)
		}
	}
}

// Random maxspeed entries from tagwatch. For the most part the data is clean
// (apart from Countrycode:Type entries), but there is a long tail of noisy entries.
var speedTable = [...]struct {
	string
	float64
}{
	{"30", 30.0}, {"50", 50.0}, {"100", 100.0},
	{"31.25", 31.25}, {"6,5", 6.5},
	{"30 km", 30.0}, {"20kmh", 20.0},
	{"30km/h", 30.0}, {"50 Km/h", 50.0},
	{"20 mph", 32.18688}, {"10 kph", 10.0},
	{"30mph", 48.28032}, {"6 Knoten", 11.112},
	{"+70", 70.0}, {"0020", 20.0},
	{"2 knots", 3.704}, {"5 kt", 9.26},
	{"100 KM", 100.0}, {"30 Km/Hour", 30.0},
	{"50 kmph", 50.0}, {"40 kmp", 40.0},
	{"40 KPH", 40.0}, {"20km/hr", 20.0},
}

func TestParseSpeed(t *testing.T) {
	for _, test := range speedTable {
		r, err := ParseSpeed(test.string)
		if err != nil {
			t.Errorf("ParseSpeed(%s) exception: %v\n", test.string, err)
		} else if r != test.float64 {
			t.Errorf("ParseSpeed(%s) = %v (should be %v)\n", test.string, r, test.float64)
		}
	}
}

var lengthTable = [...]struct {
	string
	float64
}{
	// Common mistakes according to the wiki
	{"2km", 2000.0}, {"2 km", 2000.0},
	{"0,6", 0.6}, {"0.6", 0.6},
	{"12' 6\"", 3.81}, {"12'6\"", 3.81},
	// Tagwatch examples for maxwidth
	{"2", 2.0}, {"2.2", 2.2}, {"2.5", 2.5},
	{"6'", 1.829}, {"6\"", 0.1524},
	{"7ft", 2.134}, {"7ft6in", 2.286},
	{"2,3", 2.3}, {"2.3m", 2.3},
	{"3,10", 3.1}, {"6'-6\"", 1.981},
	// Tagwatch values for length
	{"1 meter", 1.0}, {"1 metre", 1.0},
	{"1,5 Meter", 1.5}, {"12 м", 12.0},
	{"130cm", 1.3},
	// Other examples
	{"0.6 mi", 965.6064},
}

func TestParseLength(t *testing.T) {
	for _, test := range lengthTable {
		r, err := ParseLength(test.string)
		if err != nil {
			t.Errorf("ParseLength(%s) exception: %v\n", test.string, err)
		} else if math.Abs(r-test.float64) > 0.001 {
			t.Errorf("ParseLength(%s) = %v (should be %v)\n", test.string, r, test.float64)
		}
	}
}

var weightTable = [...]struct {
	string
	float64
}{
	// Tagwatch examples for maxweight
	{"7.5", 7.5}, {"3.5", 3.5},
	{"7.5T", 7.5}, {"3.5t", 3.5},
	{"7,5", 7.5}, {"22", 22.0},
	{"7.5 tons", 7.5}, {"1,5", 1.5},
	{"7.5 tonnes", 7.5}, {"15 ton", 15.0},
	{"3te", 3.0}, {"200kg", 0.2},
	{"5 To", 5.0}, {"3000 kg", 3.0},
	{"108000 lbs", 48.988},
}

func TestParseWeight(t *testing.T) {
	for _, test := range weightTable {
		r, err := ParseWeight(test.string)
		if err != nil {
			t.Errorf("ParseWeight(%s) exception: %v\n", test.string, err)
		} else if math.Abs(r-test.float64) > 0.001 {
			t.Errorf("ParseWeight(%s) = %v (should be %v)\n", test.string, r, test.float64)
		}
	}
}

package osm

// Parse units and standard data types. Tries to be a 90% solution, since
// in practice the data is often not really correct.

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var MatchUnit *regexp.Regexp
var MatchFeet *regexp.Regexp

func init() {
	var err error
	MatchUnit, err = regexp.Compile("^\\s*\\+?(\\d+(?:[.,]\\d+)?)\\s*([/\\w]*)")
	if err != nil {
		panic("could not compile regular expression")
	}

	MatchFeet, err = regexp.Compile("^\\s*(?:(\\d+)(?:feet|ft|'))?\\s*(?:-?\\s*(\\d+)(?:in|\"))?")
	if err != nil {
		panic("could not compile regular expression")
	}
}

// Returns false on unrecognized inputs, which includes keys which are not
// present to begin with.
func ParseBool(value string) bool {
	switch strings.ToLower(value) {
	case "yes", "true", "1", "designated":
		return true
	}
	return false
}

// Parse a floating point number with a potentially wrong decimal separator.
func parseNumber(s string) (float64, error) {
	var r float64
	var err error

	if strings.ContainsRune(s, ',') {
		s = strings.Replace(s, ",", ".", 1)
	}

	if strings.ContainsRune(s, '.') {
		r, err = strconv.ParseFloat(s, 64)
	} else {
		var i int
		i, err = strconv.Atoi(s)
		r = float64(i)
	}

	if err != nil {
		return 0.0, err
	}
	return r, nil
}

// Parse a number in feet'inch" format.
func parseLengthImperial(s string) (float64, bool, error) {
	if match := MatchFeet.FindStringSubmatch(s); match != nil {
		inch := 0
		valid := false

		if len(match[1]) != 0 {
			n, err := strconv.Atoi(match[1])
			if err != nil {
				return 0.0, true, err
			}
			inch += 12 * n
			valid = true
		}

		if len(match[2]) != 0 {
			n, err := strconv.Atoi(match[2])
			if err != nil {
				return 0.0, true, err
			}
			inch += n
			valid = true
		}

		if valid {
			return float64(inch) * 0.0254, true, nil
		}
	}
	return 0.0, false, nil
}

// Result in meter.
func ParseLength(s string) (float64, error) {
	length, valid, err := parseLengthImperial(s)
	if valid && err != nil {
		return 0.0, err
	} else if valid {
		return length, nil
	}

	if match := MatchUnit.FindStringSubmatch(s); match != nil {
		base, err := parseNumber(match[1])
		if err != nil {
			return 0.0, err
		}

		// If there is no explicit unit, it's already in meter.
		if len(match[2]) == 0 {
			return base, nil
		}

		// distinguish based on the unit specification.
		switch strings.ToLower(match[2]) {
		case "m", "metre", "metres", "meter":
			return base, nil
		case "cm":
			return base / 100.0, nil
		case "km", "kilometre", "kilometres", "kilometer", "kilometers":
			return base * 1000.0, nil
		case "mi", "mile", "miles":
			return base * 1609.344, nil
		}
	}

	// If we get here the string is just wrong.
	return 0.0, errors.New("Wrong length format: " + s)
}

// Result is in km/h.
func ParseSpeed(s string) (float64, error) {
	if match := MatchUnit.FindStringSubmatch(s); match != nil {
		base, err := parseNumber(match[1])
		if err != nil {
			return 0.0, err
		}
		if len(match[2]) == 0 {
			return base, nil
		}

		switch strings.ToLower(match[2]) {
		case "km/h", "kph", "kmph", "km",
			"kmh", "km/hour", "kmp", "km/hr":
			return base, nil
		case "mph":
			return base * 1.609344, nil
		case "knots", "knoten", "kt":
			return base * 1.852, nil
		}
	}

	return 0.0, errors.New("Wrong speed format: " + s)
}

// Result in tons.
func ParseWeight(s string) (float64, error) {
	if match := MatchUnit.FindStringSubmatch(s); match != nil {
		base, err := parseNumber(match[1])
		if err != nil {
			return 0.0, err
		}

		if len(match[2]) == 0 {
			return base, nil
		}

		switch strings.ToLower(match[2]) {
		case "t", "tons", "tonnes", "to", "ton", "te":
			return base, nil
		case "kg":
			return base / 1000.0, nil
		case "lb", "lbs":
			return base * 0.4536 / 1000.0, nil
		}
	}

	return 0.0, errors.New("Wrong weight format: " + s)
}

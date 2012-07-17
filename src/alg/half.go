
package alg

import "math"

func Float32ToHalf(a float32) uint16 {
	bits := math.Float32bits(a)
	s := uint16(bits >> 16) & 0x8000
	e := uint16(bits >> 23) & 0x00ff
	m := bits & 0x007fffff
	
	// Zero (or underflow)
	// e between 104 and 112 indicates a denormal number which we might
	// still be able to represent (we would loose some bits though).
	// On the other hand, this indicates a number less than 2^-14.
	// Since we use half floats to represent distances in meter, this
	// corresponds to measurments less than one micrometer. This is far
	// smaller than the best accuracy we could ever hope for wih
	// GPS data, so we just round everything to 0.
	if e < 113 {
		return s
	} else if e > 142 {
		// overflow, infinity, or NaN
		if e == 0xff && m != 0 {
			// NaN. Set the most significant bit of the mantissa, and ignore the sign.
			// This ensure that we get a quiet NaN after converting back to a real
			// floating point number. And no one uses the sign on an NaN...
			return 0x7e00
		} else {
			return s | 0x7c00 // +/- Infinity
		}
	}
	
	// Rounding is really easy at this point, because the exponent is always
	// less than 31 we can afford to overflow the mantissa. Doing so will
	// increment the exponent field, while setting the mantissa to 0, resulting
	// in infinity if the exponent was 30 before and in 2^(e - 15) otherwise.
	r := s | ((e - 112) << 10) | uint16(m >> 13)
	r += uint16((m >> 12) & 1)
	return r
}

func HalfToFloat32(a uint16) float32 {
	s := uint32(a & 0x8000) << 16
	e := uint32(a >> 10) & 0x1f
	m := uint32(a & 0x3ff)
	
	if e == 0 {
		// +/- 0, we don't produce denormals
		// (mainly because we would have to turn them into a
		// normalized number here and that's costly)
		return math.Float32frombits(s)
	} else if e == 31 {
		if m == 0 {
			// Infinity
			return math.Float32frombits(s | 0x7f800000)
		} else {
			// NaN
			return math.Float32frombits(0x7f800000 | (m << 13))
		}
	}
	
	return math.Float32frombits(s | ((e + 112) << 23) | (m << 13))
}

func Float64ToHalf(a float64) uint16 {
	return Float32ToHalf(float32(a))
}

func HalfToFloat64(a uint16) float64 {
	return float64(HalfToFloat32(a))
}

func IsNanHalf(a uint16) bool {
	m := a & 0xff
	return m != 0 && a & 0x7c00 == 0x7c00
}

func IsInfHalf(a uint16) bool {
	m := a & 0xff
	return m == 0 && a & 0x7c00 == 0x7c00
}

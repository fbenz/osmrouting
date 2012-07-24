
package spatial

import "geo"

func quadrant(x, y, m uint32) int {
	rx, ry := 0, 0
	if x & m != 0 {
		rx = 1
	}
	if y & m != 0 {
		ry = 1
	}
	return 3 * rx ^ ry
}

func HilbertLess(a, b geo.Coordinate) bool {
	x0, y0 := a.EncodeUint()
	x1, y1 := b.EncodeUint()
	for m := uint32(1 << 31); m > 0; m /= 2 {
		// Determine the curve quadrant for each of the points.
		// If they are different, we're done.
		rx0 := x0 & m != 0
		rx1 := x1 & m != 0
		ry0 := y0 & m != 0
		ry1 := y1 & m != 0
		if rx0 != rx1 || ry0 != ry1 {
			return quadrant(x0, y0, m) < quadrant(x1, y1, m)
		}
		
		// Otherwise we have to remap the curve into the first quadrant
		// and recurse.
		if ry0 {
			if !rx0 {
				x0, y0 = ^x0, ^y0
				x1, y1 = ^x1, ^y1
			}
			x0, y0 = y0, x0
			x1, y1 = y1, x1
		}
	}
	// We only get here if the points are actually equal
	return false
}

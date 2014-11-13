/*
 * Copyright 2014 Florian Benz, Steven Schäfer, Bernhard Schommer
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */


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

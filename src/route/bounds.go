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

package route

import "math"

// Computes the union of two BoundingBox'es.
func BoxUnion(a, b BoundingBox) BoundingBox {
	// Ok, maybe this wasn't such a good idea after all:
	nwLat := math.Max(a.Northwest[0], b.Northwest[0])
	nwLng := math.Min(a.Northwest[1], b.Northwest[1])
	seLat := math.Min(a.Southeast[0], b.Southeast[0])
	seLng := math.Max(a.Southeast[1], b.Southeast[1])
	return BoundingBox{Point{nwLat, nwLng}, Point{seLat, seLng}}
}

// Compute a tight bounding box for a single step.
func ComputeBoundsStep(step Step) BoundingBox {
	if len(step.Polyline) == 0 {
		// Bug?
		return BoundingBox{Point{0.0, 0.0}, Point{0.0, 0.0}}
	}

	bounds := BoundingBox{step.Polyline[0], step.Polyline[0]}
	if len(step.Polyline) > 1 {
		for _, point := range step.Polyline[1:] {
			box := BoundingBox{point, point}
			bounds = BoxUnion(bounds, box)
		}
	}
	return bounds
}

// Compute a thight bounding box for a leg
func ComputeBoundsLeg(leg Leg) BoundingBox {
	if len(leg.Steps) == 0 {
		// Bug?
		return BoundingBox{Point{0.0, 0.0}, Point{0.0, 0.0}}
	}

	bounds := ComputeBoundsStep(leg.Steps[0])
	if len(leg.Steps) > 1 {
		for _, step := range leg.Steps[1:] {
			bounds = BoxUnion(bounds, ComputeBoundsStep(step))
		}
	}
	return bounds
}

// Compute a BoundingBox containing a whole path, plus some extra
// space for aesthetics.
func ComputeBounds(route Route) BoundingBox {
	if len(route.Legs) == 0 {
		// This is a bug...
		return BoundingBox{Point{0.0, 0.0}, Point{0.0, 0.0}}
	}

	bounds := ComputeBoundsLeg(route.Legs[0])
	if len(route.Legs) > 1 {
		for _, leg := range route.Legs[1:] {
			bounds = BoxUnion(bounds, ComputeBoundsLeg(leg))
		}
	}

	return bounds
}

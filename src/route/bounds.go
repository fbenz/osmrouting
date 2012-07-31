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

package route

import (
	"fmt"
	"graph"
	"math"
)

// Returns a human readable string for the given distance value.
func FormatDistance(distance float64) Distance {
	switch {
	case distance < 0.5:
		// Yes, this is a gag. Why do you even have to ask?
		t := fmt.Sprintf("%.2f mm", distance*1000.0)
		return Distance{t, int(distance)}
	case distance < 500.0:
		t := fmt.Sprintf("%.2f m", distance)
		return Distance{t, int(distance)}
	}
	t := fmt.Sprintf("%.2f km", distance/1000.0)
	return Distance{t, int(distance)}
}

// Returns a human readable formatting for the given time-span.
func FormatDuration(seconds float64) Duration {
	switch {
	case seconds < 30.0:
		t := fmt.Sprintf("%.2f secs", seconds)
		return Duration{t, int(seconds)}
	case seconds < 1800.0:
		t := fmt.Sprintf("%.2f mins", seconds/60.0)
		return Duration{t, int(seconds)}
	}
	t := fmt.Sprintf("%.2f hours", seconds/3600.0)
	return Duration{t, int(seconds)}
}

// Since we don't actually have max_speed values yet, we make
// something up for the time values.
// Wolfram Alpha tells me that the "typical human walking speed"
// is 1.1 m/s. So let's just roll with that.
func MockupDuration(distance float64) Duration {
	return FormatDuration(distance / 1.1)
}

// Convert from graph.Step to a Point.
func StepToPoint(step graph.Step) Point {
	return Point{step.Lat, step.Lng}
}

func NodeToStep(g graph.Graph, node graph.Node) graph.Step {
	lat, lng := g.NodeLatLng(node)
	return graph.Step{lat, lng}
}

// Given a path from start to stop with intermediate steps, turn
// it into a Polyline for json output.
func StepsToPolyline(steps []graph.Step, start, stop graph.Step) Polyline {
	polyline := make([]Point, len(steps)+2)
	polyline[0] = StepToPoint(start)
	polyline[len(steps)+1] = StepToPoint(stop)
	for i, s := range steps {
		polyline[i+1] = StepToPoint(s)
	}
	return polyline
}

// Convert the path from start - steps - stop to a json Step
func PartwayToStep(steps []graph.Step, start, stop graph.Step, length float64) Step {
	instruction := fmt.Sprintf(
		"Walk from (%.4f, %.4f) to (%.4f, %.4f)",
		start.Lat, start.Lng, stop.Lat, stop.Lng)
	return Step{
		Distance:      FormatDistance(length),
		Duration:      MockupDuration(length),
		StartLocation: StepToPoint(start),
		EndLocation:   StepToPoint(stop),
		Polyline:      StepsToPolyline(steps, start, stop),
		Instruction:   instruction,
	}
}

// Convert a Path (start - steps - stop) into a json Step structure.
// This contains some additional information, which might or might
// not be accurate.
func WayToStep(steps graph.Way, start, stop graph.Step) Step {
	return PartwayToStep(steps.Steps, start, stop, steps.Length)
}

// Convert an Edge (u,v) into a json Step
func EdgeToStep(g graph.Graph, edge graph.Edge, u, v graph.Node) Step {
	return PartwayToStep(g.EdgeSteps(edge), NodeToStep(g, u), NodeToStep(g, v), g.EdgeLength(edge))
}

// Convert a single path as returned by Dijkstra to a json Leg.
func PathToLeg(g graph.Graph, distance float64, vertices []graph.Node, edges []graph.Edge, start, stop graph.Way) Leg {
	// Determine the number of steps on this path.
	totalSteps := len(edges)
	if start.Length > 1e-7 {
		totalSteps++
	}
	if stop.Length > 1e-7 {
		totalSteps++
	}
	steps := make([]Step, totalSteps)

	// Add the initial step, if present
	i := 0
	if start.Length > 1e-7 {
		// Our implementation of Dijkstra's algorithm ensures len(vertices) > 0
		steps[0] = WayToStep(start, start.Target, NodeToStep(g, vertices[0]))
		i++
	}

	// Add the intermediate steps
	for j, edge := range edges {
		from := vertices[j]
		to := vertices[j+1]
		steps[i] = EdgeToStep(g, edge, from, to)
		i++
	}

	// Add the final step, if present
	if stop.Length > 1e-7 {
		prev := vertices[len(vertices)-1]
		steps[i] = WayToStep(stop, NodeToStep(g, prev), stop.Target)
	}

	return Leg{
		Distance:      FormatDistance(distance),
		Duration:      MockupDuration(distance),
		StartLocation: StepToPoint(start.Target),
		EndLocation:   StepToPoint(stop.Target),
		Steps:         steps,
	}
}

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

package route

import (
	"fmt"
	"geo"
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

// Convert from geo.Coordinate to a Point.
func StepToPoint(step geo.Coordinate) Point {
	return Point{step.Lat, step.Lng}
}

// Given a path from start to stop with intermediate steps, turn
// it into a Polyline for json output.
func StepsToPolyline(steps []geo.Coordinate, start, stop geo.Coordinate) Polyline {
	polyline := make([]Point, len(steps)+2)
	polyline[0] = StepToPoint(start)
	polyline[len(steps)+1] = StepToPoint(stop)
	for i, s := range steps {
		polyline[i+1] = StepToPoint(s)
	}
	return polyline
}

// Convert the path from start - steps - stop to a json Step
func PartwayToStep(steps []geo.Coordinate, start, stop geo.Coordinate, maxSpeed int, c Config) Step {
	instruction := fmt.Sprintf(
		"Walk from (%.4f, %.4f) to (%.4f, %.4f)",
		start.Lat, start.Lng, stop.Lat, stop.Lng)
	length := geo.StepLength(append(append([]geo.Coordinate{start}, steps...), stop))
	
	// For cars, we know how fast they can go... otherwise, we make something up.
	duration := float64(0)
	if c.Transport == graph.Car && maxSpeed != 0 {
		// length is in meter, maxSpeed is in km/h, duration is in seconds.
		duration = length * (3600.0 / 1000.0) / float64(maxSpeed)
	} else if c.Transport == graph.Car {
		duration = length * 0.12 // 30 km/h
	} else if c.Transport == graph.Foot {
		duration = length / 1.1
	} else {
		duration = length / 13.0
	}
	
	return Step{
		Distance:      FormatDistance(length),
		Duration:      FormatDuration(duration),
		StartLocation: StepToPoint(start),
		EndLocation:   StepToPoint(stop),
		Polyline:      StepsToPolyline(steps, start, stop),
		Instruction:   instruction,
	}
}

// Convert a Path (start - steps - stop) into a json Step structure.
// This contains some additional information, which might or might
// not be accurate.
func WayToStep(steps graph.Way, start, stop geo.Coordinate, c Config) Step {
	return PartwayToStep(steps.Steps, start, stop, 0, c)
}

// Convert an Edge (u,v) into a json Step
func EdgeToStep(g graph.Graph, edge graph.Edge, u, v graph.Vertex, c Config) Step {
	step := g.EdgeSteps(edge, u, nil)
	upos := g.VertexCoordinate(u)
	vpos := g.VertexCoordinate(v)
	speed := g.EdgeMaxSpeed(edge)
	return PartwayToStep(step, upos, vpos, speed, c)
}

// Convert a single path as returned by Dijkstra to a json Leg.
func PathToLeg(g graph.Graph, vertices []graph.Vertex, edges []graph.Edge, start, stop *graph.Way, c Config) *Leg {
	// Determine the number of steps on this path.
	var startPoint, endPoint Point
	totalSteps := len(edges)
	if start != nil && start.Length > 1e-7 {
		totalSteps++
	}
	if stop != nil && stop.Length > 1e-7 {
		totalSteps++
	}
	steps := make([]Step, totalSteps)

	distance := 0
	duration := 0

	// Add the initial step, if present
	i := 0
	if start != nil && start.Length > 1e-7 {
		// Our implementation of Dijkstra's algorithm ensures len(vertices) > 0
		step := WayToStep(*start, start.Target, g.VertexCoordinate(vertices[0]), c)
		distance += step.Distance.Value
		duration += step.Duration.Value
		steps[i] = step
		i++
	}

	// Add the intermediate steps
	for j, edge := range edges {
		from := vertices[j]
		to := vertices[j+1]
		step := EdgeToStep(g, edge, from, to, c)
		distance += step.Distance.Value
		duration += step.Duration.Value
		steps[i] = step
		i++
	}

	// Add the final step, if present
	if stop != nil && stop.Length > 1e-7 {
		prev := vertices[len(vertices)-1]
		step := WayToStep(*stop, g.VertexCoordinate(prev), stop.Target, c)
		distance += step.Distance.Value
		duration += step.Duration.Value
		steps[i] = step
		i++
	}
	
	if start != nil {
		startPoint = StepToPoint(start.Target)
	} else {
		startPoint = steps[0].StartLocation
	}

	if stop != nil {
		endPoint = StepToPoint(stop.Target)
	} else {
		endPoint = steps[len(steps)-1].EndLocation
	}
	
	return &Leg{
		Distance:      FormatDistance(float64(distance)),
		Duration:      FormatDuration(float64(duration)),
		StartLocation: startPoint,
		EndLocation:   endPoint,
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
func ComputeBoundsLeg(leg *Leg) BoundingBox {
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

// Combine two legs into one leg
func CombineLegs(a, b *Leg) *Leg {
	distance := a.Distance.Value + b.Distance.Value
	duration := a.Duration.Value + b.Duration.Value
	steps := append(a.Steps, b.Steps...)
	start := a.StartLocation
	end := b.EndLocation
	return &Leg{
		Distance:      FormatDistance(float64(distance)),
		Duration:      FormatDuration(float64(duration)),
		StartLocation: start,
		EndLocation:   end,
		Steps:         steps,
	}
}

//Append a Step to a leg
func AppendStep(a *Leg, b *Step) *Leg {
	distance := a.Distance.Value + b.Distance.Value
	duration := a.Duration.Value + b.Duration.Value
	steps := append(a.Steps, *b)
	start := a.StartLocation
	end := b.EndLocation
	return &Leg{
		Distance:      FormatDistance(float64(distance)),
		Duration:      FormatDuration(float64(duration)),
		StartLocation: start,
		EndLocation:   end,
		Steps:         steps,
	}
}

/*
func WayToLeg(way *graph.Way, g graph.Graph, forward bool, target graph.Vertex) *Leg {
	var step Step
	if forward {
		step = PartwayToStep(way.Steps, way.Target, g.VertexCoordinate(target), 0)
	} else {
		step = PartwayToStep(way.Steps, g.VertexCoordinate(target), way.Target, 0)
	}
	return &Leg{
		Distance:      step.Distance,
		Duration:      step.Duration,
		StartLocation: step.StartLocation,
		EndLocation:   step.EndLocation,
		Steps:         []Step{step},
	}
}
*/

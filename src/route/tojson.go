package route

import (
	"fmt"
	"geo"
	"graph"
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
func (r *RoutePlanner) PartwayToStep(steps []geo.Coordinate, start, stop geo.Coordinate, maxSpeed int) Step {
	length := geo.StepLength(append(append([]geo.Coordinate{start}, steps...), stop))

	// For cars, we know how fast they can go... otherwise, we make something up.
	duration := float64(0)
	if r.Transport == graph.Car && maxSpeed != 0 {
		// length is in meter, maxSpeed is in km/h, duration is in seconds.
		duration = length * (3600.0 / 1000.0) / float64(maxSpeed)
	} else if r.Transport == graph.Car {
		duration = length * 0.12 // 30 km/h
	} else if r.Transport == graph.Foot {
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
	}
}

// Convert a Path (start - steps - stop) into a json Step structure.
// This contains some additional information, which might or might
// not be accurate.
func (r *RoutePlanner) WayToStep(steps graph.Way, start, stop geo.Coordinate) Step {
	return r.PartwayToStep(steps.Steps, start, stop, 0)
}

// Convert an Edge (u,v) into a json Step
func (r *RoutePlanner) EdgeToStep(g graph.Graph, edge graph.Edge, u, v graph.Vertex) Step {
	step := g.EdgeSteps(edge, u, nil)
	upos := g.VertexCoordinate(u)
	vpos := g.VertexCoordinate(v)
	speed := g.EdgeMaxSpeed(edge)
	return r.PartwayToStep(step, upos, vpos, speed)
}

func Orientation(p, q, r Point) string {
	s := (q[0]-p[0])*(r[1]-p[1]) - (q[1]-p[1])*(r[0]-p[0])
	if s < 1e-9 && s > -1e-9 {
		return "Continue straightforward"
	}
	if s > 0 {
		return "Turn right"
	}
	return "Turn left"
}

// Assemble a sequence of Steps into a Leg.
func (r *RoutePlanner) StepsToLeg(steps []Step, start, stop graph.Way, startc, stopc geo.Coordinate) Leg {
	// Determine the number of steps on this path.
	var startPoint, endPoint Point
	totalSteps := len(steps)
	if start.Length > 1e-7 {
		totalSteps++
	}
	if stop.Length > 1e-7 {
		totalSteps++
	}
	fullsteps := make([]Step, totalSteps)

	distance := 0
	duration := 0

	// Add the initial step, if present
	i := 0
	if start.Length > 1e-7 {
		// Our implementation of Dijkstra's algorithm ensures len(vertices) > 0
		step := r.WayToStep(start, start.Target, startc)
		distance += step.Distance.Value
		duration += step.Duration.Value
		fullsteps[i] = step
		i++
	}

	// Add the intermediate steps
	for _, step := range steps {
		if i > 0 {
			step.Instruction = Orientation(fullsteps[i-1].StartLocation, fullsteps[i-1].EndLocation, step.EndLocation)
		}
		distance += step.Distance.Value
		duration += step.Duration.Value
		fullsteps[i] = step
		i++
	}

	// Add the final step, if present
	if stop.Length > 1e-7 {
		step := r.WayToStep(stop, stopc, stop.Target)
		if i > 0 {
			step.Instruction = Orientation(fullsteps[i-1].StartLocation, fullsteps[i-1].EndLocation, step.EndLocation)
		}
		distance += step.Distance.Value
		duration += step.Duration.Value
		fullsteps[i] = step
		i++
	}

	if totalSteps > 0 {
		fullsteps[0].Instruction = "Start your journey"
	}
	startPoint = StepToPoint(start.Target)
	endPoint = StepToPoint(stop.Target)

	return Leg{
		Distance:      FormatDistance(float64(distance)),
		Duration:      FormatDuration(float64(duration)),
		StartLocation: startPoint,
		EndLocation:   endPoint,
		Steps:         fullsteps,
	}
}

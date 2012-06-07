package main

import (
	"graph"
	"container/list"
	"fmt"
)

// Returns a human readable string for the given distance value.
func FormatDistance(distance float64) Distance {
	switch {
	case distance < 0.5:
		// Yes, this is a gag. Why do you even have to ask?
		t := fmt.Sprintf("%.2f mm", distance * 1000.0)
		return Distance{t, int(distance)}
	case distance < 500.0:
		t := fmt.Sprintf("%.2f m", distance)
		return Distance{t, int(distance)}
	}
	t := fmt.Sprintf("%.2f km", distance / 1000.0)
	return Distance{t, int(distance)}
}

// Returns a human readable formatting for the given time-span.
func FormatDuration(seconds float64) Duration {
	switch {
	case seconds < 30.0:
		t := fmt.Sprintf("%.2f s", seconds)
		return Duration{t, int(seconds)}
	case seconds < 1800.0:
		t := fmt.Sprintf("%.2f m", seconds / 60.0)
		return Duration{t, int(seconds)}
	}
	t := fmt.Sprintf("%.2f h", seconds / 3600.0)
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

func NodeToStep(node graph.Node) graph.Step {
	lat, lng := node.LatLng()
	return graph.Step{lat, lng}
}

// Given a path from start to stop with intermediate steps, turn
// it into a Polyline for json output.
func StepsToPolyline(steps []graph.Step, start, stop graph.Step) Polyline {
	polyline := make([]Point,len(steps)+2)
	polyline[0] = StepToPoint(start)
	polyline[len(steps)+1] = StepToPoint(stop)
	for i,s:=range steps {
		polyline[i + 1] = StepToPoint(s)
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
func EdgeToStep(edge graph.Edge, u, v graph.Node) Step {
	return PartwayToStep(edge.Steps(), NodeToStep(u), NodeToStep(v), edge.Length())
}

func PathToLeg(distance float64, vertices, edges *list.List, start, stop graph.Way) Leg {
	// Determine the number of steps on this path.
	totalSteps := edges.Len()
	if len(start.Steps) > 0 {
		totalSteps++
	}
	if len(stop.Steps) > 0 {
		totalSteps++
	}
	steps:=make([]Step,totalSteps)
	
	// Add the initial step, if present
	i := 0
	if len(start.Steps) > 0 {
		next := vertices.Front().Value.(graph.Node)
		steps[0] = WayToStep(start, start.Target, NodeToStep(next))
		i++
	}
	
	// Add the intermediate steps
	vertexIter := vertices.Front()
	edgeIter   := edges.Front()
	for edgeIter != nil {
		edge := edgeIter.Value.(graph.Edge)
		from := vertexIter.Value.(graph.Node)
		to   := vertexIter.Next().Value.(graph.Node)
		steps[i] = EdgeToStep(edge, from, to)
		vertexIter = vertexIter.Next()
		edgeIter   = edgeIter.Next()
		i++
	}
	
	// Add the final step, if present
	if len(stop.Steps) > 0 {
		prev := vertices.Back().Value.(graph.Node)
		steps[i] = WayToStep(stop, NodeToStep(prev), stop.Target)
	}
	
	return Leg{
		Distance:      FormatDistance(distance),
		Duration:      MockupDuration(distance),
		StartLocation: StepToPoint(start.Target),
		EndLocation:   StepToPoint(stop.Target),
		Steps:         steps,
	}
}

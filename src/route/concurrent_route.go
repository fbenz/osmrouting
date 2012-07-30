// Actually we don't want to do this concurrently but to pass all checks
// with our current implementation this is necessary.

package route

import (
	"graph"
)

func ConcurrentRoutes(g *graph.ClusterGraph, waypoints []Point, config Config) *Result {
	distance := 0.0
	duration := 0.0
	legs := make([]*Leg, len(waypoints)-1)
	if len(waypoints)-1 > 1 {
		c := make(chan int, len(waypoints)-1)
		// fork
		for i := 0; i < len(waypoints)-1; i++ {
			go func(j int) {
				legs[j] = leg(g, waypoints, j, config)
				c <- j
			}(i)
		}

		// join
		for i := 0; i < len(waypoints)-1; i++ {
			<-c
		}

		for i := 0; i < len(waypoints)-1; i++ {
			distance += float64(legs[i].Distance.Value)
			duration += float64(legs[i].Duration.Value)
		}
	} else {
		// no goroutine for a single leg
		legs[0] = leg(g, waypoints, 0, config)
		distance += float64(legs[0].Distance.Value)
		duration += float64(legs[0].Duration.Value)
	}

	route := Route{
		Distance:      FormatDistance(distance),
		Duration:      FormatDuration(duration),
		StartLocation: legs[0].StartLocation,
		EndLocation:   legs[len(legs)-1].EndLocation,
		Legs:          legs,
	}

	result := &Result{
		BoundingBox: ComputeBounds(route),
		Routes:      []Route{route},
	}
	return result
}

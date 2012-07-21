package route

import (
	"alg"
	"geo"
	"graph"
	"kdtree"
)

func Routes(g graph.Graph, kdt *kdtree.KdTree, waypoints []Point) *Result {
	distance := 0.0
	duration := 0.0
	legs := make([]*Leg, len(waypoints)-1)
	for i := 0; i < len(waypoints)-1; i++ {
		legs[i] = leg(g, kdt, waypoints, i)
		distance += float64(legs[i].Distance.Value)
		duration += float64(legs[i].Duration.Value)
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

func leg(g graph.Graph, kdt *kdtree.KdTree, waypoints []Point, i int) *Leg {
	_, startWays := alg.NearestNeighbor(kdt, waypoints[i][0], waypoints[i][1], true /* forward */)
	_, endWays := alg.NearestNeighbor(kdt, waypoints[i+1][0], waypoints[i+1][1], false /* forward */)
	allequal := true
	oneequal := false
	if len(startWays) != len(endWays) {
		allequal = false
	}
	for _, startPoint := range startWays {
		existequal := false
		for _, endPoint := range endWays {
			existequal = existequal || (startPoint.Node == endPoint.Node)
		}
		oneequal = oneequal || existequal
		allequal = allequal && existequal
	}
	// Start and Endpoint lie on the same edge
	if allequal {
		// Start node == End node
		if len(startWays) == 1 {
			polyline := make([]Point, 1)
			startpoint := Point{startWays[0].Target.Lat, startWays[0].Target.Lng}
			polyline[0] = startpoint
			instruction := "Stay where you are" // Mockup describtion
			step := Step{FormatDistance(0), MockupDuration(0), startpoint, startpoint, polyline, instruction}
			steps := make([]Step, 1)
			steps[0] = step
			return &Leg{FormatDistance(0), MockupDuration(0), startpoint, startpoint, steps}
		} else { // Start and End node are on the same edge
			var correctStartWay, correctEndWay graph.Way
		S:
			for _, startPoint := range startWays {
				for _, endPoint := range endWays {
					if startPoint.Node == endPoint.Node && (startPoint.Length-endPoint.Length) > 0 {
						correctStartWay = startPoint
						correctEndWay = endPoint
						break S
					}
				}
			}
			polyline := make([]graph.Step, 0)
			// Find the steps from start to endpoint
			startsteps := correctStartWay.Steps
			if len(startsteps) > 0 && len(correctEndWay.Steps) >= 0 {
				for i := 0; startsteps[i] != correctEndWay.Steps[len(correctEndWay.Steps)-1]; i++ {
					polyline = append(polyline, startsteps[i])
				}
			} else {
				// TODO no route was found
				// It is fine to output an empty polyline at the moment
			}
			stepDistance := g.WayLength(polyline)
			step := PartwayToStep(polyline, correctStartWay.Target, correctEndWay.Target, stepDistance)
			steps := make([]Step, 1)
			steps[0] = step
			return &Leg{step.Distance, step.Duration, step.StartLocation, step.EndLocation, steps}
		}
	} else if oneequal {
		if len(startWays) == 1 { // If the end node is on the edge outgoing from s
			var correctEndWay graph.Way
			for _, i := range endWays {
				if i.Node == startWays[0].Node {
					correctEndWay = i
					break
				}
			}
			n := len(correctEndWay.Steps)
			polyline := make([]graph.Step, n)
			for i, item := range correctEndWay.Steps {
				polyline[n-i-1] = item
			}
			step := PartwayToStep(polyline, startWays[0].Target, correctEndWay.Target, correctEndWay.Length)
			steps := make([]Step, 1)
			steps[0] = step
			return &Leg{step.Distance, step.Duration, step.StartLocation, step.EndLocation, steps}
		} else if len(endWays) == 1 { // If the start node is on the edge outgoint from e
			var correctStartWay graph.Way
			for _, i := range startWays {
				if i.Node == endWays[0].Node {
					correctStartWay = i
					break
				}
			}
			step := PartwayToStep(correctStartWay.Steps, correctStartWay.Target, endWays[0].Target, correctStartWay.Length)
			steps := make([]Step, 1)
			steps[0] = step
			return &Leg{step.Distance, step.Duration, step.StartLocation, step.EndLocation, steps}
		} else { // we have s->u->e so they are on adjacent edges.
			var correctStartWay, correctEndWay graph.Way
			for _, i := range startWays {
				for _, j := range endWays {
					if i.Node == j.Node {
						correctStartWay = i
						correctEndWay = j
					}
				}
			}
			step1 := PartwayToStep(correctStartWay.Steps, correctStartWay.Target, NodeToStep(g, correctStartWay.Node),
				correctStartWay.Length)
			step2 := PartwayToStep(correctEndWay.Steps, NodeToStep(g, correctEndWay.Node), correctEndWay.Target,
				correctEndWay.Length)
			steps := make([]Step, 2)
			steps[0] = step1
			steps[1] = step2
			legDistance := correctStartWay.Length + correctEndWay.Length
			return &Leg{FormatDistance(legDistance), MockupDuration(legDistance), step1.StartLocation, step2.EndLocation, steps}
		}
	}

	// Use the Dijkatrs version using a large slice only for long routes where the map of the
	// other version can get quite large
	if getDistance(g, startWays[0].Node, endWays[0].Node) > 100.0*1000.0 { // > 100km
		dist, vertices, edges, start, end := alg.DijkstraSlice(g, startWays, endWays)
		return PathToLeg(g, dist, vertices, edges, start, end)
	}
	dist, vertices, edges, start, end := alg.Dijkstra(g, startWays, endWays)
	return PathToLeg(g, dist, vertices, edges, start, end)
}

// getDistance returns the distance between the two given nodes
func getDistance(g graph.Graph, n1 graph.Node, n2 graph.Node) float64 {
	lat1, lng1 := g.NodeLatLng(n1)
	lat2, lng2 := g.NodeLatLng(n2)
	return geo.Coordinate{Lat: lat1, Lng: lng1}.Distance(geo.Coordinate{Lat: lat2, Lng: lng2})
}

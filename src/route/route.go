package route

import (
	"geo"
	"graph"
	"kdtree"
)

func Routes(g *graph.ClusterGraph, kdt *kdtree.KdTree, waypoints []Point, m graph.Metric, trans graph.Transport) *Result {
	distance := 0.0
	duration := 0.0
	legs := make([]*Leg, len(waypoints)-1)
	for i := 0; i < len(waypoints)-1; i++ {
		legs[i] = leg(g, kdt, waypoints, i, m, trans)
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

func leg(g *graph.ClusterGraph, kdt *kdtree.KdTree, waypoints []Point, i int, m graph.Metric, trans graph.Transport) *Leg {
	//_, startWays := alg.NearestNeighbor(kdt, waypoints[i][0], waypoints[i][1], true /* forward */)
	//_, endWays := alg.NearestNeighbor(kdt, waypoints[i+1][0], waypoints[i+1][1], false /* forward */)
	startWays := make([]graph.Way, 0)
	endWays := make([]graph.Way, 0)
	startCluster := 0 //TODO kdt should return cluster index of the start vertex
	endCluster := 0   //TODO kdt should return cluster index of the target vertex

	allequal := true
	oneequal := false
	if len(startWays) != len(endWays) {
		allequal = false
	}
	for _, startPoint := range startWays {
		existequal := false
		for _, endPoint := range endWays {
			existequal = existequal || (startPoint.Vertex == endPoint.Vertex)
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
					if startPoint.Vertex == endPoint.Vertex && (startPoint.Length-endPoint.Length) > 0 {
						correctStartWay = startPoint
						correctEndWay = endPoint
						break S
					}
				}
			}
			polyline := make([]geo.Coordinate, 0)
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
			stepDistance := geo.StepLength(polyline)
			step := PartwayToStep(polyline, correctStartWay.Target, correctEndWay.Target, stepDistance)
			steps := make([]Step, 1)
			steps[0] = step
			return &Leg{step.Distance, step.Duration, step.StartLocation, step.EndLocation, steps}
		}
	} else if oneequal {
		if len(startWays) == 1 { // If the end node is on the edge outgoing from s
			var correctEndWay graph.Way
			for _, i := range endWays {
				if i.Vertex == startWays[0].Vertex {
					correctEndWay = i
					break
				}
			}
			n := len(correctEndWay.Steps)
			polyline := make([]geo.Coordinate, n)
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
				if i.Vertex == endWays[0].Vertex {
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
					if i.Vertex == j.Vertex {
						correctStartWay = i
						correctEndWay = j
					}
				}
			}
			step1 := PartwayToStep(correctStartWay.Steps, correctStartWay.Target, g.Cluster[startCluster].VertexCoordinate(correctStartWay.Vertex),
				correctStartWay.Length)
			step2 := PartwayToStep(correctEndWay.Steps, g.Cluster[endCluster].VertexCoordinate(correctEndWay.Vertex), correctEndWay.Target,
				correctEndWay.Length)
			steps := make([]Step, 2)
			steps[0] = step1
			steps[1] = step2
			legDistance := correctStartWay.Length + correctEndWay.Length
			return &Leg{FormatDistance(legDistance), MockupDuration(legDistance), step1.StartLocation, step2.EndLocation, steps}
		}
	}

	// Both are in the same Cluster
	if startCluster == endCluster {
		startElements := make([]*Element, len(startWays))
		endElements := make([]*Element, len(endWays))
		for i, n := range startWays {
			e := NewElement(n.Vertex, n.Length)
			startElements[i] = e
		}
		for i, n := range endWays {
			e := NewElement(n.Vertex, n.Length)
			endElements[i] = e
		}
		distance, vertices, edges := DijkstraStarter(g.Cluster[startCluster], startElements, endElements, m, trans)
		indexstart := -1
		for i, n := range startWays {
			if vertices[0] == n.Vertex {
				indexstart = i
				break
			}
		}
		indexend := -1
		for i, n := range endWays {
			if vertices[len(vertices)-1] == n.Vertex {
				indexend = i
				break
			}
		}
		return PathToLeg(g.Cluster[startCluster], distance, vertices, edges, &startWays[indexstart], &endWays[indexend])
	} else { //They are in different Clusters
		startelms := []*Element(nil)
		endelms := []*Element(nil)
		// TODO check if start/end vertex is boundary note
		c := make(chan int, 2)
		go func() {
			startelms = DijkstraComplete(g.Cluster[startCluster], startWays, m, trans, true)
			c <- 0
		}()
		go func() {
			endelms = DijkstraComplete(g.Cluster[endCluster], endWays, m, trans, false)
			c <- 1
		}()
		<-c
		<-c
		startboundary := make([]*Element, g.Overlay.ClusterSize(startCluster))
		endboundary := make([]*Element, g.Overlay.ClusterSize(endCluster))
		reachable := false
		for i := 0; i < g.Overlay.ClusterSize(startCluster); i++ {
			reachable = reachable || startelms[i] != nil
			e := NewElement(g.Overlay.ClusterVertex(startCluster, startelms[i].vertex), startelms[i].priority)
			startboundary[i] = e
		}
		if !reachable { // Cannot reach the boundary from this vertex
			return nil
		}
		reachable = false
		for i := 0; i < g.Overlay.ClusterSize(endCluster); i++ {
			reachable = reachable || endelms[i] != nil
			e := NewElement(g.Overlay.ClusterVertex(endCluster, endelms[i].vertex), endelms[i].priority)
			endboundary[i] = e
		}
		if !reachable { // Cannot reach the boundary from this vertex
			return nil
		}
		distance, vertices, edges := DijkstraStarter(g.Overlay, startboundary, endboundary, m, trans)
		// No path found
		if vertices == nil {
			return nil
		}
		crossvertices := make([]int, 1)
		for i := 0; i < len(vertices)-1; i++ {
			c1, _ := g.Overlay.VertexCluster(vertices[i])
			c2, _ := g.Overlay.VertexCluster(vertices[i+1])
			if c1 == c2 {
				crossvertices = append(crossvertices, i)
			}
		}
		tmplegs := make([]*Leg, len(crossvertices))
		if len(crossvertices) == 1 {
		}
		_ = tmplegs
		_ = distance
		_ = edges
	}
	return nil

	// Use the Dijkatrs version using a large slice only for long routes where the map of the
	// other version can get quite large
	/*if getDistance(g, startWays[0].Vertex, endWays[0].Vertex) > 100.0*1000.0 { // > 100km		dist, vertices, edges, start, end := alg.DijkstraSlice(g, startWays, endWays)
		return PathToLeg(g, dist, vertices, edges, start, end)
	}
	dist, vertices, edges, start, end := alg.Dijkstra(g, startWays, endWays)
	return PathToLeg(g, dist, vertices, edges, start, end)*/
}

// getDistance returns the distance between the two given nodes
func getDistance(g graph.Graph, v1 graph.Vertex, v2 graph.Vertex) float64 {
	return g.VertexCoordinate(v1).Distance(g.VertexCoordinate(v2))
}

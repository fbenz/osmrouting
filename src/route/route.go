package route

import (
	"fmt"
	"geo"
	"graph"
	"kdtree"
	"math"
)

func Routes(g *graph.ClusterGraph, waypoints []Point, m graph.Metric, trans graph.Transport) *Result {
	distance := 0.0
	duration := 0.0
	legs := make([]*Leg, len(waypoints)-1)
	for i := 0; i < len(waypoints)-1; i++ {
		legs[i] = leg(g, waypoints, i, m, trans)
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

func leg(g *graph.ClusterGraph, waypoints []Point, i int, m graph.Metric, trans graph.Transport) *Leg {
	startCoord := geo.Coordinate{Lat: waypoints[i][0], Lng: waypoints[i][1]}
	endCoord := geo.Coordinate{Lat: waypoints[i+1][0], Lng: waypoints[i+1][1]}
	startCluster, startWays := kdtree.NearestNeighbor(startCoord, true /* forward */, trans)
	endCluster, endWays := kdtree.NearestNeighbor(endCoord, false /* forward */, trans)
	
	if ok,l := edgeLeg(g, startWays, endWays, startCluster, endCluster); ok {
		return l
	}

	// Both are in the same Cluster
	if startCluster == endCluster && startCluster != -1 {
		return sameCluster(g, startWays, endWays, startCluster, trans, m)
	} else { //They are in different Clusters or on the overlay graph
		startRunner := Router{Forward: true, Transport: trans, Metric: m}
		endRunner := Router{Forward: false, Transport: trans, Metric: m}
		overlayRunner := BidiRouter{Transport: trans, Metric: m}
		overlayRunner.Reset(g.Overlay)
		// TODO check if start/end vertex is boundary note
		switch {
		case startCluster == -1 && endCluster == -1:
			for _, s := range startWays {
				overlayRunner.AddSource(s.Vertex, float32(s.Length)) // TODO remove cast
			}
			for _, t := range endWays {
				overlayRunner.AddTarget(t.Vertex, float32(t.Length))
			}
		case startCluster == -1 && endCluster != -1:
			for _, s := range startWays {
				overlayRunner.AddSource(s.Vertex, float32(s.Length)) // TODO remove cast
			}
			endRunner.Reset(g.Cluster[endCluster])
			for _, e := range endWays {
				endRunner.AddSource(e.Vertex, float32(e.Length)) // TODO remove cast
			}
			endRunner.Run()
			reachable := false
			for i := 0; i < g.Overlay.ClusterSize(endCluster); i++ {
				v := graph.Vertex(i)
				if endRunner.Reachable(v) {
					reachable = true
					overlayRunner.AddTarget(g.Overlay.ClusterVertex(endCluster, v), endRunner.Distance(v))
				}
			}
			if !reachable {
				panic("No boundary vertices can reach the target.")
			}
		case startCluster != -1 && endCluster == -1:
			startRunner.Reset(g.Cluster[startCluster])
			for _, t := range endWays {
				overlayRunner.AddTarget(t.Vertex, float32(t.Length))
			}
			for _, e := range startWays {
				startRunner.AddSource(e.Vertex, float32(e.Length)) // TODO remove cast
			}
			startRunner.Run()
			reachable := false
			for i := 0; i < g.Overlay.ClusterSize(startCluster); i++ {
				v := graph.Vertex(i)
				if startRunner.Reachable(v) {
					reachable = true
					overlayRunner.AddSource(g.Overlay.ClusterVertex(endCluster, v), endRunner.Distance(v))
				}
			}
			if !reachable {
				panic("No boundary vertices are reachable from the source.")
			}
		case startCluster != -1 && endCluster != -1:
			startRunner.Reset(g.Cluster[startCluster])
			endRunner.Reset(g.Cluster[endCluster])
			for _, e := range startWays {
				startRunner.AddSource(e.Vertex, float32(e.Length)) // TODO remove cast
			}
			for _, e := range endWays {
				endRunner.AddSource(e.Vertex, float32(e.Length)) // TODO remove cast
			}
			c := make(chan int, 2)
			go func() {
				startRunner.Run()
				c <- 1
			}()
			go func() {
				endRunner.Run()
				c <- 1
			}()
			<-c
			<-c
			reachable := false
			for i := 0; i < g.Overlay.ClusterSize(startCluster); i++ {
				v := graph.Vertex(i)
				if startRunner.Reachable(v) {
					reachable = true
					overlayRunner.AddSource(g.Overlay.ClusterVertex(startCluster, v), startRunner.Distance(v))
				}
			}
			if !reachable {
				panic("No boundary vertices are reachable from the source.")
			}
			reachable = false
			for i := 0; i < g.Overlay.ClusterSize(endCluster); i++ {
				v := graph.Vertex(i)
				if endRunner.Reachable(v) {
					reachable = true
					overlayRunner.AddTarget(g.Overlay.ClusterVertex(endCluster, v), endRunner.Distance(v))
				}
			}
			if !reachable {
				fmt.Printf(" * endCluster: %v\n", endCluster)
				fmt.Printf(" * endClusterSize: %v\n", g.Overlay.ClusterSize(endCluster))
				fmt.Printf(" * endWays: %v\n", endWays)
				panic("No boundary vertices can reach the target.")
			}
		}

		overlayRunner.Run()
		vertices := overlayRunner.VPath()

		// No path found
		if math.IsInf(float64(overlayRunner.Distance()), 0)  {
			panic("Overlay runner found no path.")
		}

		crossvertices := make([]int,0)
		for i := 0; i < len(vertices)-1; i++ {
			c1, _ := g.Overlay.VertexCluster(vertices[i])
			c2, _ := g.Overlay.VertexCluster(vertices[i+1])
			if c1 == c2 {
				crossvertices = append(crossvertices, i)
			}
		}

		tmplegs := []*Leg(nil)
		switch len(crossvertices) {
		case 0: // No intermediate routes
		case 1: // Just two intermediate routes, don't create a goroutine
			tmplegs = make([]*Leg, len(crossvertices))
			cluster, svertex := g.Overlay.VertexCluster(vertices[crossvertices[0]])
			_, evertex := g.Overlay.VertexCluster(vertices[crossvertices[0]+1])
			router := BidiRouter{Transport: trans, Metric: m}
			router.Reset(g.Cluster[cluster])
			router.AddSource(svertex, 0)
			router.AddTarget(evertex, 0)
			router.Run()
			path, edpath := router.Path()
			dist := float64(router.Distance()) // TODO remove cast
			tmplegs[0] = PathToLeg(g.Cluster[cluster], dist, path, edpath, nil, nil)
		default: // More than two intermediate results
			tmplegs = make([]*Leg, len(crossvertices))
			c := make(chan int, len(crossvertices))
			for i := 0; i < len(crossvertices); i++ {
				go func(j int) {
					cluster, svertex := g.Overlay.VertexCluster(vertices[crossvertices[j]])
					_, evertex := g.Overlay.VertexCluster(vertices[crossvertices[j]+1])

					router := BidiRouter{Transport: trans, Metric: m}
					router.Reset(g.Cluster[cluster])
					router.AddSource(svertex, 0)
					router.AddTarget(evertex, 0)
					router.Run()
					path, edpath := router.Path()
					dist := float64(router.Distance()) // TODO remove cast
					tmplegs[j] = PathToLeg(g.Cluster[cluster], dist, path, edpath, nil, nil)
					c <- j
				}(i)
			}
			// Wait till everyone is finished
			for i := 0; i < len(crossvertices); i++ {
				<-c
			}
		}
		// Compute the start leg
		startLeg := (*Leg)(nil)
		if startCluster == -1 { // Start is on overlay graph
			if len(startWays) == 1 { // Exactly one startvertex on the overlay graph
				startLeg = WayToLeg(&startWays[0], g.Overlay, true, vertices[0])
			} else { // The start is on an edge between two overlay graphs
				for _, w := range startWays {
					if w.Vertex == vertices[0] {
						startLeg = WayToLeg(&w, g.Overlay, true, vertices[0])
						break
					}
				}
			}
		} else {
			_, endvertex := g.Overlay.VertexCluster(vertices[0])
			path, pathedges := startRunner.Path(endvertex)
			startWay := (*graph.Way)(nil)
			for _, w := range startWays {
				if w.Vertex == endvertex {
					startWay = &w
					break
				}
			}
			startLeg = PathToLeg(g.Cluster[startCluster], float64(startRunner.Distance(endvertex)), path, pathedges, startWay, nil) // TODO remove cast
		}

		// Compute the end leg
		endLeg := (*Leg)(nil)
		if endCluster == -1 { // End is on overlay graph
			if len(endWays) == 1 { // Exactly one startvertex on the overlay graph
				endLeg = WayToLeg(&endWays[0], g.Overlay, false, vertices[len(vertices)-1])
			} else { // The start is on an edge between two overlay graphs
				for _, w := range endWays {
					if w.Vertex == vertices[len(vertices)-1] {
						endLeg = WayToLeg(&w, g.Overlay, false, vertices[len(vertices)-1])
						break
					}
				}
			}
		} else {
			_, endvertex := g.Overlay.VertexCluster(vertices[len(vertices)-1])
			path, pathedges := endRunner.Path(endvertex)
			endWay := (*graph.Way)(nil)
			for _, w := range endWays {
				if w.Vertex == endvertex {
					endWay = &w
					break
				}
			}
			endLeg = PathToLeg(g.Cluster[endCluster], float64(endRunner.Distance(endvertex)), path, pathedges, nil, endWay) // TODO remove cast
		}

		// Put the route together
		leg := startLeg
		if tmplegs == nil {// No intermediate cluster
			for i := 0; i < len(vertices)-1; i++ {
				edge := overlayRunner.EdgeBetween(vertices[i], vertices[i+1])
				step := EdgeToStep(g.Overlay,edge,vertices[i],vertices[i+1])
				leg = AppendStep(leg, &step)
			}
		} else {
			for i, j := 0, 0; i < len(vertices)-1; i++ {
				if tmplegs != nil && j < len(crossvertices) && crossvertices[j] == i { // The path crosses a cluster
					leg = CombineLegs(leg, tmplegs[j])
					j++
				} else { // It is just an edge
					edge := overlayRunner.EdgeBetween(vertices[i], vertices[i+1])
					step := EdgeToStep(g.Overlay, edge, vertices[i], vertices[i+1])
					leg = AppendStep(leg, &step)
				}
			}
		}
		leg = CombineLegs(leg, endLeg)

		return leg
	}
	panic("And in the very end, panic.")
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

// Compute the Leg, when start and endvertex share an edge
func edgeLeg(g *graph.ClusterGraph, startWays, endWays []graph.Way, startCluster, endCluster int) (bool, *Leg) {
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
			return true,&Leg{FormatDistance(0), MockupDuration(0), startpoint, startpoint, steps}
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
			return true,&Leg{step.Distance, step.Duration, step.StartLocation, step.EndLocation, steps}
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
			return true,&Leg{step.Distance, step.Duration, step.StartLocation, step.EndLocation, steps}
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
			return true, &Leg{step.Distance, step.Duration, step.StartLocation, step.EndLocation, steps}
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
			return true,&Leg{FormatDistance(legDistance), MockupDuration(legDistance), step1.StartLocation, step2.EndLocation, steps}
		}
	}
	return false,nil
}

func sameCluster(g *graph.ClusterGraph, startWays, endWays []graph.Way, cluster int, trans graph.Transport, m graph.Metric) *Leg {
	router := BidiRouter{Transport: trans, Metric: m}
	router.Reset(g.Cluster[cluster])
	for _, n := range startWays {
		router.AddSource(n.Vertex, float32(n.Length)) // TODO remove cast
	}
	for _, n := range endWays {
		router.AddTarget(n.Vertex, float32(n.Length)) // TODO remove cast
	}
	router.Run()
	vertices, edges := router.Path()
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
	if indexstart == -1 || indexend == -1 {
		panic("Did not find a path between two points in the same cluster.")
	}	
	return PathToLeg(g.Cluster[cluster], float64(router.Distance()), vertices, edges, &startWays[indexstart], &endWays[indexend]) // TODO remove cast
}

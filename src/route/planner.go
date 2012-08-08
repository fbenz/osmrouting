
package route

import (
	"geo"
	"graph"
	"kdtree"
	"math"
)

type RoutePlanner struct {
	// Underlying graph structure
	Graph           *graph.ClusterGraph
	// User input
	Waypoints       []geo.Coordinate
	// Graph setting
	Transport       graph.Transport
	Metric          graph.Metric
	AvoidFerries    bool
	// Planner options
	ConcurrentKd    bool
	ConcurrentLegs  bool
	ConcurrentPaths bool
	// KdTree Output
	Locations       []kdtree.Location
}

// Execute f(0), f(1), ..., f(n-1) and do so in parallel, based on the
// corresponding flag.
func Multiplex(n int, inParallel bool, f func (int)) {
	if inParallel && n > 1 {
		barrier := make(chan int, n)
		for i := 0; i < n; i++ {
			go func(i int) {
				f(i)
				barrier <- 1
			}(i)
		}
		for i := 0; i < n; i++ {
			<-barrier
		}
	} else {
		for i := 0; i < n; i++ {
			f(i)
		}
	}
}

func (r *RoutePlanner) Run() *Result {
	// Compute the closest point in the graph for each user specified waypoint.
	count := len(r.Waypoints)
	r.Locations = make([]kdtree.Location, count)
	Multiplex(count, r.ConcurrentKd, func (i int) {
		r.Locations[i] = kdtree.NearestNeighbor(r.Waypoints[i], r.Transport)
	})
	
	// Now compute a shortest path for each leg.
	// If ConcurrentLegs is set compute the legs concurrently.
	legs := make([]Leg, count-1)
	Multiplex(count-1, r.ConcurrentLegs, func (i int) {
		legs[i] = r.ComputeLeg(i)
	})

	// Format the results.
	distance := 0
	duration := 0
	for _, leg := range legs {
		distance += leg.Distance.Value
		duration += leg.Duration.Value
	}

	route := Route{
		Distance:      FormatDistance(float64(distance)),
		Duration:      FormatDuration(float64(duration)),
		StartLocation: legs[0].StartLocation,
		EndLocation:   legs[len(legs)-1].EndLocation,
		Legs:          legs,
	}

	return &Result{
		BoundingBox: ComputeBounds(route),
		Routes:      []Route{route},
	}
}

func (r *RoutePlanner) UnionGraph(src, dst kdtree.Location) (*graph.UnionGraph, int, int) {
	overlay := r.Graph.Overlay
	cluster := []*graph.GraphFile(nil)
	indices := []int(nil)
	srcCluster := -1
	dstCluster := -1
	
	if src.Cluster != -1 {
		cluster = append(cluster, r.Graph.Cluster[src.Cluster])
		indices = append(indices, src.Cluster)
		srcCluster = 0
	}
	
	if dst.Cluster != -1 && src.Cluster != dst.Cluster {
		cluster = append(cluster, r.Graph.Cluster[dst.Cluster])
		indices = append(indices, dst.Cluster)
		dstCluster = srcCluster + 1
	} else if src.Cluster == dst.Cluster {
		dstCluster = srcCluster
	}
	
	g := graph.NewUnionGraph(overlay, cluster, indices)
 	return g, srcCluster, dstCluster
}

// Convenience function to find a forward edge (of minimum weight) from
// vertex u to vertex v. Returns -1 if no edge was found.
func (r *RoutePlanner) EdgeBetween(g graph.Graph, u, v graph.Vertex) graph.Edge {
	minEdge   := graph.Edge(-1)
	minWeight := math.Inf(1)
	for _, e := range g.VertexEdges(u, true, r.Transport, nil) {
		n := g.EdgeOpposite(e, u)
		if n != v {
			continue
		}
		weight := g.EdgeWeight(e, r.Transport, r.Metric)
		if weight < minWeight {
			minEdge = e
			minWeight = weight
		}
	}
	return minEdge
}

// Compute one path segment between location[waypointIndex] and location[waypointIndex+1]
func (r *RoutePlanner) ComputeLeg(waypointIndex int) Leg {
	src := r.Locations[waypointIndex]
	dst := r.Locations[waypointIndex+1]
	buf := []geo.Coordinate(nil)
	srcWays := src.Decode(true  /* forward */, r.Transport, &buf)
	dstWays := dst.Decode(false /* forward */, r.Transport, &buf)
	
	// Compute the union of the source and target clusters.
	g, srcCluster, dstCluster := r.UnionGraph(src, dst)
	
	// Run Dijkstra on the union graph
	router := &BidiRouter{
		Transport: r.Transport,
		Metric:    r.Metric,
	}
	router.Reset(g)
	for _, srcWay := range srcWays {
		v := g.ToUnionVertex(srcWay.Vertex, srcCluster)
		router.AddSource(v, float32(srcWay.Length))
	}
	for _, dstWay := range dstWays {
		v := g.ToUnionVertex(dstWay.Vertex, dstCluster)
		router.AddTarget(v, float32(dstWay.Length))
	}
	router.Run()
	
	// Gather the result path.
	vpath    := router.VPath()
	segments := [][]Step(nil)
	sketches := []int(nil)
	indices  := []int(nil)
	i        := 0
	for i < len(vpath)-1 {
		u, v   := vpath[i], vpath[i+1]
		uindex := g.VertexToCluster(u)
		vindex := g.VertexToCluster(v)
		steps  := []Step(nil)
		
		if uindex == -1 && vindex == -1 {
			// This might be a shortcut edge, or it might just be an edge on
			// the overlay graph. For simplicity we always treat this as a single
			// step.
			overlay := r.Graph.Overlay
			e := r.EdgeBetween(overlay, u, v)
			if int(e) == -1 {
				// Shortcut edge, we will have to elaborate it later.
				sketches = append(sketches, len(segments))
				indices  = append(indices, i)
			} else {
				// Cut edge
				steps = append(steps, r.EdgeToStep(overlay, e, u, v))
			}
			i++
		} else {
			// A path within a cluster. We collect all edges until we hit the
			// next boundary vertex.
			u, cluster := g.ToClusterVertex(u, uindex)
			j, done := i + 1, false
			for !done && j < len(vpath) {
				v := vpath[j]
				
				// Project the vertex v onto the current cluster.
				vindex := g.VertexToCluster(v)
				if vindex == -1 {
					done = true
					_, v = g.Overlay.VertexCluster(v)
				} else {
					v, _ = g.ToClusterVertex(v, vindex)
					j++
				}
				
				// Find the matching u - v edge in the current cluster
				e := r.EdgeBetween(cluster, u, v)
				steps = append(steps, r.EdgeToStep(cluster, e, u, v))
				u = v
			}
			i = j
		}
		
		segments = append(segments, steps)
	}
	
	// Elaborate the result path
	Multiplex(len(sketches), r.ConcurrentPaths, func (i int) {
		// Find the boundary vertices and cluster corresponding to this shortcut.
		index := indices[i]
		clusterIndex, u := g.Overlay.VertexCluster(vpath[index])
		cluster := r.Graph.Cluster[clusterIndex]
		_, v := g.Overlay.VertexCluster(vpath[index+1])
		
		// Run Dijkstra to find a u -> v path.
		router := &BidiRouter{
			Transport: r.Transport,
			Metric:    r.Metric,
		}
		router.Reset(cluster)
		router.AddSource(u, 0)
		router.AddTarget(v, 0)
		router.Run()
		
		// Convert this path into a step array.
		vertices, edges := router.Path()
		steps := make([]Step, len(edges))
		for j, edge := range edges {
			s := vertices[j]
			t := vertices[j+1]
			steps[j] = r.EdgeToStep(cluster, edge, s, t)
		}
		segments[sketches[i]] = steps
	})
	
	// Build Leg
	indexstart, indexend := -1, -1
	var startc, stopc geo.Coordinate
	for i, srcWay := range srcWays {
		vertex := g.ToUnionVertex(srcWay.Vertex, srcCluster)
		if vpath[0] == vertex {
			indexstart = i
			startc = src.Graph.VertexCoordinate(srcWay.Vertex)
			break
		}
	}
	for i, dstWay := range dstWays {
		vertex := g.ToUnionVertex(dstWay.Vertex, dstCluster)
		if vpath[len(vpath)-1] == vertex {
			indexend = i
			stopc = dst.Graph.VertexCoordinate(dstWay.Vertex)
			break
		}
	}
	steps := []Step(nil)
	for _, segment := range segments {
		steps = append(steps, segment...)
	}
	return r.StepsToLeg(steps, srcWays[indexstart], dstWays[indexend], startc, stopc)
}

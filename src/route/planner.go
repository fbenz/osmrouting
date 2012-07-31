
package route

import (
	"geo"
	"graph"
	"kdtree"
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
	if inParallel && n > 0 {
		barrier := make(chan int, n)
		for i := 0; i < n; i++ {
			go func() {
				f(i)
				barrier <- 1
			}
		}
		for i := 0; i < count; i++ {
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
	Multiplex(count, r.ConcurrentKd, func (i int) {
		r.Locations[i] = kdtree.NearestNeighbor(r.Waypoints[i], r.Transport)
	})
	
	// Now compute a shortest path for each leg.
	// If ConcurrentLegs is set compute the legs concurrently.
	legs  := make([]Leg, count-1)
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
		Distance:      FormatDistance(distance),
		Duration:      FormatDuration(duration),
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
	overlay := r.Graph.OverlayGraph
	cluster := []*graph.GraphFile(nil)
	srcCluster := -1
	dstCluster := -1
	
	if a.Cluster != -1 {
		cluster = append(cluster, r.Graph.Cluster[src.Cluster])
		srcCluster = 0
	}
	
	if b.Cluster != -1 && b.Cluster != a.Cluster {
		cluster = append(cluster, r.Graph.Cluster[dst.Cluster])
		dstCluster = srcCluster + 1
	}
	
	g := graph.NewUnionGraph(overlay, cluster)
 	return g, srcCluster, dstCluster
}

// Compute one path segment between location[i] and location[i+1]
func (r *RoutePlanner) ComputeLeg(i int) Leg {
	src := r.Locations[i]
	dst := r.Locations[i+1]
	srcWays := src.Decode(true  /* forward */, r.Transport)
	dstWays := dst.Decode(false /* forward */, r.Transport)
	
	// Compute the union of the source and target clusters.
	g, srcCluster, dstCluster := r.UnionGraph(src, dst)
	
	// Run Dijkstra on the union graph
	router := &BidiRouter{
		Transport: r.Transport,
		Metric:    r.Metric,
	}
	router.Reset(g)
	for _, srcWay := range srcWays {
		v := g.ClusterVertex(srcWay.Vertex, srcCluster)
		router.AddSource(v, srcWay.Length)
	}
	for _, dstWay := range dstWays {
		v := g.ClusterVertex(dstWay.Vertex, dstCluster)
		router.AddTarget(v, dstWay.Length)
	}
	router.Run()
	
	// Elaborate the result path.
	vpath := router.VPath()
	segments := []Step(nil)
	sketches := []int(nil)
	i, prev  := 0, -1
	for i < len(vpath)-1 {
		u, j   := vpath[i], i+1
		uindex := g.VertexIndex(u)
		step   := Step{}
		for j < len(vpath) {
			v      := vpath[j]
			vindex := g.VertexIndex(v)
			if uindex == -1 && vindex == -1 {
				// Shortcut step
				sketches = append(sketches, len(segments))
				segments = append(segments, step)
				break
			}
			// TODO: Add a normal step
			j++
		}
		i = j
	}
	
	// TODO: build leg
}

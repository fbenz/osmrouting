
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
	ConcurrentPaths bool
	ConcurrentLegs  bool
	// KdTree Output
	Locations       []kdtree.Location
}

// Translate between waypoints and graph locations.
// This runs concurrently, based on the ConcurrentKd flag.
func (r *RoutePlanner) NearestNeighbors() {
	count := len(r.Waypoints)
	if r.ConcurrentKd {
		barrier := make(chan int, count)
		for i := 0; i < count; i++ {
			go func() {
				query := r.Waypoints[i]
				r.Locations[i] = kdtree.NearestNeighbor(query, r.Transport)
				barrier <- 1
			}()
		}
		for i := 0; i < count; i++ {
			<-barrier
		}
	} else {
		for i := 0; i < count; i++ {
			query := r.Waypoints[i]
			r.Locations[i] = kdtree.NearestNeighbor(query, r.Transport)
		}
	}
}

func (r *RoutePlanner) Run() *Result {
	// Compute the closest point in the graph for each user specified waypoint.
	r.NearestNeighbors()
	
	// Now compute a shortest path for each leg.
	// If ConcurrentLegs is set compute the legs concurrently.
	count := len(r.Waypoints)
	legs  := make([]*Leg, count-1)
	if count > 2 && r.ConcurrentLegs {
		barrier := make(chan int, count-1)
		for i := 0; i < count-1; i++ {
			go func() {
				legs[i] = r.ComputeLeg(i)
				barrier <- 1
			}()
		}
		for i := 0; i < count-1; i++ {
			<-barrier
		}
	} else {
		for i := 0; i < count-1; i++ {
			legs[i] = r.ComputeLeg(i)
		}
	}

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

func (r *RoutePlanner) UnionGraph(a, b kdtree.Location) *graph.UnionGraph {
	overlay := r.Graph.OverlayGraph
	cluster := []*graph.GraphFile(nil)
	if a.Cluster != -1 {
		cluster = append(cluster, r.Graph.Cluster[a.Cluster])
	}
	if b.Cluster != -1 && b.Cluster != a.Cluster {
		cluster = append(cluster, r.Graph.Cluster[b.Cluster])
	}
 	return graph.NewUnionGraph(overlay, cluster)
}

// Compute one path segment between location[i] and location[i+1]
func (r *RoutePlanner) ComputeLeg(i int) *Leg {
	src := r.Locations[i]
	dst := r.Locations[i+1]
	srcWays := src.Decode(true  /* forward */, r.Transport)
	dstWays := dst.Decode(false /* forward */, r.Transport)
	
	// Cluster indices in the union graph
	srcCluster := -1
	dstCluster := -1
	if src.Cluster != -1 {
		srcCluster = 0
	}
	if dst.Cluster != -1 && dst.Cluster != src.Cluster {
		dstCluster = srcCluster + 1
	}
	
	// Compute the union of the source and target clusters.
	g := r.UnionGraph(src, dst)
	router := &BidiRouter{
		Transport: r.Transport,
		Metric:    r.Metric,
	}
	router.Reset(g)
	
	// Add the source and target vertices
	for _, srcWay := range srcWays {
		v := g.ClusterVertex(srcWay.Vertex, srcCluster)
		router.AddSource(v, srcWay.Length)
	}
	for _, dstWay := range dstWays {
		v := g.ClusterVertex(dstWay.Vertex, dstCluster)
		router.AddTarget(v, dstWay.Length)
	}
	
	// Run Dijkstra on the union graph. The result is a vertex path over
	// the union path which we have to elaborate later on.
	router.Run()
	vpath := router.VPath()
	steps := make([]*Leg, len(vpath)-1)
	for i := 0; i < len(vpath)-1; i++ {
		u := vpath[i]
		v := vpath[i+1]
		uindex := g.VertexIndex(u)
		vindex := g.VertexIndex(v)
		if uindex == vindex && uindex == -1 {
			// shortcut edge
		} else if uindex == -1 {
			u, t = VertexCluster(u)
		}
	}
}

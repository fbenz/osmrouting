package graph

// Index Mapping:
// Every vertex should exist only once in the graph. We map the overlay
// vertices into the range [0..overlay vertex count). Now for each cluster
// we consider the boundary vertices are already in the graph, since they
// are part of the overlay graph. So let C be some cluster, we add the non
// boundary vertices of C as the next consecutive vertices in the graph.
// Now, for a vertex i we can determine the actual index of i in two steps.
// First if i < Overlay.VertexCount it is an overlay vertex, but may well
// be part of one of the cluster graphs as well. We can call VertexCluster
// to retrieve the cluster and vertex ids. If the id is present in the
// indices array, we may need to consider additional neighbors of v.
// On the other hand, if i >= Overlay.VertexCount, then i is an internal
// vertex of some cluster. We precompute the offsets for each cluster
// and determine the correct graph by linear search (the number of clusters
// is very small). Then the index within the cluster is simply i + ClusterSize.

// There should have been a BasicGraph interface which is sufficient for
// Dijkstra and nothing else... but alas, there is no time for such changes.

type UnionGraph struct {
	Overlay *OverlayGraphFile
	Cluster []*GraphFile
	Indices []int
	Offsets []int
	Size    int
}

func NewUnionGraph(overlay *OverlayGraphFile, cluster []*GraphFile, indices []int) *UnionGraph {
	size := overlay.VertexCount()
	offsets := make([]int, len(cluster))
	for i, g := range cluster {
		offsets[i] = size
		boundary := overlay.ClusterSize(indices[i])
		size += g.VertexCount() - boundary
	}
	return &UnionGraph{
		Overlay: overlay,
		Cluster: clust,
		Indices: indices,
		Offsets: offsets,
		Size:    size,
	}
}

func (g *UnionGraph) VertexCount() int {
	return g.Size
}

func (g *UnionGraph) VertexIndex(v Vertex) int {
	i := 0
	for i < len(g.Cluster) {
		if v < g.Cluster[i] {
			break
		}
		i++
	}
	return i
}

// Given a Vertex with index -1 returns the corresponding cluster vertex, or nil.
func (g *UnionGraph) VertexCluster(v Vertex) (Vertex, *GraphFile) {
	clusterId, vertexId := g.OverlayGraph.VertexCluster(v)
	for i, id := g.Indices {
		if clusterId == id {
			return vertexId, g.Cluster[i]
		}
	}
	return Vertex(-1), nil
}

func (g *UnionGraph) ClusterVertex(v Vertex, index int) Vertex {
	if index != -1 {
		clusterId := g.Indices[index]
		cluster   := g.Cluster[index]
		offset    := g.OverlayGraph.ClusterSize(clusterId)
		if v > offset {
			// internal vertex
			return v - offset + g.Offsets[index]
		} else {
			// boundary vertex
			return g.OverlayGraph.ClusterVertex(clusterId, v)
		}
	}
	return v
}

func (g *UnionGraph) VertexNeighbors(v Vertex, forward bool, t Transport, m Metric, buf []Dart) []Dart {
	index := g.VertexIndex(v)
	if index == -1 {
		// The vertex is in the overlay graph and we can always add the neighbors in the
		// overlay graph.
		buf = g.OverlayGraph.VertexNeighbors(v, forward, t, m, buf)

		// It might happen, that this is the boundary vertex of some cluster that's part
		// of this union graph. We have to iterate over the cluster indices to handle this.
		clusterId, vertexId := g.OverlayGraph.VertexCluster(v)
		for i, id := range g.Indices {
			if clusterId == id {
				// Add the in cluster edges, and remember that they are offset.
				cluster := g.Cluster[i]
				current := len(buf)
				offset  := g.OverlayGraph.ClusterSize(i)
				buf = cluster.VertexNeighbors(vertexId, forward, t, m, buf)
				for j := current; j < len(buf); j++ {
					v := buf[j].Vertex
					if v > offset {
						// internal vertex
						buf[j].Vertex = v - offset + g.Offsets[i]
					} else {
						// boundary vertex
						buf[j].Vertex = g.OverlayGraph.ClusterVertex(clusterId, v)
					}
				}
				break
			}
		}

		return buf
	}
	
	// This is a cluster vertex, we have to compute the correct id within
	// the cluster and defer to the implementation in GraphFile.
	// TODO: refactor into a function.
	clusterId := g.Indices[index]
	cluster   := g.Cluster[index]
	offset    := g.OverlayGraph.ClusterSize(clusterId)
	id        := v + offset
	buf = cluster.VertexNeighbors(id, forward, t, m, buf)
	for i := 0; i < len(buf); i++ {
		v := buf[i].Vertex
		if v > offset {
			// internal vertex
			buf[i].Vertex = v - offset + g.Offsets[index]
		} else {
			// boundary vertex
			buf[i].Vertex = g.OverlayGraph.ClusterVertex(clusterId, v)
		}
	}
	return buf
}


// Mockups which you should never use, but which ensure that the interface is complete...

func (g *UnionGraph) EdgeCount() int {
	return 0
}

func (g *UnionGraph) VertexEdges(v Vertex, forward bool, t Transport, buf []Edge) []Edge {
	return buf
}

func (g *UnionGraph) VertexAccessible(v Vertex, t Transport) bool {
	return false
}

func (g *UnionGraph) VertexCoordinate(Vertex) geo.Coordinate {
	return geo.Coordinate{0,0}
}

func (g *UnionGraph) EdgeOpposite(Edge, v Vertex) Vertex {
	return v
}

func (g *UnionGraph) EdgeSteps(Edge, Vertex, []geo.Coordinate) []geo.Coordinate {
	return nil
}

func (g *UnionGraph) EdgeWeight(Edge, Transport, Metric) float64 {
	return 0
}

// direct access to edge attributes
func (g *UnionGraph) EdgeFerry(Edge) bool {
	return false
}

func (g *UnionGraph) EdgeMaxSpeed(Edge) int {
	return 0
}

func (g *UnionGraph) EdgeOneway(Edge, Transport) bool {
	return false
}

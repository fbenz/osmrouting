
package route

import (
	//"fmt"
	"graph"
	"math"
)

type Router struct {
	Graph  graph.Graph
	Parent []graph.Vertex
	Dist   []float32
	Heap   Heap
}

// Problem Setup

func (r *Router) Reset(g graph.Graph) {
	vertexCount := g.VertexCount()
	r.Graph = g
	
	// We use the parent array to reconstruct the shortest path
	// tree. Since there might be multiple source nodes we initialize
	// the array to the identity (corresponding to n self loops).
	// This makes it easy to recognize root nodes later on.
	if r.Parent == nil || cap(r.Parent) < vertexCount {
		//fmt.Printf("Reallocating the Parent Array with capacity %v.\n", vertexCount)
		r.Parent = make([]graph.Vertex, vertexCount)
	} else {
		r.Parent = r.Parent[:vertexCount]
	}
	for i := range r.Parent {
		r.Parent[i] = graph.Vertex(i)
	}
	
	// The distance array is only valid if a vertex is already
	// processed, so there is no need to initialize it.
	if r.Dist == nil || cap(r.Dist) < vertexCount {
		//fmt.Printf("Reallocating the Distance Array with capacity %v.\n", vertexCount)
		r.Dist = make([]float32, vertexCount)
	} else {
		r.Dist = r.Dist[:vertexCount]
	}
	
	(&r.Heap).Reset(vertexCount)
}

func (r *Router) AddSource(v graph.Vertex, distance float32) {
	// The Dist field will be set during Run.
	(&r.Heap).Push(v, distance)
}

// Dijkstra

func (r *Router) Run(forward bool, t graph.Transport, m graph.Metric) {
	g := r.Graph
	h := &r.Heap
	edges := []graph.Edge(nil)
	for !h.Empty() {
		curr, dist := h.Pop()
		r.Dist[curr] = dist
		edges = g.VertexEdges(curr, forward, t, edges)
		for _, e := range edges {
			n := g.EdgeOpposite(e, curr)
			// switch h.Color(n) {
			i := h.Index[n]
			if i == 0 { // Color == White
				// New vertex
				tmpDist := dist + float32(g.EdgeWeight(e, t, m))
				h.Push(n, tmpDist)
				r.Parent[n] = curr
			} else if i > 1 { // Color == Gray
				// Already in the heap
				tmpDist := dist + float32(g.EdgeWeight(e, t, m))
				// if tmpDist < h.Priority(n) {
				if tmpDist < h.Items[i-2].Priority {
					h.DecreaseKey(n, tmpDist)
					r.Parent[n] = curr
				}
			}
		}
	}
}

// Result Queries

func (r *Router) Distance(v graph.Vertex) float32 {
	c := (&r.Heap).Color(v)
	if c == Black {
		return r.Dist[int(v)]
	} else if c == Gray {
		return (&r.Heap).Priority(v)
	}
	return float32(math.Inf(1))
}

func (r *Router) Reachable(v graph.Vertex) bool {
	return (&r.Heap).Color(v) != White
}

func (r *Router) Processed(v graph.Vertex) bool {
	return (&r.Heap).Color(v) == Black
}

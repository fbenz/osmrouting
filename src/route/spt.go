
package route

import (
	"log"
	"graph"
	"math"
)

type Router struct {
	Graph     graph.Graph
	Parent    []graph.Vertex
	Dist      []float32
	Heap      Heap
	Forward   bool
	Transport graph.Transport
	Metric    graph.Metric
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

// Add a new Source if Forward == true, or a sink if Forward == false.
func (r *Router) AddSource(v graph.Vertex, distance float32) {
	// The Dist field will be set during Run.
	(&r.Heap).Push(v, distance)
}

// Dijkstra

func (r *Router) Run() {
	g, h    := r.Graph, &r.Heap
	t, m    := r.Transport, r.Metric
	forward := r.Forward
	edges   := []graph.Edge(nil)
	
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

func (r *Router) parent_edge(v graph.Vertex, buf []graph.Edge) (graph.Edge, []graph.Edge) {
	g := r.Graph
	u := r.Parent[v]
	
	// Since there are parallel edges in the graph we have to look for the edge of minimum
	// weight between u and v.
	minEdge   := graph.Edge(-1)
	minWeight := math.Inf(1)
	found := false
	buf = g.VertexEdges(u, r.Forward, r.Transport, buf)
	for _, e := range buf {
		n := g.EdgeOpposite(e, u)
		if n != v {
			continue
		}
		weight := g.EdgeWeight(e, r.Transport, r.Metric)
		if !found || weight < minWeight {
			minEdge   = e
			minWeight = weight
		}
		found = true
	}
	
	if !found {
		log.Fatalf("Found no edge between a vertex and its parent in the shortest path tree.")
	}
	
	// Use this opportunity to check that our solution is dual feasible (=> optimal).
	//w := r.Dist[v] - r.Dist[u]
	//if math.Abs(float64(w - float32(minWeight))) > 0.1 {
	//	log.Printf("Edge %v from %v to %v.\n", minEdge, u, v)
	//	log.Fatalf("Dual infeasible solution in Dijkstra, dual: %v, weight: %v.",
	//		w, float32(minWeight))
	//}
	
	return minEdge, buf
}

// Returns a shortest path from a source vertex to the vertex t or nil
// if t is not reachable from any source vertex.
// If Forward == true then the returned path starts at a source vertex
// and extends to t, otherwise it starts at t and leads to a source
// vertex.
// The return value contains n+1 vertices vs and n edges es such that
// es[i] is the edge from vertex vs[i] to vs[i+1].
func (r *Router) Path(t graph.Vertex) ([]graph.Vertex, []graph.Edge) {
	stepCount, s := 0, t
	for r.Parent[s] != s {
		stepCount++
		s = r.Parent[s]
	}
	
	if stepCount == 0 {
		// t is a source vertex
		return []graph.Vertex{t}, []graph.Edge(nil)
	}
	
	vertices := make([]graph.Vertex, stepCount+1)
	path     := make([]graph.Edge,   stepCount)
	
	iv, ie, dir := stepCount, stepCount-1, -1
	if !r.Forward {
		iv, ie, dir = 0, 0, 1
	}
	
	v   := t
	buf := []graph.Edge(nil)
	for r.Parent[v] != v {
		var e graph.Edge
		e, buf = r.parent_edge(v, buf)
		vertices[iv] = v
		path[ie] = e
		v = r.Parent[v]
		iv += dir
		ie += dir
	}
	vertices[iv] = v
	
	return vertices, path
}

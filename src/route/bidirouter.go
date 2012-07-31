package route

import (
	"graph"
	"log"
	"math"
)

type BidiRouter struct {
	// Data Structures
	SParent []graph.Vertex
	SDist   []float32
	SHeap   Heap
	TParent []graph.Vertex
	TDist   []float32
	THeap   Heap
	// Graph Data
	Graph     graph.Graph
	Transport graph.Transport
	Metric    graph.Metric
	// Results
	MeetVertex graph.Vertex
	MDistance  float32
}

// Problem Setup

func (r *BidiRouter) Reset(g graph.Graph) {
	vertexCount := g.VertexCount()
	r.Graph = g

	// We use the parent array to reconstruct the shortest path
	// tree. Since there might be multiple source nodes we initialize
	// the array to the identity (corresponding to n self loops).
	// This makes it easy to recognize root nodes later on.
	if r.SParent == nil || cap(r.SParent) < vertexCount {
		//fmt.Printf("Reallocating the Parent Array with capacity %v.\n", vertexCount)
		r.SParent = make([]graph.Vertex, vertexCount)
		r.TParent = make([]graph.Vertex, vertexCount)
	} else {
		r.SParent = r.SParent[:vertexCount]
		r.TParent = r.TParent[:vertexCount]
	}
	for i := range r.SParent {
		r.SParent[i] = graph.Vertex(i)
		r.TParent[i] = graph.Vertex(i)
	}

	// The distance array is only valid if a vertex is already
	// processed, so there is no need to initialize it.
	if r.SDist == nil || cap(r.SDist) < vertexCount {
		//fmt.Printf("Reallocating the Distance Array with capacity %v.\n", vertexCount)
		r.SDist = make([]float32, vertexCount)
		r.TDist = make([]float32, vertexCount)
	} else {
		r.SDist = r.SDist[:vertexCount]
		r.TDist = r.TDist[:vertexCount]
	}

	(&r.SHeap).Reset(vertexCount)
	(&r.THeap).Reset(vertexCount)
	r.MeetVertex = graph.Vertex(-1)
	r.MDistance = float32(math.Inf(1))
}

func (r *BidiRouter) update_meet(v graph.Vertex) {
	sh, th := &r.SHeap, &r.THeap
	if sh.Color(v) == Gray && th.Color(v) == Gray {
		dist := sh.Priority(v) + th.Priority(v)
		if dist < r.MDistance {
			r.MDistance = dist
			r.MeetVertex = v
		}
	}
}

func (r *BidiRouter) AddSource(v graph.Vertex, distance float32) {
	// The Dist field will be set during Run.
	(&r.SHeap).Push(v, distance)
	r.update_meet(v)
}

func (r *BidiRouter) AddTarget(v graph.Vertex, distance float32) {
	(&r.THeap).Push(v, distance)
	r.update_meet(v)
}

// Dijkstra

func (r *BidiRouter) Run() {
	g      := r.Graph
	sh, th := &r.SHeap, &r.THeap
	t, m   := r.Transport, r.Metric
	darts  := []graph.Dart(nil)
	
	// Maintain an upper bound on the optimal distance.
	upperBound := r.MDistance
	meetVertex := r.MeetVertex

	// As soon as one heap is empty we know that we will never find
	// a path... in principle we could also just check one of them,
	// as this situation should never happen.
	// The real termination condition is the following: If our upper
	// bound is less than the sum of the weights of the top elements
	// in the heap, the current meetVertex lies on the shortest path.
	for !sh.Empty() && !th.Empty() && upperBound > sh.Top()+th.Top() {
		if sh.Top() <= th.Top() {
			// Source step
			curr, dist := sh.Pop()
			r.SDist[curr] = dist
			darts = g.VertexNeighbors(curr, true /* forward */, t, m, darts)
			for _, d := range darts {
				n := d.Vertex
				if sh.Processed(n) {
					continue
				}
				
				tmpDist := dist + d.Weight
				if sh.Update(n, tmpDist) {
					r.SParent[n] = curr

					// Update the distance upper bound
					if !th.Unvisited(n) {
						tdist := float32(0)
						if th.Processed(n) {
							tdist = r.TDist[n]
						} else {
							tdist = th.Priority(n)
						}
						if tmpDist+tdist < upperBound {
							upperBound = tmpDist + tdist
							meetVertex = n
						}
					}
				}
			}
		} else {
			// Target step
			curr, dist := th.Pop()
			r.TDist[curr] = dist
			darts = g.VertexNeighbors(curr, false /* forward */, t, m, darts)
			if len(darts) == 0 {
				//println("Isolated vertex?")
			}
			//log.Printf("curr: %v, dist: %v", curr, dist)
			for _, d := range darts {
				n := d.Vertex
				//log.Printf("  * n: %v, w: %v, processed: %v", n, d.Weight, th.Processed(n))
				if th.Processed(n) {
					continue
				}
				
				tmpDist := dist + d.Weight
				if th.Update(n, tmpDist) {
					r.TParent[n] = curr

					// Update the distance upper bound
					if !sh.Unvisited(n) {
						sdist := float32(0)
						if sh.Processed(n) {
							sdist = r.SDist[n]
						} else {
							sdist = sh.Priority(n)
						}
						if tmpDist+sdist < upperBound {
							upperBound = tmpDist + sdist
							meetVertex = n
						}
					}
				}
			}
		}
	}

	// Record the shortest path
	r.MeetVertex = meetVertex
	r.MDistance = upperBound
	
	if meetVertex == -1 {
		panic("Did not find a path!")
	}
}

// Result Queries

func (r *BidiRouter) Distance() float32 {
	return r.MDistance
}

func (r *BidiRouter) parent_edge(u, v graph.Vertex, forward bool, buf []graph.Edge) (graph.Edge, []graph.Edge) {
	g := r.Graph

	// Since there are parallel edges in the graph we have to look for the edge of minimum
	// weight between u and v.
	minEdge := graph.Edge(-1)
	minWeight := math.Inf(1)
	found := false
	buf = g.VertexEdges(u, forward, r.Transport, buf)
	for _, e := range buf {
		n := g.EdgeOpposite(e, u)
		if n != v {
			continue
		}
		weight := g.EdgeWeight(e, r.Transport, r.Metric)
		if !found || weight < minWeight {
			minEdge = e
			minWeight = weight
		}
		found = true
	}

	if !found {
		log.Fatalf("Found no edge between a vertex and its parent in the shortest path tree.")
	}

	return minEdge, buf
}

// Returns a shortest path from a source vertex to a target vertex.
func (r *BidiRouter) Path() ([]graph.Vertex, []graph.Edge) {
	// Determine the length of the path along with the source vertex s
	// and target vertex t.
	sourceSteps, targetSteps := 0, 0
	s, t := r.MeetVertex, r.MeetVertex
	for r.SParent[s] != s {
		sourceSteps++
		s = r.SParent[s]
	}
	for r.TParent[t] != t {
		targetSteps++
		t = r.TParent[t]
	}

	stepCount := sourceSteps + targetSteps
	if stepCount == 0 {
		// The meet vertex is both a source and a target vertex.
		return []graph.Vertex{t}, []graph.Edge(nil)
	}
	vpath := make([]graph.Vertex, stepCount+1)
	epath := make([]graph.Edge, stepCount)
	buf := []graph.Edge(nil)

	// Path from a source vertex to the meet vertex
	if sourceSteps != 0 {
		v := r.MeetVertex
		i := sourceSteps
		for r.SParent[v] != v {
			var e graph.Edge
			u := r.SParent[v]
			e, buf = r.parent_edge(u, v, true, buf)
			vpath[i] = v
			epath[i-1] = e
			v = u
			i--
		}
		vpath[0] = v
	}

	// Path from the meet vertex to a target vertex.
	if targetSteps != 0 {
		v := r.MeetVertex
		i := sourceSteps
		for r.TParent[v] != v {
			var e graph.Edge
			u := r.TParent[v]
			e, buf = r.parent_edge(u, v, false, buf)
			vpath[i] = v
			epath[i] = e
			v = u
			i++
		}
		vpath[i] = v
	}

	return vpath, epath
}

// Convenience function to find a forward edge (of minimum weight) from
// vertex u to vertex v. Useful for reconstructing cut edges.
func (r *BidiRouter) EdgeBetween(u, v graph.Vertex) graph.Edge {
	g         := r.Graph
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

	if int(minEdge) == -1 {
		log.Fatalf("Found no edge between the two given vertices.")
	}

	return minEdge
}

// Returns the vertices on a shortest path from a source vertex to a target vertex.
// Has the great advantage that it actually works for OverlayGraphs.
func (r *BidiRouter) VPath() []graph.Vertex {
	// Determine the length of the path along with the source vertex s
	// and target vertex t.
	sourceSteps, targetSteps := 0, 0
	s, t := r.MeetVertex, r.MeetVertex
	for r.SParent[s] != s {
		sourceSteps++
		s = r.SParent[s]
	}
	for r.TParent[t] != t {
		targetSteps++
		t = r.TParent[t]
	}

	stepCount := sourceSteps + targetSteps
	if stepCount == 0 {
		// The meet vertex is both a source and a target vertex.
		return []graph.Vertex{t}
	}
	vpath := make([]graph.Vertex, stepCount+1)

	// Path from a source vertex to the meet vertex
	if sourceSteps != 0 {
		v := r.MeetVertex
		i := sourceSteps
		for r.SParent[v] != v {
			vpath[i] = v
			v = r.SParent[v]
			i--
		}
		vpath[0] = v
	}

	// Path from the meet vertex to a target vertex.
	if targetSteps != 0 {
		v := r.MeetVertex
		i := sourceSteps
		for r.TParent[v] != v {
			vpath[i] = v
			v = r.TParent[v]
			i++
		}
		vpath[i] = v
	}

	return vpath
}

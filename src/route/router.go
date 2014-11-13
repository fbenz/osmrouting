/*
 * Copyright 2014 Florian Benz, Steven Schäfer, Bernhard Schommer
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package route

import (
	"errors"
	"fmt"
	"graph"
	"log"
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
	g, h := r.Graph, &r.Heap
	t, m := r.Transport, r.Metric
	forward := r.Forward
	darts := []graph.Dart(nil)

	for !h.Empty() {
		curr, dist := h.Pop()
		r.Dist[curr] = dist
		darts = g.VertexNeighbors(curr, forward, t, m, darts)
		for _, d := range darts {
			n := d.Vertex
			if h.Processed(n) {
				continue
			}

			if h.Update(n, dist+d.Weight) {
				r.Parent[n] = curr
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
	minEdge := graph.Edge(-1)
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
	path := make([]graph.Edge, stepCount)

	iv, ie, dir := stepCount, stepCount-1, -1
	if !r.Forward {
		iv, ie, dir = 0, 0, 1
	}

	v := t
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

// Check that the distance function is dual feasible for all Reachable
// vertices and that the parent pointers define a primal solution which
// obeys the complementary slackness conditions. This implies that the
// solution is optimal, as we assumed positive edge weights.
// More concretely we need to check for each edge e = (u,v) with weight w:
//  * Dist[v] <= Dist[u] + w
//  * Dist[v]  = Dist[u] + w  if u == Parent[v] and e is in the SPT.
// The last check is deferred, since we may have parallel edge in the graph.
//
// TODO:
// Additionally, we should check that Dist[s] <= Init[s] for source vertices,
// and that equality holds for the roots of the shortest path forest. This would
// be additional work though, as we currently do not store the source vertices.
func (r *Router) CertifySolution() (bool, error) {
	// Check dual feasibility along with reachability and the edge weights.
	g := r.Graph
	buf := []graph.Edge(nil)
	for i := 0; i < g.VertexCount(); i++ {
		u := graph.Vertex(i)
		if !r.Reachable(u) {
			continue
		}

		buf = g.VertexEdges(u, r.Forward, r.Transport, buf)
		for _, e := range buf {
			v := g.EdgeOpposite(e, u)
			if !r.Reachable(v) {
				return false, errors.New(fmt.Sprintf("Reachable set is not closed: "+
					"vertex %v is reachable, and there is an edge %v to vertex %v which "+
					"is marked as unreachable.", u, e, v))
			}

			// Check that the edge weights are sensible
			w := float32(g.EdgeWeight(e, r.Transport, r.Metric))
			// There shouldn't be any zero weight edges either, but that needs to be
			// ensured in the parser...
			if w == 0 || math.IsInf(float64(w), 0) || math.IsNaN(float64(w)) {
				return false, errors.New(fmt.Sprintf("Edge %v has weight %v.", e, w))
			}

			if r.Dist[v] > r.Dist[u]+w {
				return false, errors.New(fmt.Sprintf("Solution is not dual feasible. "+
					"For edge %v from %v to %v we have: "+
					"Dist[%v] = %v > %v = %v + %v = Dist[%v] + Weight[%v].",
					e, u, v, v, r.Dist[v], r.Dist[u]+w, r.Dist[u], w, u, e))
			}
		}
	}

	// Check complementary slackness of the primal solution.
	for i := 0; i < g.VertexCount(); i++ {
		v := graph.Vertex(i)
		if !r.Reachable(v) || r.Parent[v] == v {
			continue
		}

		// Find the weight of the tree edge from Parent[v] to v.
		minEdge := graph.Edge(-1)
		minWeight := float32(math.Inf(1))
		buf = g.VertexEdges(v, !r.Forward, r.Transport, buf)
		for _, e := range buf {
			u := g.EdgeOpposite(e, v)
			if u != r.Parent[v] {
				continue
			}
			w := float32(g.EdgeWeight(e, r.Transport, r.Metric))
			if w < minWeight {
				minWeight = w
				minEdge = e
			}
		}

		// Check that this edge is tight.
		u := r.Parent[v]
		if r.Dist[v] != r.Dist[u]+minWeight {
			return false, errors.New(fmt.Sprintf("Solution is not optimal. "+
				"For the tree edge %v from %v to %v we have: "+
				"Dist[%v] = %v != %v = %v + %v = Dist[%v] + Weight[%v].",
				minEdge, u, v, v, r.Dist[v], r.Dist[u]+minWeight,
				r.Dist[u], minWeight, u, minEdge))
		}
	}

	return true, nil
}

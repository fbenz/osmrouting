package route

import (
	"alg"
	"graph"
	"log"
)

func DijkstraBidirectional(g graph.Graph, s, t []*Element, m graph.Metric, trans graph.Transport) (float64, []graph.Vertex, []graph.Edge) {
	var meetvertex graph.Vertex
	spq := ArrayToPriorityQueue(&s)
	tpq := ArrayToPriorityQueue(&t)
	elms := make([]*Element, g.VertexCount())
	elmt := make([]*Element, g.VertexCount())
	visited := alg.NewBitVector(uint(g.VertexCount()))
	edges := []graph.Edge(nil)

	for !Empty(spq) && !Empty(tpq) {
		scurrElem := ExtractMin(spq)
		scurr := scurrElem.vertex
		visited.Set(int64(scurr), true)
		scurrDist := elms[scurr].priority
		edges = g.VertexEdges(scurr, true, trans, edges)
		for _, e := range edges {
			n := g.EdgeOpposite(e, scurr)
			if elem := elms[n]; elem != nil {
				if tmpDist := scurrDist + g.EdgeWeight(e, trans, m); tmpDist < elem.priority {
					ChangePriority(spq, elem, tmpDist)
					elem.p = scurr
					elem.ep = e

				}

			} else {
				x := NewElement(n, scurrDist+g.EdgeWeight(e, trans, m))
				elms[x.vertex] = x
				x.p = scurr
				x.ep = e
			}

		}
		tcurrElem := ExtractMin(tpq)
		tcurr := tcurrElem.vertex
		tcurrDist := elms[scurr].priority
		edges = g.VertexEdges(tcurr, true, trans, edges)
		for _, e := range edges {
			n := g.EdgeOpposite(e, tcurr)
			if elem := elms[n]; elem != nil {
				if tmpDist := tcurrDist + g.EdgeWeight(e, trans, m); tmpDist < elem.priority {
					ChangePriority(tpq, elem, tmpDist)
					elem.p = tcurr
					elem.ep = e

				}

			} else {
				x := NewElement(n, tcurrDist+g.EdgeWeight(e, trans, m))
				elmt[x.vertex] = x
				x.p = tcurr
				x.ep = e
			}

		}
		if visited.Get(int64(tcurr)) {
			meetvertex = tcurr
			break
		}
	}
	if ok, path, edges := ComputePath(elms, elmt, meetvertex); ok {
		return elms[meetvertex].priority + elmt[meetvertex].priority, path, edges
	}
	path := make([]graph.Vertex, 1)
	path[0] = s[0].vertex
	return 0.0, path, nil
}

func ComputePath(elms, elmt []*Element, meetvertex graph.Vertex) (bool, []graph.Vertex, []graph.Edge) {
	stepCountS := 0
	curr := meetvertex
	for elem := elms[curr]; elem != nil && elem.vertex != elem.p; elem = elms[curr] {
		curr = elem.p
		stepCountS++
	}
	stepCountT := 0
	curr = meetvertex
	for elem := elmt[curr]; elem != nil && elem.vertex != elem.p; elem = elmt[curr] {
		curr = elem.p
		stepCountT++
	}
	path := make([]graph.Vertex, stepCountS+stepCountT+1)
	edges := make([]graph.Edge, stepCountS+stepCountT)
	if stepCountS+stepCountT == 0 {
		log.Printf("WARNING: dijkstra found no path\n")
		return false, nil, nil

	}
	if stepCountS > 0 {
		position := stepCountS - 1
		curr = meetvertex
		for elem := elms[curr]; elem != nil && elem.vertex != elem.p; elem = elms[curr] {
			path[position+1] = elem.vertex
			edges[position] = elem.ep
			curr = elem.p
			position--
		}
		path[0] = curr
	}
	if stepCountT > 0 {
		position := stepCountS
		curr := elmt[meetvertex].p
		for elem := elmt[curr]; elem != nil && elem.vertex != elem.p; elem = elmt[curr] {
			path[position+1] = elem.vertex
			edges[position] = elem.ep
			curr = elem.p
			position++
		}
		path[stepCountS+stepCountT] = curr
	}
	return true, path, edges
}

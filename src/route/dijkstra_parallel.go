package route

import (
	"alg"
	"graph"
)

func DijkstraRunner(g graph.Graph, s []*Element, m graph.Metric, trans graph.Transport, direction bool, result chan []*Element, stop chan bool, vertices chan graph.Vertex) {
	pq := ArrayToPriorityQueue(&s)
	elm := make([]*Element, g.VertexCount())
	edges := []graph.Edge(nil)

	for !Empty(pq) {
		select {
		case <-stop: // We have found a common vertex
			result <- elm // Send the result
		default:
			currElem := ExtractMin(pq)
			curr := currElem.vertex
			vertices <- curr
			currDist := elm[curr].priority
			edges = g.VertexEdges(curr, direction, trans, edges)
			for _, e := range edges {
				n := g.EdgeOpposite(e, curr)
				if elem := elm[n]; elem != nil {
					if tmpDist := currDist + g.EdgeWeight(e, trans, m); tmpDist < elem.priority {
						ChangePriority(pq, elem, tmpDist)
						elem.p = curr
						elem.ep = e
					}
				} else {
					x := NewElement(n, currDist+g.EdgeWeight(e, trans, m))
					elm[x.vertex] = x
					x.p = curr
					x.ep = e
				}
			}

		}
	}
	//This is the case were we have explored the whole graph
	result <- elm
}

func DijkstraStarter(g graph.Graph, s, t []*Element, m graph.Metric, trans graph.Transport) (float64, []graph.Vertex, []graph.Edge) {
	finishedS := make(chan bool)
	finishedT := make(chan bool)
	resultS := make(chan []*Element)
	resultT := make(chan []*Element)
	verticesS := make(chan graph.Vertex)
	verticesT := make(chan graph.Vertex)
	visitedS := alg.NewBitVector(uint(g.VertexCount()))
	visitedT := alg.NewBitVector(uint(g.VertexCount()))

	go DijkstraRunner(g, s, m, trans, true, resultS, finishedS, verticesS)
	go DijkstraRunner(g, t, m, trans, false, resultT, finishedT, verticesT)

	for {
		select {
		case i := <-verticesS:
			visitedS.Set(int64(i), true)
			if visitedT.Get(int64(i)) {
				finishedS <- true
				finishedT <- true
				elms := <-resultS
				elmt := <-resultT
				if ok, path, edges := ComputePath(elms, elmt, i); ok {
					return elms[i].priority + elmt[i].priority, path, edges
				}
				path := make([]graph.Vertex, 1)
				path[0] = s[0].vertex
				return 0.0, path, nil
			}
		case i := <-verticesT:
			visitedT.Set(int64(i), true)
			if visitedS.Get(int64(i)) {
				finishedS <- true
				finishedT <- true
				elms := <-resultS
				elmt := <-resultT
				if ok, path, edges := ComputePath(elms, elmt, i); ok {
					return elms[i].priority + elmt[i].priority, path, edges
				}
				path := make([]graph.Vertex, 1)
				path[0] = s[0].vertex
				return 0.0, path, nil
			}
		case elms := <-resultS: // the forward dijkstra is finished
			elmt := <-resultT
			minvertex := -1
			for i, e := range elms {
				if e != nil && elmt[i] != nil {
					if minvertex == -1 {
						minvertex = i
					} else {
						if elmt[i].priority+elms[i].priority < elmt[minvertex].priority+elms[minvertex].priority {
							minvertex = i
						}
					}
				}
			}
			if minvertex != -1 {
				if ok, path, edges := ComputePath(elms, elmt, graph.Vertex(minvertex)); ok {
					return elms[minvertex].priority + elmt[minvertex].priority, path, edges
				}
			}
			path := make([]graph.Vertex, 1)
			path[0] = s[0].vertex
			return 0.0, path, nil
		case elmt := <-resultT: // the backward dijkstra is finished
			elms := <-resultS
			minvertex := -1
			for i, e := range elms {
				if e != nil && elmt[i] != nil {
					if minvertex == -1 {
						minvertex = i
					} else {
						if elmt[i].priority+elms[i].priority < elmt[minvertex].priority+elms[minvertex].priority {
							minvertex = i
						}
					}
				}
			}
			if minvertex != -1 {
				if ok, path, edges := ComputePath(elms, elmt, graph.Vertex(minvertex)); ok {
					return elms[minvertex].priority + elmt[minvertex].priority, path, edges
				}
			}
			path := make([]graph.Vertex, 1)
			path[0] = s[0].vertex
			return 0.0, path, nil
		}

	}
	return 0.0, nil, nil
}

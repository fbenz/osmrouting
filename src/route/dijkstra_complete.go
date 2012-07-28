package route

import (
	"graph"
	"log"
)

func DijkstraComplete(g graph.Graph, s []graph.Way, m graph.Metric, trans graph.Transport, forward bool) []*Element {
	elements := make([]*Element, g.VertexCount())
	q := NewPriorityQueue(1024)
	edges := []graph.Edge(nil)

	for _, str := range s {
		priority := str.Length
		x := NewElement(str.Vertex, priority)
		elements[x.vertex] = x
		PushElement(&q, x)
	}

	for !Empty(&q) {
		currElem := ExtractMin(&q)
		curr := currElem.vertex
		currDist := elements[curr].priority
		edges = g.VertexEdges(curr, forward, trans, edges)
		for _, e := range edges {
			n := g.EdgeOpposite(e, curr)
			if elem := elements[n]; elem != nil {
				if elem.index != -1 {
					if tmpDist := currDist + g.EdgeWeight(e, trans, m); tmpDist < elem.priority {
						ChangePriority(&q, elem, tmpDist)
						elem.p = curr
						elem.ep = e
					}
				}
			} else {
				x := NewElement(n, currDist+g.EdgeWeight(e, trans, m))
				elements[x.vertex] = x
				x.p = curr
				x.ep = e
				PushElement(&q, x)
			}
		}
	}
	return elements
}

func ConstructForwardPath(spt []*Element, v graph.Vertex) ([]graph.Vertex, []graph.Edge) {
	stepCount := 0
	curr := v
	for elem := spt[curr]; elem != nil && elem.vertex != elem.p; elem = spt[curr] {
		curr = elem.p
		stepCount++
	}
	path := make([]graph.Vertex, stepCount+1)
	edges := make([]graph.Edge, stepCount)
	if stepCount == 0 {
		log.Printf("WARNING: dijkstra found no path\n")
		return nil, nil
	}
	position := stepCount - 1
	curr = v
	for elem := spt[curr]; elem != nil && elem.vertex != elem.p; elem = spt[curr] {
		path[position+1] = elem.vertex
		edges[position] = elem.ep
		curr = elem.p
		position--
	}
	path[0] = curr
	return path, edges
}

func ConstructBackwardPath(spt []*Element, v graph.Vertex) ([]graph.Vertex, []graph.Edge) {
	stepCount := 0
	curr := v
	for elem := spt[curr]; elem != nil && elem.vertex != elem.p; elem = spt[curr] {
		curr = elem.p
		stepCount++
	}
	path := make([]graph.Vertex, stepCount+1)
	edges := make([]graph.Edge, stepCount)
	if stepCount == 0 {
		log.Printf("WARNING: dijkstra found no path\n")
		return nil, nil
	}
	position := 0
	curr = v
	path[0] = curr
	for elem := spt[curr]; elem != nil && elem.vertex != elem.p; elem = spt[curr] {
		path[position+1] = elem.vertex
		edges[position] = elem.ep
		curr = elem.p
		position++
	}
	return path, edges
}

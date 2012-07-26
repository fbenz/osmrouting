package route

import (
	"graph"
)

func DijkstraComplete(g graph.Graph, s []graph.Way, m graph.Metric, forward bool) []*Element {
	elements := make([]*Element, g.VertexCount())
	q := NewPriorityQueue(1024)

	for _, str := range s {
		priority := str.Length
		x := NewElement(str.Node, priority)
		elements[x.vertex] = x
		PushElement(&q, x)
	}

	for !Empty(&q) {
		currElem := ExtractMin(&q)
		curr := currElem.vertex
		currDist := elements[curr].priority
		iter := g.VertexEdgeIterator(curr, forward, m)
		for e, ok := iter.Next(); ok; e, ok = iter.Next() {
			n := g.EdgeOpposite(e, curr)
			if elem := elements[n]; elem != nil {
				if tmpDist := currDist + g.EdgeWeight(e, m); tmpDist < elem.priority {
					ChangePriority(&q, elem, tmpDist)
					elem.p = curr
					elem.ep = e
				}
			} else {
				x := NewElement(n, currDist+g.EdgeWeight(e, m))
				elements[x.vertex] = x
				x.p = curr
				x.ep = e
				PushElement(&q, x)
			}
		}
	}

	return elements

}

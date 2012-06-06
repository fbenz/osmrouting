package alg

import (
	"graph"
	"container/heap"
	"container/list"
)

func isInList(w []graph.Way, s graph.Node) bool {
	for _, way := range w {
		if s == way.Node {
			return true
		}
	}
	return false
}

// A slightly optimized version of dijkstras algorithm
// Takes an graph as argument and returns an list of vertices in order
// of the path
func Dijkstra(s, t []graph.Way) (float64, *list.List, *list.List) {
	d := make(map[graph.Node]float64)                // I assume distance is an integer
	p := make(map[graph.Node]graph.Node)               // Predecessor list
	ep := make(map[graph.Node]graph.Edge) // Edge Predecessors
	q := NewPriorityQueue(100 /* initialCapacity */) // 100 is just a first guess
	final := make(map[graph.Node]bool)
	for _, tar := range t {
		final[tar.Node] = false
	}
	for _, str := range s {
		priority := str.Length
		x := NewElement(str.Node, int(priority)) // TODO check this cast
		heap.Push(&q, x)
		d[str.Node] = priority
	}
	for !q.Empty() {
		currElem := (heap.Pop(&q)).(*Element) // Get the first element
		curr := currElem.Value.(graph.Node)            // Unbox the node
		if isfinal, _ := final[curr]; isfinal {                    
			final[curr] = true
			finished := true
			for _, node := range final {
				finished = finished && node
			}
			if !finished {
				break
			}
		}
		currDist := d[curr]
		for _, e := range curr.Edges() {
			n := e.EndPoint()
			if dist, ok := d[n]; ok {
				if tmpDist := currDist + e.Length(); tmpDist < dist {
					q.ChangePriority(currElem, int(tmpDist)) // TODO again check cast
					d[n] = tmpDist
					p[n] = curr
					ep[n]=e
				}
			} else {
				d[n] = currDist + e.Length()
				p[n] = curr
				elem := NewElement(n, int(currDist))  // TODO again check cast
				heap.Push(&q, elem)
			}
		}
	}
	path := list.New()
	edges := list.New()
	// Construct the list by moving from t to s
	first := true
	var curr graph.Node
	var currdist float64
	for _, targetnode := range t {
		tmpnode := targetnode.Node
		tmpdist := d[tmpnode] + targetnode.Length
		if first {
			curr = tmpnode
			currdist = tmpdist
		} else {
			if tmpdist<currdist {
				curr = tmpnode
				currdist = tmpdist
			}
		}
	}
	for isInList(s, curr) {
		path.PushFront(curr)
		curr = p[curr]
		edges.PushFront(p[curr])
	}
	path.PushFront(p[curr])
	// TODO fix, t[0] is not necessarily optimal
	return d[t[0].Node], path, edges
}

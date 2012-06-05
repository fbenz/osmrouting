package alg

import (
	"../graph"
	"../pq"
	"container/heap"
	"container/list"
)

// A slightly optimized version of dijkstras algorithm
// Takes an graph as argument and returns an list of vertices in order
// of the path
func Dijkstra(g graph.Graph, s, t uint) (int, *list.List) {
	d := make(map[uint]int)  // I assume distance is an integer
	p := make(map[uint]uint) // Predecessor list
	q := pq.New(100 /* initialCapacity */) // 100 is just a first guess
	start := pq.NewElement(s, 0)
	heap.Push(&q, start)
	d[s] = 0
	for !q.Empty() {
		currElem := (heap.Pop(&q)).(*pq.Element) // Get the first element
		curr := currElem.Value.(uint) // Unbox the id
		if curr == t { // If we remove t from the queue we can stop since dist(x)>=dist(t) for all x in q
			break
		}
		currDist := d[curr]
		for _, e := range g.Outgoing(curr) {
			n := e.Endpoint()
			if dist, ok := d[n]; ok {
				if tmpDist := currDist + e.Weight(); tmpDist < dist {
					q.ChangePriority(currElem, tmpDist)
					d[n] = currDist + e.Weight()
					p[n] = curr
				}
			} else {
				d[n] = currDist + e.Weight()
				p[n] = curr
				elem := pq.NewElement(n, currDist)
				heap.Push(&q, elem)
			}
		}
	}
	path := list.New()
	// Construct the list by moving from t to s
	for curr := t; curr != s; {
		path.PushFront(curr)
		curr = p[curr]
	}
	path.PushFront(s)
	return d[t], path
}


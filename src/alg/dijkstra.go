package alg

import (
	"graph"
	"container/heap"
	"container/list"
)

func isInList(w []Way,s Node) bool {
	for _,node:=range w {
		if s==w {
			return true
		}
	}
	return false
}

// A slightly optimized version of dijkstras algorithm
// Takes an graph as argument and returns an list of vertices in order
// of the path
func Dijkstra(s, t []Way) (float64, *list.List, *list.List) {
	d := make(map[Node]float64)                // I assume distance is an integer
	p := make(map[Node]Node)               // Predecessor list
	ep := make(map[Node]Edge) // Edge Predecessors
	q := pq.New(100 /* initialCapacity */) // 100 is just a first guess
	final := make(map[Node]bool)
	for _,tar := range t {
		final[t]=false
	}
	for _,str := range s {
		priority := str.Length()
		x := heap.NewElement(str.Node(),priority)
		heap.Push(q,x)
		d[str]=priority
	}
	for !q.Empty() {
		currElem := (heap.Pop(&q)).(*pq.Element) // Get the first element
		curr := currElem.Value.(uint)            // Unbox the id
		if isfinal,val:=final[curr];isfinal {                    
			final[curr]=true
			finished:=true
			for _,node := range final {
				finished &= node
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
					q.ChangePriority(currElem, tmpDist)
					d[n] = tmpDist
					p[n] = curr
					ep[n]=e
				}
			} else {
				d[n] = currDist + e.Length()
				p[n] = curr
				elem := pq.NewElement(n, currDist)
				heap.Push(&q, elem)
			}
		}
	}
	path := list.New()
	edges := list.New()
	// Construct the list by moving from t to s
	first := true
	var curr Node
	var currdist float64
	for _,targetnode := range t {
		tmpnode := targetnode.Node()
		tmpdist := d[tmpnode] + targetnode.Length()
		if first {
			curr = tmpnode
			currdist = tmpdist
		}
		else {
			if tmpdist<currdist {
				curr = tmpnode
				currdist = tmpdist
			}

		}
	}
	for isInList(curr,s) {
		path.PushFront(curr)
		curr = p[curr]
		edges.PushFront(p[curr])
	}
	path.PushFront(p[curr])
	return d[t], path,edges
}

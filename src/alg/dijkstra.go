package alg

import (
	"container/heap"
	"container/list"
	"log"
	//"fmt"
	"graph"
)

func isInList(w []graph.Way, s graph.Node) (bool, graph.Way) {
	var resultway graph.Way
	for _, way := range w {
		if s == way.Node {
			return true, way
		}
	}
	return false, resultway
}

// A slightly optimized version of dijkstras algorithm
// Takes an graph as argument and returns an list of vertices in order
// of the path
func Dijkstra(g graph.Graph, s, t []graph.Way) (float64, *list.List, *list.List, graph.Way, graph.Way) {
	d := make(map[graph.Node]float64)
	p := make(map[graph.Node]graph.Node)  // Predecessor list
	ep := make(map[graph.Node]graph.Edge) // Edge Predecessors
	q := NewPriorityQueue(1024)
	final := make(map[graph.Node]bool)
	for _, tar := range t {
		final[tar.Node] = true
	}
	for _, str := range s {
		priority := str.Length
		x := NewElement(str.Node, priority) // TODO check this cast
		heap.Push(&q, x)
		d[str.Node] = priority
	}
	for !q.Empty() {
		currElem := (heap.Pop(&q)).(*Element) // Get the first element
		curr := currElem.Value.(graph.Node)            // Unbox the node
		if final[curr] {
			// We're done as soon as we hit the last final node
			final[curr] = false
			finished := true
			for _, tar := range t {
				if final[tar.Node] {
					finished = false
					break
				}
			}
			if finished {
				break
			}
		}
		currDist := d[curr]
		
		for currEdge, endEdge := g.NodeEdges(curr); currEdge <= endEdge; currEdge++ {
			n := g.EdgeEndPoint(currEdge)
			elem := NewElement(n, currDist)
			if dist, ok := d[n]; ok {
				if tmpDist := currDist + g.EdgeLength(currEdge); tmpDist < dist {
					q.ChangePriority(elem, tmpDist)
					d[n] = tmpDist
					p[n] = curr
					ep[n] = currEdge
				}
			} else {
				d[n] = currDist + g.EdgeLength(currEdge)
				p[n] = curr
				ep[n] = currEdge
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
	var endway graph.Way
	var startway graph.Way
	for _, targetnode := range t {
		tmpnode := targetnode.Node
		dist, ok := d[tmpnode]
		if !ok {
			continue
		}
		tmpdist := dist + targetnode.Length
		if first {
			curr = tmpnode
			currdist = tmpdist
			endway = targetnode
			first = false
		} else {
			if tmpdist < currdist {
				curr = tmpnode
				currdist = tmpdist
				endway = targetnode
			}
		}
	}
	
	var start bool
	for start, startway = isInList(s, curr); !start; start, startway = isInList(s, curr) {
		if curr == p[curr] {
			log.Printf("WARNING: dijkstra found no path\n")
			break
		}

		//fmt.Printf("curr: %v\n", curr)
		//fmt.Printf("p:    %v\n", p[curr])
		//fmt.Printf("ep:   %v\n", ep[curr])
		path.PushFront(curr)
		edges.PushFront(ep[curr])
		curr = p[curr]
	}
	path.PushFront(curr)
	//fmt.Printf("path: %v\n", path)
	//fmt.Printf("dist: %v\n", currdist)
	//fmt.Printf("edges: %v\n", edges)
	//fmt.Printf("startway: %v\n", startway)
	//fmt.Printf("endway: %v\n", endway)
	// TODO fix, t[0] is not necessarily optimal
	
	//fmt.Printf("len(d) = %v\n", len(d))
	//fmt.Printf("len(p) = %v\n", len(p))
	//fmt.Printf("len(ep) = %v\n", len(ep))
	
	return currdist, path, edges, startway, endway
}

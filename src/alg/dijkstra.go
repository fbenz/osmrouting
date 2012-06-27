package alg

import (
	"container/heap"
	"log"
	"graph"
	
	//"fmt"
	//"time"
)

// A slightly optimized version of dijkstras algorithm
// Takes an graph as argument and returns an list of vertices in order
// of the path
func Dijkstra(g graph.Graph, s, t []graph.Way) (float64, []graph.Node, []graph.Edge, graph.Way, graph.Way) {
	//time1 := time.Now()
	
	elements := make(map[graph.Node]*Element)
	q := NewPriorityQueue(1024)

	final := make(map[graph.Node]bool)
	for _, tar := range t {
		final[tar.Node] = true
	}
	for _, str := range s {
		priority := str.Length
		x := NewElement(str.Node, priority, priority)
		elements[x.node] = x
		heap.Push(&q, x)
	}
	
	//time2 := time.Now()
	
	for !q.Empty() {
		currElem := (heap.Pop(&q)).(*Element) // Get the first element
		curr := currElem.node
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
		currDist := elements[curr].d
		
		for currEdge, endEdge := g.NodeEdges(curr); currEdge <= endEdge; currEdge++ {
			n := g.EdgeEndPoint(currEdge)
			if elem, ok := elements[n]; ok {
				if tmpDist := currDist + g.EdgeLength(currEdge); tmpDist < elem.d {
					q.ChangePriority(elem, tmpDist) // TODO A*? tmpDist + estimate 
					elem.d = tmpDist
					elem.p = curr
					elem.ep = currEdge
				}
			} else {
				x := NewElement(n, currDist /* priority */, currDist + g.EdgeLength(currEdge) /* d*/)
				elements[x.node] = x
				x.p = curr
				x.ep = currEdge
				heap.Push(&q, x)
			}
		}
	}
	
	//time3 := time.Now()

	// Construct the list by moving from t to s
	first := true
	var curr graph.Node
	var currdist float64
	var endway graph.Way
	var startway graph.Way
	for _, targetnode := range t {
		tmpnode := targetnode.Node
		elem, ok := elements[tmpnode]
		if !ok {
			continue
		}
		dist := elem.d
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
	
	oldCurr := curr
	stepCount := 0
	for elem, ok := elements[curr]; ok && elem.node != elem.p; elem, ok = elements[curr] {
		curr = elem.p
		stepCount++
	}
	curr = oldCurr
	
	path := make([]graph.Node, stepCount + 1)
	edges := make([]graph.Edge, stepCount)
	
	if stepCount == 0 {
		log.Printf("WARNING: dijkstra found no path\n")
		path[0] = s[0].Node
		return 0, path, edges, s[0], t[0]
	}
	
	position := stepCount - 1
	for elem, ok := elements[curr]; ok && elem.node != elem.p; elem, ok = elements[curr] {
		//fmt.Printf("curr: %v\n", curr)
		//fmt.Printf("p:    %v\n", p[curr])
		//fmt.Printf("ep:   %v\n", ep[curr])
		path[position + 1] = elem.node
		edges[position] = elem.ep
		curr = elem.p
		position--
	}
	path[0] = curr

	// Choose startway corresponding to the path created by Dijkstra's algorithm
	for _, way := range s {
		if path[0] == way.Node {
			startway = way
			break
		}
	}

	//fmt.Printf("path: %v\n", path)
	//fmt.Printf("dist: %v\n", currdist)
	//fmt.Printf("edges: %v\n", edges)
	//fmt.Printf("startway: %v\n", startway)
	//fmt.Printf("endway: %v\n", endway)
	// TODO fix, t[0] is not necessarily optimal
	
	//fmt.Printf("len(d) = %v\n", len(d))
	//fmt.Printf("len(p) = %v\n", len(p))
	//fmt.Printf("len(ep) = %v\n", len(ep))
	
	/*time4 := time.Now()
	fmt.Printf("time 1-2: %v\n", time2.Sub(time1).Nanoseconds() / 1000)
	fmt.Printf("time 2-3:func isInList(w []graph.Way, s graph.Node) (bool, graph.Way) {
	var resultway graph.Way
	for _, way := range w {
		if s == way.Node {
			return true, way
		}
	}
	return false, resultway
} %v\n", time3.Sub(time2).Nanoseconds() / 1000)
	fmt.Printf("time 3-4: %v\n", time4.Sub(time3).Nanoseconds() / 1000)
	fmt.Printf("stepCount: %v, %v, %v\n", stepCount, len(path), len(edges))*/
	
	return currdist, path, edges, startway, endway
}

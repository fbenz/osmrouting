package main

import (
	"alg"
	"fmt"
	"graph"
	"math/rand"
)

const (
	MaxTrials = 100
	MinSize   = 0.50
)

func Reach(g graph.Graph, v graph.Vertex, forward bool, mode graph.Transport) []byte {
	result := make([]byte, (g.VertexCount() + 7) / 8)
	queue  := make([]graph.Vertex, 1, 128)
	alg.SetBit(result, uint(v))
	queue[0] = v
	
	for len(queue) > 0 {
		s := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		iter := g.VertexEdgeIterator(s, forward, mode)
		for e, ok := iter.Next(); ok; e, ok = iter.Next() {
			//fmt.Printf("e: %v\n", e)
			t := g.EdgeOpposite(e, s)
			if !alg.GetBit(result, uint(t)) {
				alg.SetBit(result, uint(t))
				queue = append(queue, t)
			}
		}
	}
	
	return result
}

func SCC(g graph.Graph, v graph.Vertex, t graph.Transport) ([]byte, int) {
	r0 := Reach(g, v, true,  t)
	r1 := Reach(g, v, false, t)
	scc := alg.Intersection(r0, r1)
	return scc, alg.Popcount(scc)
}

func RandomVertex(g graph.Graph) graph.Vertex {
	return graph.Vertex(rand.Intn(g.VertexCount()))
}

func LargeSCC(g graph.Graph, t graph.Transport) ([]byte, int) {
	maxSCC  := []byte(nil)
	maxSize := 0
	fmt.Printf("Computing a large SCC for t = %v\n", t)
	for i := 0; i < MaxTrials; i++ {
		scc, size := SCC(g, RandomVertex(g), t)
		if size > maxSize {
			maxSize = size
			maxSCC  = scc
			fmt.Printf("Found an SCC of size %v (frac: %.2f)\n",
				size, float64(size) / float64(g.VertexCount()))
			if size > int(MinSize * float64(g.VertexCount())) {
				return scc, size
			}
		}
	}
	return maxSCC, maxSize
}

func AccessibleRegion(g *graph.GraphFile) []byte {
	r := []byte(nil)
	for t := 0; t < int(graph.TransportMax); t++ {
		scc, _ := LargeSCC(g, graph.Transport(t))
		if r == nil {
			r = scc
		} else {
			r = alg.Union(r, scc)
		}
	}
	size := alg.Popcount(r)
	fmt.Printf("Accessible: %v (frac: %.2f)\n",
		size, float64(size) / float64(g.VertexCount()))
	return r
}

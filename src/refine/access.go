package main

import (
	"alg"
	"fmt"
	"graph"
	"math/rand"
)

const (
	MaxTrials = 10
	MinSize   = 0.5
)

func SanityCheck(g graph.Graph, mode graph.Transport) {
	outHistogram := alg.NewHistogram(fmt.Sprintf("out degrees %v", mode))
	inHistogram  := alg.NewHistogram(fmt.Sprintf("in degrees %v", mode))
	inEdgeCount  := 0
	outEdgeCount := 0
	edges := []graph.Edge(nil)
	for i := 0; i < g.VertexCount(); i++ {
		v := graph.Vertex(i)
		edges = g.VertexEdges(v, true, mode, edges)
		outDegree := len(edges)
		edges = g.VertexEdges(v, false, mode, edges)
		inDegree := len(edges)
		outHistogram.Add(fmt.Sprintf("%v", outDegree))
		inHistogram.Add(fmt.Sprintf("%v", inDegree))
		outEdgeCount += outDegree
		inEdgeCount  += inDegree
	}
	if inEdgeCount != outEdgeCount {
		fmt.Printf("Graph in/out edges are broken (t: %v):\n", mode)
		fmt.Printf(" - EdgeCount: %v\n", g.EdgeCount())
		fmt.Printf(" - Out edges: %v\n", outEdgeCount)
		fmt.Printf(" - In  edges: %v\n", inEdgeCount)
	}
	outHistogram.Print()
	inHistogram.Print()
}

func Reach(g graph.Graph, v graph.Vertex, forward bool, mode graph.Transport) []byte {
	result := make([]byte, (g.VertexCount() + 7) / 8)
	queue  := make([]graph.Vertex, 1, 128)
	alg.SetBit(result, uint(v))
	queue[0] = v
	
	edges := []graph.Edge(nil)
	
	for len(queue) > 0 {
		s := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		edges = g.VertexEdges(s, forward, mode, edges)
		for _, e := range edges {
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
			fmt.Printf(" - Found an SCC of size %v (frac: %.2f)\n",
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
		mode := graph.Transport(t)
		// SanityCheck(g, mode)
		scc, _ := LargeSCC(g, mode)
		g.Access[mode] = scc
		if r == nil {
			r = scc
		} else {
			r = alg.Union(r, scc)
		}
	}
	size := alg.Popcount(r)
	fmt.Printf("Accessible: %v (frac: %.2f, trash: %v)\n",
		size, float64(size) / float64(g.VertexCount()), g.VertexCount() - size)
	return r
}

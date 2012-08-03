package main

import (
	"alg"
	"fmt"
	"graph"
	"math/rand"
)

const (
	MaxTrials = 20
	MinSize   = 1 << 15 // minimum size of a SCC to be considered
)

func UndirectedSanityCheck(g *graph.GraphFile) {
	histogram := alg.NewHistogram("degrees")
	edgeCount := 0
	min, max := 100, 0
	edges := []graph.Edge(nil)
	for i := 0; i < g.VertexCount(); i++ {
		v := graph.Vertex(i)
		edges = g.VertexRawEdges(v, edges)
		histogram.Add(fmt.Sprintf("%v", len(edges)))
		edgeCount += len(edges)
		if len(edges) < min {
			min = len(edges)
		} else if len(edges) > max {
			max = len(edges)
		}
	}
	histogram.Print()
	fmt.Printf("\n")
	fmt.Printf("Base Graph:\n")
	fmt.Printf(" - |V| = %v\n", g.VertexCount())
	fmt.Printf(" - |E| = %v\n", edgeCount)
	fmt.Printf(" - average degree: %.2f\n", float64(edgeCount)/float64(g.VertexCount()))
	fmt.Printf(" - minimum degree: %v\n", min)
	fmt.Printf(" - maximum degree: %v\n", max)
}

func SanityCheck(g graph.Graph, mode graph.Transport) {
	outHistogram := alg.NewHistogram(fmt.Sprintf("out degrees %v", mode))
	inHistogram := alg.NewHistogram(fmt.Sprintf("in degrees %v", mode))
	inEdgeCount := 0
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
		inEdgeCount += inDegree
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
	result := make([]byte, (g.VertexCount()+7)/8)
	queue := make([]graph.Vertex, 1, 128)
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
	r0 := Reach(g, v, true, t)
	r1 := Reach(g, v, false, t)
	scc := alg.Intersection(r0, r1)
	return scc, alg.Popcount(scc)
}

// RandomVertex returns a random vertex that is not in the final graph yet
func RandomVertex(g graph.Graph, in []byte) graph.Vertex {
	for {
		r := rand.Intn(g.VertexCount())
		if !alg.GetBit(in, uint(r)) {
			return graph.Vertex(r)
		}
	}
	return -1
}

// LargeSCC returns the union of all large SCCs found
func LargeSCC(g graph.Graph, t graph.Transport) ([]byte, int) {
	in := make([]byte, g.VertexCount())
	totalSize := 0
	sccCount := 0
	fmt.Printf("Computing SCCs for t = %v\n", t)
	// Stops after MaxTrials unsuccessful trials or if it is not possible to find a SCC of sufficent size anymore
	for i := 0; i < MaxTrials && totalSize+MinSize <= g.VertexCount(); i++ {
		scc, size := SCC(g, RandomVertex(g, in), t)
		if size >= MinSize {
			i--
			in = alg.Union(in, scc)
			totalSize += size
			fmt.Printf(" - Found an SCC of size %v (frac: %.2f)\n",
				size, float64(size)/float64(g.VertexCount()))
			sccCount++
		}
	}
	fmt.Printf("Found %v SCCs with a minimum size of %v\n", sccCount, MinSize)
	return in, totalSize
}

func AccessibleRegion(g *graph.GraphFile) []byte {
	r := []byte(nil)
	UndirectedSanityCheck(g)
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
		size, float64(size)/float64(g.VertexCount()), g.VertexCount()-size)
	return r
}

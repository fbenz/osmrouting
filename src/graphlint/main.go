package main

import (
	"alg"
	"flag"
	"fmt"
	"os"
	"graph"
)

var (
	// command line flags
	InputFile    string
	InputCluster string
)

func init() {
	flag.StringVar(&InputFile, "i", "", "input graph file")
	flag.StringVar(&InputCluster, "ic", "", "input graph cluster")
}

func ValidateEdges(g graph.Graph) {
	file := g.(*graph.GraphFile)
	maxDegree := 0
	histogram := alg.NewHistogram("degrees")
	for i := 0; i < g.VertexCount(); i++ {
		v := graph.Vertex(i)
		
		if file.FirstOut[v] > file.FirstOut[v+1] {
			fmt.Printf("    FirstOut array is not monotone at i: %v.\n", v)
			panic("FirstOut array is broken.")
		}
		
		degree := int(file.FirstOut[v+1] - file.FirstOut[v])
		if degree > maxDegree {
			maxDegree = degree
		}
		histogram.Add(fmt.Sprintf("%v", degree))
		
		for j := file.FirstOut[v]; j < file.FirstOut[v+1]; j++ {
			e := graph.Edge(j)
			u := g.EdgeOpposite(e, v)
			if u < 0 || int(u) >= g.VertexCount() {
				fmt.Printf("    Wrong edge: edges[%v] = %v (= %v ^ [%v])\n", e, file.Edges[e], v, u)
				panic("Edges array is broken.")
			}
		}
	}

	if maxDegree > 255 {
		fmt.Printf("Out degree too high: %v\n", maxDegree)
		histogram.Print()
		panic("Out Degrees are broken?")
	}
}

func ValidateSteps(g graph.Graph) {
	file := g.(*graph.GraphFile)
	//histogram := alg.NewHistogram("step size")
	for i := 0; i < g.VertexCount(); i++ {
		v := graph.Vertex(i)
		for j := file.FirstOut[i]; j < file.FirstOut[i+1]; j++ {
			e := graph.Edge(j)
			stepLength := len(g.EdgeSteps(e, v))
			if stepLength > 2000 {
				panic("Open streetmap data is broken.")
			}
			//histogram.Add(fmt.Sprintf("%v", stepLength))
		}
	}
	//histogram.Print()
}

func main() {
	flag.Parse()
	if InputCluster == "" && InputFile == "" {
		flag.Usage()
		os.Exit(1)
	}
	
	if InputCluster != "" {
		println("Open cluster graph.")
		h, err := graph.OpenClusterGraph(InputCluster)
		if err != nil {
			println(err.Error())
			os.Exit(1)
		}
		
		println("Validate edges.")
		for i, g := range h.Cluster {
			fmt.Printf(" * Cluster %v/%v\n", i, len(h.Cluster))
			fmt.Printf("   MaxVertex: %v\n", g.VertexCount()-1)
			ValidateEdges(g)
			ValidateSteps(g)
		}
	} else {
		println("Open graph.")
		g, err := graph.OpenGraphFile(InputFile, false)
		if err != nil {
			println(err.Error())
			os.Exit(1)
		}
		
		println("Validate Edges.")
		fmt.Printf("    MaxVertex: %v\n", g.VertexCount()-1)
		ValidateEdges(g)
		println("Validate Steps.")
		ValidateSteps(g)
	}
}

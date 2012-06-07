package main

import (
	"fmt"
	"graph"
	"kdtree"
	"log"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(8)

	// just for know
	g, err := graph.Open("")
	if err != nil {
		log.Fatal("Loading graph:", err)
		return
	}
	fmt.Printf("Nodes: %v\n", g.NodeCount())
	kdTreeErr := kdtree.WriteKdTree(g.(graph.Positions))
	if kdTreeErr != nil {
		log.Fatal("Creating k-d tree:", kdTreeErr)
		return
	}
}

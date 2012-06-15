package main

import (
	"flag"
	"fmt"
	"graph"
	"kdtree"
	"log"
	"runtime"
)

var (
	FlagBaseDir string
)

func init() {
	flag.StringVar(&FlagBaseDir, "dir", "", "directory of the graph files")
}

func main() {
	runtime.GOMAXPROCS(8)

	// just for know
	g, err := graph.Open(FlagBaseDir)
	if err != nil {
		log.Fatal("Loading graph:", err)
		return
	}
	fmt.Printf("Nodes: %v\n", g.NodeCount())
	kdTreeErr := kdtree.WriteKdTree(FlagBaseDir, g.(graph.Positions))
	if kdTreeErr != nil {
		log.Fatal("Creating k-d tree:", kdTreeErr)
		return
	}
}

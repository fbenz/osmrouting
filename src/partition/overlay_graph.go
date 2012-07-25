package main

import (
	"fmt"
	"graph"
	"log"
	"path"
	"time"
)

func (pi *PartitionInfo) createOverlayGraph(g *graph.GraphFile, base string) {
	time1 := time.Now()

	vertexIndices := make([]int, g.VertexCount())
	vertexCount := 0
	for _, v := range pi.BorderVertices {
		for _, globalIndex := range v {
			vertexIndices[globalIndex] = vertexCount
			vertexCount++
		}
	}

	err := g.WriteSubgraph(path.Join(base, "/overlay"), vertexIndices, pi.Table)
	if err != nil {
		log.Fatal("Writing the overlay graph: ", err)
	}

	fmt.Printf("Overlay graph, vertex count %d\n", vertexCount)

	time2 := time.Now()
	fmt.Printf("Creating overlay graph: %v s\n", time2.Sub(time1).Seconds())
}

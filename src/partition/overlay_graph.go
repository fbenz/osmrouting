package main

import (
	"fmt"
	"graph"
	"log"
	"mm"
	"os"
	"path"
	"time"
)

func (pi *PartitionInfo) createOverlayGraph(g *graph.GraphFile, base string) {
	time1 := time.Now()

	dir := path.Join(base, "/overlay")
	err := os.Mkdir(dir, os.ModeDir|os.ModePerm)
	if err != nil {
		log.Fatal("Creating dir for overlay graph: ", err)
	}

	var partitions []uint16
	err = mm.Create(path.Join(dir, "partitions.ftf"), pi.Count+1, &partitions)
	if err != nil {
		log.Fatal("mm.Create failed: ", err)
	}

	vertexIndices := make([]int, g.VertexCount())
	for i := range vertexIndices {
		vertexIndices[i] = -1
	}
	vertexCount := 0
	partitions[0] = 0
	total := 0
	for i, v := range pi.BorderVertices {
		total += len(v)
		partitions[i+1] = uint16(total)
		for _, globalIndex := range v {
			vertexIndices[globalIndex] = vertexCount
			vertexCount++
		}
	}

	err = mm.Close(&partitions)
	if err != nil {
		log.Fatal("mm.Close failed: ", err)
	}

	err = g.WriteSubgraph(path.Join(base, "/overlay"), vertexIndices, pi.Table)
	if err != nil {
		log.Fatal("Writing the overlay graph: ", err)
	}

	fmt.Printf("Overlay graph, vertex count %d\n", vertexCount)

	time2 := time.Now()
	fmt.Printf("Creating overlay graph: %v s\n", time2.Sub(time1).Seconds())
}

// Writes the subgraphs resulting from the partitioning on disc

package main

import (
	"fmt"
	"graph"
	"log"
	"path"
	"time"
)

func (pi *PartitionInfo) createSubgraphs(g *graph.GraphFile, base string) {
	time1 := time.Now()

	vertexIndices := make([]int, g.VertexCount())
	for p := 0; p < pi.Count; p++ {
		// reset, -1 entries are excluded from the subgraph
		for i, _ := range vertexIndices {
			vertexIndices[i] = -1
		}

		// number the border vertices first
		for i, x := range pi.BorderVertices[p] {
			vertexIndices[x] = i
			pi.Table[x] = -1 // so that it is not considered again in the next loop
		}

		// then number all remaining vertices
		subVertexCount := len(pi.BorderVertices[p])
		for i := 0; i < g.VertexCount(); i++ {
			if pi.Table[i] == p { // and not border vertex, due to the -1 in the loop before
				vertexIndices[i] = subVertexCount
				subVertexCount++
			}
		}

		// restore pi.Table, remove the -1 for the border vertices
		for _, x := range pi.BorderVertices[p] {
			pi.Table[x] = p
		}

		// WriteSubgraph(base string, indices, partition []int) error {
		clusterString := fmt.Sprintf("/cluster%d.ftf", p+1)
		err := g.WriteSubgraph(path.Join(base, clusterString), vertexIndices, vertexIndices)
		if err != nil {
			log.Fatal("Writing the subgraph: ", err)
		}
	}

	time2 := time.Now()
	fmt.Printf("Creating subgraphs: %v s\n", time2.Sub(time1).Seconds())
}

package main

import (
	"fmt"
	"graph"
	"log"
	"mm"
	"time"
)

func (pi *PartitionInfo) createSubgraphs(g *graph.GraphFile) {
	time1 := time.Now()

	positions := g.RawPositions()
	distances := g.RawDistances()
	steps := g.RawSteps()
	stepPositions := g.RawStepPositions()

	// vertices (vertex id -> edge indices)
	// positions (coordinates of the vertices [2*id, 2*id+1])
	// edges (start vertex XOR end vertex)
	// distances (edge id -> distance)
	// steps (edge -> step indices)
	// step_positions (encoded steps)
	vertexIndices := make([]graph.Node, g.NodeCount())
	for p := 0; p < pi.Count; p++ {
		globalVertexIndices := make([]graph.Node, 0, int(U))
		for i, x := range pi.BorderVertices[p] {
			vertexIndices[x] = graph.Node(i)
			globalVertexIndices = append(globalVertexIndices, graph.Node(x))
			pi.Table[x] = -1 // so that it is not considered again in the next loop
		}
		subVertexCount := len(pi.BorderVertices[p])
		for i := 0; i < g.NodeCount(); i++ {
			if pi.Table[i] == p { // and not border vertex
				vertexIndices[i] = graph.Node(subVertexCount)
				subVertexCount++
				globalVertexIndices = append(globalVertexIndices, graph.Node(i))
			}
		}
		// restore pi.Table
		for _, x := range pi.BorderVertices[p] {
			pi.Table[x] = p
		}

		subVertexEdges := make([][]graph.Edge, subVertexCount)
		for i, _ := range subVertexEdges {
			subVertexEdges[i] = make([]graph.Edge, 0)
		}
		subEdgeCount := 0
		for i, gi := range globalVertexIndices {
			startEdge, endEdge := g.NodeEdges(gi)
			for j := startEdge; j <= endEdge; j++ {
				// exclude boundary edges
				if opposite := g.EdgeEndPoint(j); pi.Table[opposite] == p {
					subVertexEdges[i] = append(subVertexEdges[i], j)
					subEdgeCount++
				}
			}
		}

		var subVertices []uint32
		var subPositions []float64
		var subEdges []uint32
		var subDistances []float64 // TODO []uint16
		var subSteps []uint32

		partString := fmt.Sprintf(".part%d.ftf", p+1)
		err := mm.Create("vertices"+partString, subVertexCount+1, &subVertices)
		if err != nil {
			log.Fatal("mm.Create failed: ", err)
		}
		err = mm.Create("positions"+partString, 2*subVertexCount, &subPositions)
		if err != nil {
			log.Fatal("mm.Create failed: ", err)
		}
		err = mm.Create("edges"+partString, subEdgeCount, &subEdges)
		if err != nil {
			log.Fatal("mm.Create failed: ", err)
		}
		err = mm.Create("distances"+partString, subEdgeCount, &subDistances)
		if err != nil {
			log.Fatal("mm.Create failed: ", err)
		}
		err = mm.Create("steps"+partString, subEdgeCount+1, &subSteps)
		if err != nil {
			log.Fatal("mm.Create failed: ", err)
		}

		var c uint32 = 0
		subVertices[0] = c
		for i, gi := range globalVertexIndices {
			c += uint32(len(subVertexEdges[i]))
			subVertices[i+1] = c
			subPositions[2*i] = positions[2*gi]
			subPositions[2*i+1] = positions[2*gi+1]
		}

		subStepPositions := make([]float64, 0, 2*4*subEdgeCount) // TODO byte
		subSteps[0] = 0
		j := 0
		for /*vertex*/ _, edges := range subVertexEdges {
			for _, e := range edges {
				subEdges[j] = uint32(g.EdgeEndPoint(e)) //uint32(graph.Node(vertex) ^ g.EdgeEndPoint(e)) // TODO opposite
				subDistances[j] = distances[e]
				subSteps[j+1] = c
				c += steps[e+1] - steps[e]
				subStepPositions = append(subStepPositions, stepPositions[steps[e]:steps[e+1]]...)
				j++
			}
		}

		err = mm.Close(&subVertices)
		if err != nil {
			log.Fatal("mm.Close failed: ", err)
		}
		err = mm.Close(&subPositions)
		if err != nil {
			log.Fatal("mm.Close failed: ", err)
		}
		err = mm.Close(&subEdges)
		if err != nil {
			log.Fatal("mm.Close failed: ", err)
		}
		err = mm.Close(&subDistances)
		if err != nil {
			log.Fatal("mm.Close failed: ", err)
		}
		err = mm.Close(&subSteps)
		if err != nil {
			log.Fatal("mm.Close failed: ", err)
		}

		var subStepPositionsFinal []float64
		err = mm.Create("step_positions"+partString, len(subStepPositions), &subStepPositionsFinal)
		if err != nil {
			log.Fatal("mm.Create failed: ", err)
		}
		copy(subStepPositionsFinal, subStepPositions)
		err = mm.Close(&subStepPositionsFinal)
		if err != nil {
			log.Fatal("mm.Close failed: ", err)
		}
	}

	time2 := time.Now()
	fmt.Printf("Creating subgraphs: %v s\n", time2.Sub(time1).Seconds())
}

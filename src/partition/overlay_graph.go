package main

import (
	"fmt"
	"graph"
	"log"
	"mm"
	"time"
)

func (pi *PartitionInfo) createOverlayGraph(g *graph.GraphFile) {
	time1 := time.Now()

	positions := g.RawPositions()
	distances := g.RawDistances()
	steps := g.RawSteps()
	stepPositions := g.RawStepPositions()

	var overlayPartitions []uint16
	partString := ".overlay.ftf"
	err := mm.Create("partitions"+partString, pi.Count+1, &overlayPartitions)
	if err != nil {
		log.Fatal("mm.Create failed: ", err)
	}

	globalVertexIndices := make([]graph.Node, 0, 100*pi.Count)

	vertexCount := 0
	overlayPartitions[0] = 0
	for i, v := range pi.BorderVertices {
		overlayPartitions[i+1] = uint16(len(v))
		vertexCount += len(v)
		for _, gi := range v {
			globalVertexIndices = append(globalVertexIndices, gi)
		}
	}

	overlayVertexEdges := make([][]graph.Edge, vertexCount)
	for i, _ := range overlayVertexEdges {
		overlayVertexEdges[i] = make([]graph.Edge, 0)
	}
	edgeCount := 0
	for i, gi := range globalVertexIndices {
		startEdge, endEdge := g.NodeEdges(gi)
		for j := startEdge; j <= endEdge; j++ {
			// only boundary edges
			if opposite := g.EdgeEndPoint(j); pi.Table[opposite] != pi.Table[gi] {
				overlayVertexEdges[i] = append(overlayVertexEdges[i], j)
				edgeCount++
			}
		}
	}

	fmt.Printf("Overlay graph, vertex count %d, edge count %d\n", vertexCount, edgeCount)

	var overlayVertices []uint32
	var overlayPositions []float64
	var overlayEdges []uint32
	var overlayDistances []float64 // TODO []uint16
	var overlaySteps []uint32

	err = mm.Create("vertices"+partString, vertexCount+1, &overlayVertices)
	if err != nil {
		log.Fatal("mm.Create failed: ", err)
	}
	err = mm.Create("positions"+partString, 2*vertexCount, &overlayPositions)
	if err != nil {
		log.Fatal("mm.Create failed: ", err)
	}
	err = mm.Create("edges"+partString, edgeCount, &overlayEdges)
	if err != nil {
		log.Fatal("mm.Create failed: ", err)
	}
	err = mm.Create("distances"+partString, edgeCount, &overlayDistances)
	if err != nil {
		log.Fatal("mm.Create failed: ", err)
	}
	err = mm.Create("steps"+partString, edgeCount+1, &overlaySteps)
	if err != nil {
		log.Fatal("mm.Create failed: ", err)
	}

	var c uint32 = 0
	overlayVertices[0] = c
	for i, gi := range globalVertexIndices {
		c += uint32(len(overlayVertexEdges[i]))
		overlayVertices[i+1] = c
		overlayPositions[2*i] = positions[2*gi]
		overlayPositions[2*i+1] = positions[2*gi+1]
	}

	overlayStepPositions := make([]float64, 0, 2*4*edgeCount) // TODO byte
	overlaySteps[0] = 0
	j := 0
	for /*vertex*/ _, edges := range overlayVertexEdges {
		for _, e := range edges {
			overlayEdges[j] = uint32(g.EdgeEndPoint(e)) //uint32(graph.Node(vertex) ^ g.EdgeEndPoint(e)) // TODO opposite
			overlayDistances[j] = distances[e]
			overlaySteps[j+1] = c
			c += steps[e+1] - steps[e]
			overlayStepPositions = append(overlayStepPositions, stepPositions[steps[e]:steps[e+1]]...)
			j++
		}
	}

	err = mm.Close(&overlayPartitions)
	if err != nil {
		log.Fatal("mm.Close failed: ", err)
	}
	err = mm.Close(&overlayVertices)
	if err != nil {
		log.Fatal("mm.Close failed: ", err)
	}
	err = mm.Close(&overlayPositions)
	if err != nil {
		log.Fatal("mm.Close failed: ", err)
	}
	err = mm.Close(&overlayEdges)
	if err != nil {
		log.Fatal("mm.Close failed: ", err)
	}
	err = mm.Close(&overlayDistances)
	if err != nil {
		log.Fatal("mm.Close failed: ", err)
	}
	err = mm.Close(&overlaySteps)
	if err != nil {
		log.Fatal("mm.Close failed: ", err)
	}

	var overlayStepPositionsFinal []float64
	err = mm.Create("step_positions"+partString, len(overlayStepPositions), &overlayStepPositionsFinal)
	if err != nil {
		log.Fatal("mm.Create failed: ", err)
	}
	copy(overlayStepPositionsFinal, overlayStepPositions)
	err = mm.Close(&overlayStepPositionsFinal)
	if err != nil {
		log.Fatal("mm.Close failed: ", err)
	}

	time2 := time.Now()
	fmt.Printf("Creating overlay graph: %v s\n", time2.Sub(time1).Seconds())
}

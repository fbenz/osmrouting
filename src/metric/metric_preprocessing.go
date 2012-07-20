// This version of the preprocessing compiles, but is not complete as the
// new graph is missing.

package main

import (
	"flag"
	"fmt"
	"log"
	"mm"
	"time"
)

var (
	FlagMetric int
)

func init() {
	flag.IntVar(&FlagMetric, "metric", -1, "restrict the preprocessing to one metric; -1 means all metrics")
}

func main() {
	fmt.Printf("Metric preprocessing\n")

	// TODO load graphs

	if FlagMetric >= 0 {
		if FlagMetric >= MetricMax {
			log.Fatal("metric index is too large: ", FlagMetric)
		}
		// computeMatrices(overlay, subgraphs, FlagMetric)
	} else {
		// preprocessAll(overlay, subgraphs)
	}
}

func preprocessAll(overlay OverlayGraph, subgraphs []Graph) {
	for i := 0; i < MetricMax; i++ {
		computeMatrices(overlay, subgraphs, i)
	}
}

func computeMatrices(overlay OverlayGraph, subgraphs []Graph, metric int) {
	time1 := time.Now()

	// compute the matrices for all subgraphs
	matrices := make([][][]float32, len(subgraphs))
	size := 0
	for p, g := range subgraphs {
		boundaryVertexCount := overlay.PartitionSize(p)
		matrices[p] = computeMatrix(g, boundaryVertexCount, metric)
		size += boundaryVertexCount * boundaryVertexCount
	}

	// write all matrices in row-major layout in one file (sorted by partition ID)
	var matrixFile []float32
	partString := fmt.Sprintf(".metric%d.ftf", metric+1)
	err := mm.Create("matrices"+partString, size, &matrixFile)
	if err != nil {
		log.Fatal("mm.Create failed: ", err)
	}
	// iterate over all rows
	pos := 0
	for _, matrix := range matrices {
		for _, row := range matrix {
			copy(matrixFile[pos:len(row)], row)
			pos += len(row)
		}
	}
	err = mm.Close(&matrixFile)
	if err != nil {
		log.Fatal("mm.Close failed: ", err)
	}

	time2 := time.Now()
	fmt.Printf("Preprocessing time for metric %d: %v s\n", metric, time2.Sub(time1).Seconds())
}

func computeMatrix(subgraph Graph, boundaryVertexCount, metric int) [][]float32 {
	// TODO precompute the result of the metric for every edge and store the result for the graph
	// An alternative would be an computation on-the-fly during each run of Dijkstra (preprocessing here + live query)
	for i := 0; i < subgraph.EdgeCount(); i++ {
		// apply metric on edge weight and possibly other data
	}

	matrix := make([][]float32, boundaryVertexCount)

	// Boundary vertices always have the lowest IDs. Therefore, iterating from 0 to boundaryVertexCount-1 is possible here.
	// In addition, the simple copy from the distance array returned from Dijkstra's algorithm is possible.
	for i, _ := range matrix {
		// run Dijkstra starting at vertex i with the given metric
		d := dummyDijkstra(i, metric)
		matrix[i] = d[:boundaryVertexCount]
	}

	return matrix
}

// TODO replace
func dummyDijkstra(start, metric int) []float32 {
	d := make([]float32, 1000)
	return d
}

// This version of the preprocessing compiles, but is not complete as the
// new graph is missing.

package main

import (
	"flag"
	"fmt"
	"graph"
	"log"
	"mm"
	"path"
	"route"
	"runtime"
	"time"
)

const (
	MaxThreads = 8
)

var (
	FlagBaseDir string
	FlagMetric  int
)

func init() {
	flag.StringVar(&FlagBaseDir, "dir", "", "directory of the graph")
	flag.IntVar(&FlagMetric, "metric", -1, "restrict the preprocessing to one metric; -1 means all metrics")
}

func main() {
	runtime.GOMAXPROCS(MaxThreads)
	flag.Parse()

	fmt.Printf("Metric preprocessing\n")

	clusterGraph, err := graph.OpenClusterGraph(FlagBaseDir)
	if err != nil {
		log.Fatal("Open cluster graph: ", err)
	}

	if FlagMetric >= 0 {
		if FlagMetric >= int(graph.MetricMax) {
			log.Fatal("metric index is too large: ", FlagMetric)
		}
		preprocessOne(clusterGraph, FlagMetric)
	} else {
		preprocessAll(clusterGraph)
	}
}

// preprocessAll computes the metric matrices for all metrics
func preprocessAll(g *graph.ClusterGraph) {
	for i := 0; i < int(graph.MetricMax); i++ {
		preprocessOne(g, i)
	}
}

// preprocessOne computes the metric matrices for one metrics
func preprocessOne(g *graph.ClusterGraph, metric int) {
	for i := 0; i < int(graph.TransportMax); i++ {
		computeMatrices(g, metric, i)
	}
}

// computeMatrices computes the metric matrices for the given metric and transport mode
func computeMatrices(g *graph.ClusterGraph, metric, trans int) {
	time1 := time.Now()

	// compute the matrices for all Cluster
	matrices := make([][][]float32, len(g.Cluster))
	size := 0
	for p, subgraph := range g.Cluster {
		boundaryVertexCount := g.Overlay.ClusterSize(p)
		matrices[p] = computeMatrix(subgraph, boundaryVertexCount, metric, trans)
		size += boundaryVertexCount * boundaryVertexCount
	}

	// write all matrices in row-major layout in one file (sorted by partition ID)
	var matrixFile []float32
	fileName := fmt.Sprintf("matrices.trans%d.metric%d.ftf", trans+1, metric+1)
	err := mm.Create(path.Join(FlagBaseDir, fileName), size, &matrixFile)
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

// computeMatrix computes the metric matrix for the given subgraph and metric
func computeMatrix(subgraph graph.Graph, boundaryVertexCount, metric, trans int) [][]float32 {
	// TODO precompute the result of the metric for every edge and store the result for the graph
	// An alternative would be an computation on-the-fly during each run of Dijkstra (preprocessing here + live query)
	//for i := 0; i < subgraph.EdgeCount(); i++ {
	// apply metric on edge weight and possibly other data
	//}

	matrix := make([][]float32, boundaryVertexCount)

	// Boundary vertices always have the lowest IDs. Therefore, iterating from 0 to boundaryVertexCount-1 is possible here.
	// In addition, only the first elements returned from Dijkstra's algorithm have to be considered.
	for i, _ := range matrix {
		// run Dijkstra starting at vertex i with the given metric
		vertex := graph.Vertex(i)
		s := make([]graph.Way, 1)
		target := subgraph.VertexCoordinate(vertex)
		s[0] = graph.Way{Length: 0, Vertex: vertex, Steps: nil, Target: target}

		elements := route.DijkstraComplete(subgraph, s, graph.Metric(metric), graph.Transport(trans), true /* forward */)
		for j, elem := range elements[:boundaryVertexCount] {
			matrix[i][j] = elem.Weight()
		}
	}

	return matrix
}

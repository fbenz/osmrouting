// This version of the preprocessing compiles, but is not complete as the
// new graph is missing.

package main

import (
	"flag"
	"fmt"
	"graph"
	"log"
	"math"
	"mm"
	"os"
	"path"
	"route"
	"runtime"
	"runtime/pprof"
	"time"
)

const (
	MaxThreads = 8
)

var (
	FlagBaseDir    string
	FlagCpuProfile string
	FlagMetric     int
)

type Job struct {
	Graph         *graph.ClusterGraph
	Matrices      [][]float32
	Start, Stride int
	Transport     graph.Transport
	Metric        graph.Metric
}

func init() {
	flag.StringVar(&FlagBaseDir, "dir", "", "directory of the graph")
	flag.StringVar(&FlagCpuProfile, "cpuprofile", "", "write cpu profile to file")
	flag.IntVar(&FlagMetric, "metric", -1, "restrict the preprocessing to one metric; -1 means all metrics")
}

func main() {
	runtime.GOMAXPROCS(MaxThreads)
	flag.Parse()
	
	if FlagCpuProfile != "" {
		f, err := os.Create(FlagCpuProfile + ".pprof")
		if err != nil {
			println("Unable to open cpuprofile:", err.Error())
			os.Exit(1)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	fmt.Printf("Metric preprocessing\n")

	clusterGraph, err := graph.OpenClusterGraph(FlagBaseDir, false /* loadMatrices */)
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

	// compute the matrices for all Clusters
	matrices := make([][]float32, len(g.Cluster))
	ready := make(chan int, MaxThreads)
	for i := 0; i < MaxThreads; i++ {
		job := &Job{
			Graph:     g,
			Matrices:  matrices,
			Start:     i,
			Stride:    MaxThreads,
			Transport: graph.Transport(trans),
			Metric:    graph.Metric(metric),
		}
		go computeMatrixThreadRouter(ready, job)
	}
	for i := 0; i < MaxThreads; i++ {
		<-ready
	}

	// Compute the size of the complete file.
	size := 0
	for _, m := range matrices {
		size += len(m)
	}

	// write all matrices in row-major layout in one file (sorted by partition ID)
	var matrixFile []float32
	fileName := fmt.Sprintf("matrices.trans%d.metric%d.ftf", trans+1, metric+1)
	err := mm.Create(path.Join(FlagBaseDir, fileName), size, &matrixFile)
	if err != nil {
		log.Fatal("mm.Create failed: ", err)
	}
	// iterate over all matrices
	pos := 0
	for _, matrix := range matrices {
		copy(matrixFile[pos:], matrix)
		pos += len(matrix)
	}
	err = mm.Close(&matrixFile)
	if err != nil {
		log.Fatal("mm.Close failed: ", err)
	}

	time2 := time.Now()
	fmt.Printf("Preprocessing time for metric %d: %v s\n", metric, time2.Sub(time1).Seconds())
}

func computeMatrixThread(ready chan<- int, job *Job) {
	g := job.Graph
	metric := job.Metric
	trans := job.Transport
	for i := job.Start; i < len(g.Cluster); i += job.Stride {
		//fmt.Printf("%v, %v, Cluster: %v\n", metric, trans, i+1)
		boundaryVertexCount := g.Overlay.ClusterSize(i)
		job.Matrices[i] = computeMatrix(g.Cluster[i], boundaryVertexCount, int(metric), int(trans))
	}
	ready <- 1
}

// computeMatrix computes the metric matrix for the given subgraph and metric
func computeMatrix(subgraph graph.Graph, boundaryVertexCount, metric, trans int) []float32 {
	// TODO precompute the result of the metric for every edge and store the result for the graph
	// An alternative would be an computation on-the-fly during each run of Dijkstra (preprocessing here + live query)
	//for i := 0; i < subgraph.EdgeCount(); i++ {
	// apply metric on edge weight and possibly other data
	//}
	
	if boundaryVertexCount > subgraph.VertexCount() {
		log.Fatalf("Wrong boundaryVertexCount: %v > %v",
			boundaryVertexCount, subgraph.VertexCount())
	}

	matrix := make([]float32, boundaryVertexCount * boundaryVertexCount)

	// Boundary vertices always have the lowest IDs. Therefore, iterating from 0 to boundaryVertexCount-1 is possible here.
	// In addition, only the first elements returned from Dijkstra's algorithm have to be considered.
	for i := 0; i < boundaryVertexCount; i++ {
		// run Dijkstra starting at vertex i with the given metric
		vertex := graph.Vertex(i)
		s := make([]graph.Way, 1)
		target := subgraph.VertexCoordinate(vertex)
		s[0] = graph.Way{Length: 0, Vertex: vertex, Steps: nil, Target: target}

		elements := route.DijkstraComplete(subgraph, s, graph.Metric(metric), graph.Transport(trans), true /* forward */)
		for j, elem := range elements[:boundaryVertexCount] {
			if elem != nil {
				matrix[boundaryVertexCount * i + j] = elem.Weight()
			} else {
				matrix[boundaryVertexCount * i + j] = float32(math.Inf(1))
			}
		}
	}

	return matrix
}

func computeMatrixThreadRouter(ready chan<- int, job *Job) {
	g := job.Graph
	router := &route.Router{
		Forward:   true,
		Transport: job.Transport,
		Metric:    job.Metric,
	}
	
	for i := job.Start; i < len(g.Cluster); i += job.Stride {
		boundaryVertexCount := g.Overlay.ClusterSize(i)
		job.Matrices[i] = computeMatrixRouter(router, g.Cluster[i], boundaryVertexCount)
	}
	
	ready <- 1
}

// computeMatrix computes the metric matrix for the given subgraph and metric
func computeMatrixRouter(router *route.Router, g graph.Graph, boundaryVertexCount int) []float32 {
	if boundaryVertexCount > g.VertexCount() {
		log.Fatalf("Wrong boundaryVertexCount: %v > %v",
			boundaryVertexCount, g.VertexCount())
	}
	
	if boundaryVertexCount == 0 {
		log.Printf("Empty Cluster")
		return nil
	}

	matrix := make([]float32, boundaryVertexCount * boundaryVertexCount)

	// Boundary vertices always have the lowest IDs. Therefore, iterating from 0 to boundaryVertexCount-1 is possible here.
	// In addition, only the first elements returned from Dijkstra's algorithm have to be considered.
	//pathLen := 0
	for i := 0; i < boundaryVertexCount; i++ {
		// run Dijkstra starting at vertex i with the given metric
		router.Reset(g)
		router.AddSource(graph.Vertex(i), 0)
		//println("router.Run()")
		router.Run()
		if ok, err := router.CertifySolution(); !ok {
			log.Fatalf(err.Error())
		}
		
		for j := 0; j < boundaryVertexCount; j++ {
			v := graph.Vertex(j)
			index := boundaryVertexCount * i + j
			matrix[index] = router.Distance(v)
			//if router.Reachable(v) {
			//	vs, _ := router.Path(v)
			//	pathLen += len(vs)
			//}
		}
	}

	//fmt.Printf("Average path length: %v\n", float64(pathLen) / float64(len(matrix)))

	return matrix
}

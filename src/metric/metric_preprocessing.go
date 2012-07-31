// This version of the preprocessing compiles, but is not complete as the
// new graph is missing.

package main

import (
	"flag"
	"fmt"
	"graph"
	"log"
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

func computeMatrixThreadRouter(ready chan<- int, job *Job) {
	g := job.Graph
	router := &route.Router{
		Forward:   true,
		Transport: job.Transport,
		Metric:    job.Metric,
	}

	queries  := 0
	duration := time.Duration(0)
	
	for i := job.Start; i < len(g.Cluster); i += job.Stride {
		boundaryVertexCount := g.Overlay.ClusterSize(i)
		matrix, diff := computeMatrixRouter(router, g.Cluster[i], boundaryVertexCount)
		job.Matrices[i] = matrix
		duration += diff
		queries  += boundaryVertexCount
	}
	
	log.Printf("Average Querytime: %.2f ms\n", float64(duration) / float64(time.Duration(queries) * time.Millisecond))
	
	ready <- 1
}

// computeMatrix computes the metric matrix for the given subgraph and metric
func computeMatrixRouter(router *route.Router, g graph.Graph, boundaryVertexCount int) ([]float32, time.Duration) {
	if boundaryVertexCount > g.VertexCount() {
		log.Fatalf("Wrong boundaryVertexCount: %v > %v",
			boundaryVertexCount, g.VertexCount())
	}
	
	if boundaryVertexCount == 0 {
		log.Printf("Empty Cluster")
		return nil, time.Duration(0)
	}

	matrix := make([]float32, boundaryVertexCount * boundaryVertexCount)
	
	//bidirouter := &route.BidiRouter{
	//	Metric:    router.Metric,
	//	Transport: router.Transport,
	//}

	// Boundary vertices always have the lowest IDs. Therefore, iterating from 0 to boundaryVertexCount-1 is possible here.
	// In addition, only the first elements returned from Dijkstra's algorithm have to be considered.
	//pathLen := 0
	uniduration := time.Duration(0)
	//duration := time.Duration(0)
	for i := 0; i < boundaryVertexCount; i++ {
		// run Dijkstra starting at vertex i with the given metric
		t1 := time.Now()
		router.Reset(g)
		router.AddSource(graph.Vertex(i), 0)
		router.Run()
		uniduration += time.Since(t1)
		//if ok, err := router.CertifySolution(); !ok {
		//	log.Fatalf(err.Error())
		//}
		
		for j := 0; j < boundaryVertexCount; j++ {
			v := graph.Vertex(j)
			index := boundaryVertexCount * i + j
			matrix[index] = router.Distance(v)
			
			/*
			t1 := time.Now()
			bidirouter.Reset(g)
			bidirouter.AddSource(graph.Vertex(i), 0)
			bidirouter.AddTarget(graph.Vertex(j), 0)
			bidirouter.Run()
			duration += time.Since(t1)
			
			// Since floating point addition is not associative, we can expect some
			// round off error. The relative error should be within epsilon, though.
			dist := bidirouter.Distance()
			if math.Abs(float64(dist - matrix[index]) / float64(dist)) > 4.88e-04 {
			//if dist != matrix[index] {
				log.Fatalf("Bug in Bidirouter: Distance %v should be %v.\n",
					dist, matrix[index])
			}
			*/
			
			//if router.Reachable(v) {
			//	vs, _ := router.Path(v)
			//	pathLen += len(vs)
			//}
		}
	}
	
	//queries := time.Duration(boundaryVertexCount)
	//log.Printf("Average Querytime Uni:  %.2f ms\n", float64(uniduration) / float64(queries * time.Millisecond))
	//queries = time.Duration(boundaryVertexCount * boundaryVertexCount)
	//log.Printf("Average Querytime Bidi: %.2f ms\n", float64(duration) / float64(queries * time.Millisecond))

	//fmt.Printf("Average path length: %v\n", float64(pathLen) / float64(len(matrix)))

	return matrix, uniduration
}

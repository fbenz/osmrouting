// Graph partitioning using Metis

package main

import (
	"bufio"
	"flag"
	"fmt"
	"graph"
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"time"
)

const (
	MetisGraphFile = "graph.txt"
	Ufactor        = 1.03
)

type PartitionInfo struct {
	Count          int
	Table          []int            // global vertex id -> partition number
	BorderTable    []int            // global vertex id -> vertex id in cluster
	BorderVertices [][]graph.Vertex // partition id -> boundary vertices
}

var (
	U = math.Pow(2, 15)

	FlagBaseDir string
)

func init() {
	flag.StringVar(&FlagBaseDir, "dir", "", "directory of the graph")
}

func main() {
	flag.Parse()

	g, err := graph.OpenGraphFile(FlagBaseDir, false /* ignoreErrors */)
	if err != nil {
		log.Fatal("Loading graph: ", err)
	}

	partitionCount := partitionCount(g.VertexCount(), U)
	pi := &PartitionInfo{Count: partitionCount}

	pi.metisPartitioning(g)
	pi.createSubgraphs(g, FlagBaseDir)
	pi.createOverlayGraph(g, FlagBaseDir)
}

func (pi *PartitionInfo) metisPartitioning(g *graph.GraphFile) {
	time1 := time.Now()

	fmt.Printf("Number of partitions: %d\n", pi.Count)

	out, outErr := os.Create(MetisGraphFile)
	if outErr != nil {
		fmt.Printf("failed to create file\n")
		return
	}
	output := bufio.NewWriter(out)

	fmt.Printf("Size %d %d\n", g.VertexCount(), g.EdgeCount()/2)
	fmt.Fprintf(output, "%d %d\n", g.VertexCount(), g.EdgeCount()/2)

	edges := []graph.Edge(nil)
	for i := 0; i < g.VertexCount(); i++ {
		vertex := graph.Vertex(i)
		edges = g.VertexRawEdges(vertex, edges)
		for _, e := range edges {
			opposite := g.EdgeOpposite(e, vertex)
			fmt.Fprintf(output, "%v ", opposite+1)
		}
		fmt.Fprintf(output, "\n")
	}
	output.Flush()
	out.Close()
	time2 := time.Now()
	fmt.Printf("Writing Metis file: %v s\n", time2.Sub(time1).Seconds())

	// run Metis
	cmd := exec.Command("./gpmetis" /* other option go here, like -niter=10 -ncuts=1 */, MetisGraphFile, strconv.Itoa(pi.Count))
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	time3 := time.Now()
	fmt.Printf("Metis: %v s\n", time3.Sub(time2).Seconds())

	// read output of Metis
	metisOutputName := fmt.Sprintf("%s.part.%d", MetisGraphFile, pi.Count)
	in, inErr := os.Open(metisOutputName)
	if inErr != nil {
		fmt.Printf("failed to open file\n")
		return
	}
	input := bufio.NewReader(in)
	pi.Table = make([]int, g.VertexCount())
	for i, _ := range pi.Table {
		p := -1
		_, readErr := fmt.Fscanf(input, "%d\n", &p)
		if readErr != nil {
			log.Fatal(readErr)
		}
		pi.Table[i] = p
	}
	in.Close()
	time4 := time.Now()
	fmt.Printf("Reading Metis file: %v s\n", time4.Sub(time3).Seconds())

	// remove both files
	os.Remove(MetisGraphFile)
	os.Remove(metisOutputName)

	// determine border vertices
	// here, initially pi.BorderTable maps border vertices to their partition
	crossEdges := 0
	pi.BorderTable = make([]int, g.VertexCount())
	for i, _ := range pi.BorderTable {
		pi.BorderTable[i] = -1
	}

	for i := 0; i < g.VertexCount(); i++ {
		vertex := graph.Vertex(i)
		edges = g.VertexRawEdges(vertex, edges)
		for _, e := range edges {
			sp := vertex
			ep := g.EdgeOpposite(e, vertex)
			if pi.Table[sp] != pi.Table[ep] {
				pi.BorderTable[sp] = pi.Table[sp]
				pi.BorderTable[ep] = pi.Table[ep]
				crossEdges++ // not needed
			}
		}
	}
	fmt.Printf("Cross edges %d\n", crossEdges/2)

	// collect border vertices
	// now, pi.BorderTable is changed so that it maps border vertices to their index in the cluster
	pi.BorderVertices = make([][]graph.Vertex, pi.Count)
	for i, _ := range pi.BorderVertices {
		pi.BorderVertices[i] = make([]graph.Vertex, 0, 200)
	}
	for i, _ := range pi.BorderTable {
		if pi.BorderTable[i] != -1 {
			p := pi.Table[i]
			pi.BorderVertices[p] = append(pi.BorderVertices[p], graph.Vertex(i))
			pi.BorderTable[i] = len(pi.BorderVertices[p]) - 1
		}
	}

	time5 := time.Now()
	fmt.Printf("Collecting border vertices: %v s\n", time5.Sub(time4).Seconds())

	// just statistics
	minPartSize := g.VertexCount()
	maxPartSize := 0
	for p := 0; p < pi.Count; p++ {
		curSize := 0
		for i := 0; i < g.VertexCount(); i++ {
			if pi.Table[i] == p {
				curSize++
			}
		}
		if curSize < minPartSize {
			minPartSize = curSize
		}
		if curSize > maxPartSize {
			maxPartSize = curSize
		}
	}
	fmt.Printf("Partition sizes, min: %d, max: %d (U = %v)\n", minPartSize, maxPartSize, U)
}

func partitionCount(nodes int, U float64) int {
	return int(math.Ceil(float64(nodes) / U / Ufactor))
}

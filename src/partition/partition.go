// TODO
// At the moment the partitioning works on the old graph interface, but
// it is build in a way such that only a few changes are needed so that
// it works with the new one.
//
// corner cases:
// - start and/or endpoint is on a boundary edge

package main

import (
	//"alg"
	"bufio"
	"flag"
	"fmt"
	"graph"
	"log"
	"math"
	//"mm"
	"os"
	"os/exec"
	"strconv"
	"time"
)

const (
	TravelmodeCar  = "driving"
	TravelmodeFoot = "walking"
	TravelmodeBike = "bicycling"

	MetisGraphFile = "graph.txt"
	Ufactor        = 1.03
)

type RoutingData struct {
	graph *graph.GraphFile
}

type PartitionInfo struct {
	Count          int
	Table          []int          // global vertex id -> partition number
	BorderTable    []int          // global vertex id -> vertex id in cluster
	BorderVertices [][]graph.Node // partition id -> boundary vertices
}

var (
	U = math.Pow(2, 15)

	osmData  map[string]RoutingData
	FlagMode string
)

func init() {
	flag.StringVar(&FlagMode, "mode", "driving", "travel mode")
}

func main() {
	flag.Parse()

	if err := setup(); err != nil {
		log.Fatal("Setup failed:", err)
	}

	g := osmData[FlagMode].graph
	partitionCount := partitionCount(g.NodeCount(), U)
	pi := &PartitionInfo{Count: partitionCount}

	pi.metisPartitioning(g)
	pi.createSubgraphs(g)
	pi.createOverlayGraph(g)
}

func loadFiles(base string) (*RoutingData, error) {
	g, err := graph.OpenGraphFile(base)
	if err != nil {
		log.Fatal("Loading graph: ", err)
		return nil, err
	}
	return &RoutingData{g}, nil
}

// setup from the HTTP server
func setup() error {
	osmData = map[string]RoutingData{}

	dat, err := loadFiles("car")
	if err != nil {
		return err
	}
	osmData["driving"] = *dat

	// TODO add this back or change the graph
	/*dat, err = loadFiles("bike")
	if err != nil {
		return err
	}
	osmData["bicycling"] = *dat

	dat, err = loadFiles("foot")
	if err != nil {
		return err
	}
	osmData["walking"] = *dat*/

	return nil
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

	fmt.Printf("Size %d %d\n", g.NodeCount(), g.EdgeCount()/2)
	fmt.Fprintf(output, "%d %d\n", g.NodeCount(), g.EdgeCount()/2)
	for i := 0; i < g.NodeCount(); i++ {
		start, end := g.NodeEdges(graph.Node(i))

		for j := start; j <= end; j++ {
			_, exists := g.EdgeReverse(graph.Edge(j))
			if !exists {
				panic("directed edge detected")
			}
			endpoint := g.EdgeEndPoint(graph.Edge(j))
			if endpoint == graph.Node(i) {
				endpoint = g.EdgeStartPoint(graph.Edge(j))
			}
			fmt.Fprintf(output, "%v ", endpoint+1)
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
	pi.Table = make([]int, g.NodeCount())
	for i := 0; i < g.NodeCount(); i++ {
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
	pi.BorderTable = make([]int, g.NodeCount())
	for i, _ := range pi.BorderTable {
		pi.BorderTable[i] = -1
	}
	for i := 0; i < g.EdgeCount(); i++ {
		edge := graph.Edge(i)
		sp := g.EdgeStartPoint(edge)
		ep := g.EdgeEndPoint(edge)
		if pi.Table[sp] != pi.Table[ep] {
			pi.BorderTable[sp] = pi.Table[sp]
			pi.BorderTable[ep] = pi.Table[ep]
			crossEdges++ // not needed
		}
	}
	fmt.Printf("Cross edges %d\n", crossEdges/2)

	// collect border vertices
	// now, pi.BorderTable is changed so that it maps border vertices to their index in the cluster
	pi.BorderVertices = make([][]graph.Node, pi.Count)
	for p := 0; p < pi.Count; p++ {
		pi.BorderVertices[p] = make([]graph.Node, 0, 200)
	}
	for i := 0; i < g.NodeCount(); i++ {
		if pi.BorderTable[i] != -1 {
			p := pi.Table[i]
			pi.BorderVertices[p] = append(pi.BorderVertices[p], graph.Node(i))
			pi.BorderTable[i] = len(pi.BorderVertices[p]) - 1
		}
	}

	time5 := time.Now()
	fmt.Printf("Collecting border vertices: %v s\n", time5.Sub(time4).Seconds())

	// Just statistics
	minPartSize := g.NodeCount()
	maxPartSize := 0
	for p := 0; p < pi.Count; p++ {
		curSize := 0
		for i := 0; i < g.NodeCount(); i++ {
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

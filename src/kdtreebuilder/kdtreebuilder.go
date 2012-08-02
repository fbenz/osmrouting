// Creates k-d trees for all clusters and the overlay graph. In addition, bounding boxes for all
// clusters are computed and written into one file.

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"geo"
	"graph"
	"kdtree"
	"log"
	"mm"
	"os"
	"path"
	"runtime"
	"sort"
)

const (
	MaxThreads = 8
	Parts      = 10 // the construction of the k-d trees for the cluster is split into parts

	BBoxMargin = 0.002 // bounding boxes are enlarged to ensure that points on the borders are found
)

var (
	FlagBaseDir string
)

func init() {
	flag.StringVar(&FlagBaseDir, "dir", "", "directory of the graph files")
}

func main() {
	runtime.GOMAXPROCS(MaxThreads)
	flag.Parse()

	clusterGraph, err := graph.OpenClusterGraph(FlagBaseDir, false /* loadMatrices */)
	if err != nil {
		log.Fatal("Loading graph: ", err)
	}

	fmt.Printf("Create k-d trees for the subgraphs\n")
	bboxes := make([]geo.BBox, len(clusterGraph.Cluster))

	partSize := len(clusterGraph.Cluster)/Parts + 1
	for j := 0; j < Parts; j++ {
		start := j * partSize
		end := (j + 1) * partSize
		if start >= len(clusterGraph.Cluster) {
			// for small cluster counts
			break
		}
		if end > len(clusterGraph.Cluster) {
			end = len(clusterGraph.Cluster)
		}
		subCluster := clusterGraph.Cluster[start:end]

		ready := make(chan int, len(subCluster))
		for i, g := range subCluster {
			clusterDir := fmt.Sprintf("/cluster%d", start+i+1)
			go writeKdTreeSubgraph(ready, path.Join(FlagBaseDir, clusterDir), g, bboxes, start+i)
		}
		for _, _ = range subCluster {
			<-ready
		}
		for _, g := range subCluster {
			graph.CloseGraphFile(g)
		}
	}

	// write bounding boxes to file
	fmt.Printf("Write bounding boxes\n")
	var bboxesFile []int32
	err = mm.Create(path.Join(FlagBaseDir, "bboxes.ftf"), len(bboxes)*4, &bboxesFile)
	if err != nil {
		log.Fatal("mm.Create failed: ", err)
	}
	for i, b := range bboxes {
		encodedBox := b.Encode()
		for j := 0; j < 4; j++ {
			bboxesFile[4*i+j] = encodedBox[j]
		}
	}
	err = mm.Close(&bboxesFile)
	if err != nil {
		log.Fatal("mm.Close failed: ", err)
	}

	fmt.Printf("Create k-d trees for the overlay graph\n")
	writeKdTreeOverlay(path.Join(FlagBaseDir, "/overlay"), clusterGraph.Overlay)
}

type byLat struct {
	*kdtree.KdTree
}

func (x byLat) Less(i, j int) bool {
	return x.KdTree.Coordinates[i].Lat < x.KdTree.Coordinates[j].Lat
}

type byLng struct {
	*kdtree.KdTree
}

func (x byLng) Less(i, j int) bool {
	return x.KdTree.Coordinates[i].Lng < x.KdTree.Coordinates[j].Lng
}

func createKdTreeSubgraph(g *graph.GraphFile) (*kdtree.KdTree, geo.BBox) {
	t := &kdtree.KdTree{Graph: g, EncodedSteps: []uint64(nil), Coordinates: []geo.Coordinate(nil)}
	bbox := geo.NewBBoxPoint(g.VertexCoordinate(graph.Vertex(0)))

	// line up all coordinates and their encodings in the subgraph
	steps := []geo.Coordinate(nil)
	for i := 0; i < g.VertexCount(); i++ {
		vertex := graph.Vertex(i)
		t.Coordinates = append(t.Coordinates, g.VertexCoordinate(vertex))
		t.AppendEncodedStep(encodeCoordinate(i, kdtree.MaxEdgeOffset, kdtree.MaxStepOffset))
		bbox = bbox.Union(geo.NewBBoxPoint(g.VertexCoordinate(vertex)))
		degree := g.FirstOut[i+1] - g.FirstOut[i]
		for j := uint32(0); j < degree; j++ {
			e := graph.Edge(g.FirstOut[i] + j)
			steps = g.EdgeSteps(e, vertex, steps)

			if len(steps) > 2000 {
				panic("steps > 2000")
			}

			for k, s := range steps {
				t.Coordinates = append(t.Coordinates, s)
				t.AppendEncodedStep(encodeCoordinate(i, int(j), k))
				bbox = bbox.Union(geo.NewBBoxPoint(s))
			}
		}
	}

	sortTree(t, true)
	return t, bbox
}

func createKdTreeOverlay(g *graph.OverlayGraphFile) *kdtree.KdTree {
	t := &kdtree.KdTree{
		Graph:        g.GraphFile,
		EncodedSteps: []uint64(nil),
		Coordinates:  []geo.Coordinate(nil),
	}
	cuts := g.GraphFile

	// line up all coordinates and their encodings in the overlay graph
	steps := []geo.Coordinate(nil)
	fmt.Printf("Overlay vertex count: %d\n", g.VertexCount())
	for i := 0; i < cuts.VertexCount(); i++ {
		vertex := graph.Vertex(i)
		t.Coordinates = append(t.Coordinates, cuts.VertexCoordinate(vertex))
		t.AppendEncodedStep(encodeCoordinate(i, kdtree.MaxEdgeOffset, kdtree.MaxStepOffset))
		degree := cuts.FirstOut[i+1] - cuts.FirstOut[i]
		for j := uint32(0); j < degree; j++ {
			e := graph.Edge(cuts.FirstOut[i] + j)
			steps = cuts.EdgeSteps(e, vertex, steps)

			for k, s := range steps {
				t.Coordinates = append(t.Coordinates, s)
				t.AppendEncodedStep(encodeCoordinate(i, int(j), k))
			}
		}
	}

	sortTree(t, true)
	return t
}

func subKdTree(t *kdtree.KdTree, from, to int) *kdtree.KdTree {
	// The EncodedSteps slice is restricted by pointers and not with a new slice due to its encoding.
	return &kdtree.KdTree{Graph: t.Graph, EncodedSteps: t.EncodedSteps, Coordinates: t.Coordinates[from:to],
		EncodedStepsStart: t.EncodedStepsStart + from, EncodedStepsEnd: t.EncodedStepsStart + to - 1}
}

// sortTree sorts the given tree by comparing either lat or long
func sortTree(t *kdtree.KdTree, compareLat bool) {
	if t.Len() <= 1 {
		return
	}
	if compareLat {
		sort.Sort(byLat{t})
	} else {
		sort.Sort(byLng{t})
	}
	// sort recursively both halfs with the comparison alternating between lat and long
	middle := t.Len() / 2
	sortTree(subKdTree(t, 0, middle), !compareLat)
	sortTree(subKdTree(t, middle+1, t.Len()), !compareLat)
}

// writeKdTreeSubgraph creates and stores the k-d tree for the given cluster graph
func writeKdTreeSubgraph(ready chan<- int, baseDir string, g *graph.GraphFile, bboxes []geo.BBox, pos int) {
	t, bbox := createKdTreeSubgraph(g)
	err := writeToFile(t, baseDir)
	if err != nil {
		log.Fatal("Creating k-d tree: ", err)
	}
	// a very simple margin is added
	bbox.Min.Lat -= BBoxMargin
	bbox.Min.Lng -= BBoxMargin
	bbox.Max.Lat += BBoxMargin
	bbox.Max.Lng += BBoxMargin

	bboxes[pos] = bbox
	ready <- 1
}

// writeKdTree creates and stores the k-d tree for the given graph
func writeKdTreeOverlay(baseDir string, g *graph.OverlayGraphFile) {
	t := createKdTreeOverlay(g)
	err := writeToFile(t, baseDir)
	if err != nil {
		log.Fatal("Creating k-d tree: ", err)
	}
}

// writeToFile stores the permitation created by the k-d tree
func writeToFile(t *kdtree.KdTree, baseDir string) error {
	output, err := os.Create(path.Join(baseDir, "kdtree.ftf"))
	if err != nil {
		return err
	}
	defer output.Close()
	err = binary.Write(output, binary.LittleEndian, t.EncodedSteps)
	if err != nil {
		return err
	}
	return writeCoordinates(t, baseDir)
}

func encodeCoordinate(vertexIndex, edgeOffset, stepOffset int) uint64 {
	if vertexIndex > kdtree.MaxVertexIndex {
		panic("vertex index too large")
	}
	// both offsets are at max if only a vertex is encoded
	if edgeOffset != kdtree.MaxEdgeOffset && stepOffset != kdtree.MaxStepOffset {
		if edgeOffset >= kdtree.MaxEdgeOffset {
			panic("edge offset too large")
		}
		if stepOffset >= kdtree.MaxStepOffset {
			panic("step offset too large")
		}
	}

	ec := uint64(vertexIndex) << (kdtree.EdgeOffsetBits + kdtree.StepOffsetBits)
	ec |= uint64(edgeOffset) << kdtree.StepOffsetBits
	ec |= uint64(stepOffset)
	return ec
}

func writeCoordinates(t *kdtree.KdTree, dir string) error {
	var coordinates []int32
	err := mm.Create(path.Join(dir, "coordinates.ftf"), len(t.Coordinates)*2, &coordinates)
	if err != nil {
		return err
	}

	for i, c := range t.Coordinates {
		lat, lng := c.Encode()
		coordinates[2*i] = lat
		coordinates[2*i+1] = lng
	}

	return mm.Close(&coordinates)
}

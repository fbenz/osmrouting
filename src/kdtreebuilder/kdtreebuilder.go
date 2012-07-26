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
	ready := make(chan int, len(clusterGraph.Cluster))
	for i, g := range clusterGraph.Cluster {
		clusterDir := fmt.Sprintf("/cluster%d", i+1)
		go writeKdTreeSubgraph(ready, path.Join(FlagBaseDir, clusterDir), g.(*graph.GraphFile), bboxes, i)
	}
	for _, _ = range clusterGraph.Cluster {
		<-ready
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
	writeKdTreeOverlay(path.Join(FlagBaseDir, "/overlay"), clusterGraph.Overlay.(*graph.OverlayGraphFile))
}

type byLat struct {
	kdtree.KdTree
}

func (x byLat) Less(i, j int) bool {
	return x.KdTree.Coordinates[i].Lat < x.KdTree.Coordinates[j].Lat
}

type byLng struct {
	kdtree.KdTree
}

func (x byLng) Less(i, j int) bool {
	return x.KdTree.Coordinates[i].Lng < x.KdTree.Coordinates[j].Lng
}

func createKdTreeSubgraph(g *graph.GraphFile) (kdtree.KdTree, geo.BBox) {
	estimatedSize := g.VertexCount() + 4*g.EdgeCount()
	EncodedSteps := make([]uint32, 0, estimatedSize)
	coordinates := make([]geo.Coordinate, 0, estimatedSize)

	bbox := geo.NewBBoxPoint(g.VertexCoordinate(graph.Vertex(0)))

	// line up all coordinates and their encodings in the graph
	edges := []graph.Edge(nil)
	for i := 0; i < g.VertexCount(); i++ {
		vertex := graph.Vertex(i)
		coordinates = append(coordinates, g.VertexCoordinate(vertex))
		EncodedSteps = append(EncodedSteps, encodeCoordinate(i, kdtree.MaxEdgeOffset, kdtree.MaxStepOffset))
		bbox.Union(geo.NewBBoxPoint(g.VertexCoordinate(vertex)))

		edges = g.VertexRawEdges(vertex, edges)
		for j, e := range edges {
			steps := g.EdgeSteps(e, vertex)

			if len(steps) > 2000 {
				panic("steps > 2000")
			}

			for k, s := range steps {
				coordinates = append(coordinates, s)
				EncodedSteps = append(EncodedSteps, encodeCoordinate(i, j, k))
				bbox.Union(geo.NewBBoxPoint(s))
			}
		}
	}

	t := kdtree.KdTree{Graph: g, EncodedSteps: EncodedSteps, Coordinates: coordinates}
	createSubTree(t, true)
	return t, bbox
}

func createKdTreeOverlay(g *graph.OverlayGraphFile) kdtree.KdTree {
	estimatedSize := g.VertexCount() + 4*g.EdgeCount()
	EncodedSteps := make([]uint32, 0, estimatedSize)
	coordinates := make([]geo.Coordinate, 0, estimatedSize)

	// line up all coordinates and their encodings in the graph
	edges := []graph.Edge(nil)
	for i := 0; i < g.VertexCount(); i++ {
		vertex := graph.Vertex(i)
		coordinates = append(coordinates, g.VertexCoordinate(vertex))
		EncodedSteps = append(EncodedSteps, encodeCoordinate(i, kdtree.MaxEdgeOffset, kdtree.MaxStepOffset))

		g.VertexRawEdges(vertex, edges)
		for j, e := range edges {
			steps := g.EdgeSteps(e, vertex)
			for k, s := range steps {
				coordinates = append(coordinates, s)
				EncodedSteps = append(EncodedSteps, encodeCoordinate(i, j, k))
			}
		}
	}

	t := kdtree.KdTree{Graph: g, EncodedSteps: EncodedSteps, Coordinates: coordinates}
	createSubTree(t, true)
	return t
}

func subKdTree(t kdtree.KdTree, from, to int) kdtree.KdTree {
	return kdtree.KdTree{Graph: t.Graph, EncodedSteps: t.EncodedSteps[from:to], Coordinates: t.Coordinates[from:to]}
}

func createSubTree(t kdtree.KdTree, compareLat bool) {
	if t.Len() <= 1 {
		return
	}
	if compareLat {
		sort.Sort(byLat{t})
	} else {
		sort.Sort(byLng{t})
	}
	middle := t.Len() / 2
	createSubTree(subKdTree(t, 0, middle), !compareLat)
	createSubTree(subKdTree(t, middle+1, t.Len()), !compareLat)
}

// writeKdTreeSubgraph creates and stores the k-d tree for the given cluster graph
func writeKdTreeSubgraph(ready chan<- int, baseDir string, g *graph.GraphFile, bboxes []geo.BBox, pos int) {
	t, bbox := createKdTreeSubgraph(g)
	err := writeToFile(t, baseDir)
	if err != nil {
		log.Fatal("Creating k-d tree: ", err)
	}
	// TODO add margin?
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
func writeToFile(t kdtree.KdTree, baseDir string) error {
	output, err := os.Create(path.Join(baseDir, "kdtree.ftf"))
	defer output.Close()
	if err != nil {
		return err
	}
	writeErr := binary.Write(output, binary.LittleEndian, t.EncodedSteps)
	if writeErr != nil {
		return writeErr
	}
	return nil
}

func encodeCoordinate(vertexIndex, edgeOffset, stepOffset int) uint32 {
	if vertexIndex > kdtree.MaxVertexIndex {
		panic("vertex index too large")
	}
	// both offsets are at max if only a vertex is encoded
	if edgeOffset != kdtree.MaxEdgeOffset && stepOffset != kdtree.MaxStepOffset {
		if edgeOffset >= kdtree.MaxEdgeOffset {
			panic("edge offset too large")
		}
		if stepOffset >= kdtree.MaxStepOffset {
			fmt.Printf("vertex: %d, edge offset: %v, step offset: %v\n", vertexIndex, edgeOffset, stepOffset)
			panic("step offset too large")
		}
	}

	ec := uint32(vertexIndex) << (kdtree.EdgeOffsetBits + kdtree.StepOffsetBits)
	ec |= uint32(edgeOffset) << kdtree.StepOffsetBits
	ec |= uint32(stepOffset)
	return ec
}

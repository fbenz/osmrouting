package main

import (
	"ellipsoid"
	"encoding/binary"
	"flag"
	"fmt"
	"geo"
	"graph"
	"kdtree"
	"log"
	"os"
	"path"
	"runtime"
	"sort"
)

var (
	FlagBaseDir string
)

func init() {
	flag.StringVar(&FlagBaseDir, "dir", "", "directory of the graph files")
}

func main() {
	runtime.GOMAXPROCS(16)
	flag.Parse()

	clusterGraph, err := graph.OpenClusterGraph(FlagBaseDir)
	if err != nil {
		log.Fatal("Loading graph: ", err)
	}

	ready := make(chan int, len(clusterGraph.Cluster))
	for i, g := range clusterGraph.Cluster {
		clusterDir := fmt.Sprintf("cluster%d", i+1)
		go writeKdTree(ready, path.Join(FlagBaseDir, clusterDir), g)
	}
	for _, _ = range clusterGraph.Cluster {
		<-ready
	}

	writeKdTree(ready, path.Join(FlagBaseDir, "/overlay"), clusterGraph.Overlay)

	// TODO create segment tree out of the bounding boxes
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

func createKdTree(g graph.Graph) kdtree.KdTree {
	ellipsoidGeo := ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter, ellipsoid.Longitude_is_symmetric, ellipsoid.Bearing_is_symmetric)

	estimatedSize := g.VertexCount() + 4*g.EdgeCount()
	EncodedSteps := make([]uint32, 0, estimatedSize)
	coordinates := make([]geo.Coordinate, 0, estimatedSize)

	// line up all coordinates and their encodings in the graph
	for i := 0; i < g.VertexCount(); i++ {
		vertex := graph.Vertex(i)
		coordinates = append(coordinates, g.VertexCoordinate(vertex))
		EncodedSteps = append(EncodedSteps, encodeCoordinate(i, 0xFF, 0xFF))

		iter := g.VertexEdgeIterator(vertex, true /* out edges */, 0 /* metric */)
		j := 0
		for e, ok := iter.Next(); ok; e, ok = iter.Next() {
			steps := g.EdgeSteps(e, vertex)
			for k, s := range steps {
				coordinates = append(coordinates, s)
				EncodedSteps = append(EncodedSteps, encodeCoordinate(i, j, k))
			}
			j++
		}
	}

	t := kdtree.KdTree{Graph: g, Geo: &ellipsoidGeo, EncodedSteps: EncodedSteps, Coordinates: coordinates}
	createSubTree(t, true)
	return t
}

func subKdTree(t kdtree.KdTree, from, to int) kdtree.KdTree {
	return kdtree.KdTree{Graph: t.Graph, Geo: t.Geo, EncodedSteps: t.EncodedSteps[from:to], Coordinates: t.Coordinates[from:to]}
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

// WriteKdTree creates and stores the k-d tree for the given graph
func writeKdTree(ready chan<- int, baseDir string, g graph.Graph) {
	t := createKdTree(g)
	err := writeToFile(t, baseDir)
	if err != nil {
		log.Fatal("Creating k-d tree: ", err)
	}
	ready <- 1
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
	if vertexIndex > 0xFFFF {
		panic("vertex index too large")
	}
	if edgeOffset >= 0xFF {
		panic("edge offset too large")
	}
	if stepOffset >= 0xFF {
		panic("step offset too large")
	}

	ec := uint32(vertexIndex) << 16
	ec |= uint32(edgeOffset) << 8
	ec |= uint32(stepOffset)
	return ec
}

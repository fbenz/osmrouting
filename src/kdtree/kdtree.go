// Package for creating and storing a k-d tree
// It is actually a 2-d tree for the dimensions latitude and longitude.

package kdtree

import (
	"ellipsoid"
	"encoding/binary"
	"graph"
	"os"
	"path"
	"sort"
)

const (
	FilenameKdTree = "kdtree.ftf"
)

// TODO check whether this keeps uint32 or the graph interface changes
type Nodes []uint32

type KdTree struct {
	Nodes Nodes
	Positions graph.Positions
	Geo ellipsoid.Ellipsoid
}

func (s Nodes) Len() int      { return len(s) }
func (s Nodes) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type byLat struct {
	Nodes
	tree *KdTree
}

func (x byLat) Less(i, j int) bool { return x.tree.Positions.Lat(int(x.Nodes[i])) < x.tree.Positions.Lat(int(x.Nodes[j])) }

type byLng struct {
	Nodes
	tree *KdTree
}

func (x byLng) Less(i, j int) bool { return x.tree.Positions.Lng(int(x.Nodes[i])) < x.tree.Positions.Lng(int(x.Nodes[j])) }

func (t KdTree) Lat(i int) float64 {
	return t.Positions.Lat(int(t.Nodes[i]))
}

func (t KdTree) Lng(i int) float64 {
	return t.Positions.Lng(int(t.Nodes[i]))
}

func newkdTree(positions graph.Positions) KdTree {
	nodes := make(Nodes, positions.Len())
	for i := 0; i < positions.Len(); i++ {
		nodes[i] = uint32(i)
	}
	g := ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter, ellipsoid.Longitude_is_symmetric, ellipsoid.Bearing_is_symmetric)
	t := KdTree{nodes, positions, g}
	ready := make(chan int, 1)
	go t.create(ready, t.Nodes, true)
	<- ready
	return t
}

func (t KdTree) create(ready chan<- int, nodes Nodes, compareLat bool) {
	if len(nodes) <= 1 {
		return
	}
	if compareLat {
		sort.Sort(byLat{nodes, &t})
	} else {
		sort.Sort(byLng{nodes, &t})
	}
	middle := len(nodes) / 2
	// the cut off where we stop to start new goroutines
	if len(nodes) < (len(t.Nodes) / 4) {
		t.createSeq(nodes[:middle], !compareLat) // correct without -1 as the upper bound is equal to the length
		t.createSeq(nodes[middle+1:], !compareLat)
	} else {
		childsReady := make(chan int, 2)
		go t.create(childsReady, nodes[:middle], !compareLat) // correct without -1 as the upper bound is equal to the length
		go t.create(childsReady, nodes[middle+1:], !compareLat)
		<- childsReady
		<- childsReady
	}
	ready <- 1
}

func (t KdTree) createSeq(nodes Nodes, compareLat bool) {
	if len(nodes) <= 1 {
		return
	}
	if compareLat {
		sort.Sort(byLat{nodes, &t})
	} else {
		sort.Sort(byLng{nodes, &t})
	}
	middle := len(nodes) / 2
	t.createSeq(nodes[:middle], !compareLat) // correct without -1 as the upper bound is equal to the length
	t.createSeq(nodes[middle+1:], !compareLat)
}

// writeToFile stores the permitation created by the k-d tree
func (t KdTree) writeToFile(baseDir, filename string) error {
	output, err := os.Create(path.Join(baseDir, filename))
	defer output.Close()
	if err != nil {
		return err
	}
	writeErr := binary.Write(output, binary.LittleEndian, t.Nodes)
	if writeErr != nil {
		return writeErr
	}
	return nil
}

// WriteKdTree creates and stores the k-d tree for the given positions
func WriteKdTree(baseDir string, positions graph.Positions) error {
	t := newkdTree(positions)
	return t.writeToFile(baseDir, FilenameKdTree)
}

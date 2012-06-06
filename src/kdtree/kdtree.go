// Package for creating and storing a k-d tree
// It is actually a 2-d tree for the dimensions latitude and longitude.

// TODO check castings

package kdtree

import (
	"encoding/binary"
	"graph"
	"os"
	"sort"
)

const (
	FilenameKdTree = "kdtree.ftf"
)

type Nodes []uint32

type KdTree struct {
	Nodes Nodes
	Positions graph.Positions
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
	t := KdTree{nodes, positions}
	t.create(t.Nodes, true)
	return t
}

func (t KdTree) create(nodes Nodes, compareLat bool) {
	if len(nodes) <= 1 {
		return
	}
	if compareLat {
		sort.Sort(byLat{nodes, &t})
	} else {
		sort.Sort(byLng{nodes, &t})
	}
	middle := len(nodes) / 2
	t.create(nodes[:middle], !compareLat) // correct without -1 as the upper bound is equal to the length
	t.create(nodes[middle+1:], !compareLat)
}

// writeToFile stores the permitation created by the k-d tree
func (t KdTree) writeToFile(filename string) error {
	output, err := os.Create(filename)
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
func WriteKdTree(positions graph.Positions) error {
	t := newkdTree(positions)
	return t.writeToFile(FilenameKdTree)
}

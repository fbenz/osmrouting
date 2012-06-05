package alg

import (
	"encoding/binary"
	"graph"
	"kdtree"
	"os"
)

var kdTree kdtree.KdTree

func LoadKdTree(positions graph.Positions) error {
	kdTreePermutation := make(kdtree.Nodes, positions.Len())
	input, err := os.Open(kdtree.FilenameKdTree)
	if err != nil {
		return err
	}
	err = binary.Read(input, binary.LittleEndian, kdTreePermutation)
	if err != nil {
		return err
	}
	kdTree = kdtree.KdTree{kdTreePermutation, positions}
	return nil
}

func NearestNeighbor(lat, lng float64, forward bool) (graph.Step, []graph.Way) {
	index := search(lat, lng, true)
	return kdTree.Positions.Step(index), kdTree.Positions.Ways(index, forward)
}

func search(lat, lng float64, compareLat bool) int {
	index, lineSearch := binarySearch(kdTree.Nodes, lat, lng, compareLat)
	if lineSearch {
		if lat == kdTree.Positions.Lat(index) {
			return linearSearch(lat, lng)
		}
		return linearSearch(lat, lng)
	}
	return kdTree.Nodes[index]
}

func binarySearch(nodes kdtree.Nodes, lat, lng float64, compareLat bool) (int, bool) {
	if len(nodes) == 0 {
		panic("nearestNeighbor.binarySearch")
	} else if len(nodes) == 1 {
		return 0, false
	}
	middle := len(nodes) / 2

	// exact hit
	if lat == kdTree.Positions.Lat(nodes[middle]) && lng == kdTree.Positions.Lng(nodes[middle]) {
		return middle, false
	}

	// If two or more nodes have lat/lng in common with the given point, 
	// we can not guarantee to hit OSM with exactly the coordinates of the given point.
	// But this is required for the project at the moment, so we switch to line search.
	if compareLat && lat == kdTree.Positions.Lat(nodes[middle]) {
		return middle, true
	}
	if !compareLat && lng == kdTree.Positions.Lng(nodes[middle]) {
		return middle, true
	}

	var left bool
	if compareLat {
		left = lat < kdTree.Positions.Lat(nodes[middle])
	} else {
		left = lng < kdTree.Positions.Lng(nodes[middle])
	}
	if left {
		// stop if there is nothing left of the middle
		if middle == 0 {
			return middle, false
		}
		return binarySearch(nodes[:middle], lat, lng, !compareLat)
	}
	// stop if there is nothing right of the middle
	if middle == len(nodes)-1 {
		return middle, false
	}
	index, linearSearch := binarySearch(nodes[middle+1:], lat, lng, !compareLat)
	return middle + 1 + index, linearSearch
}

func linearSearch(lat, lng float64) int {
	for i := range kdTree.Nodes {
		if lat == kdTree.Positions.Lat(i) && lng == kdTree.Positions.Lng(i) {
			return i
		}
	}
	panic("nearestNeighbor.linearSearch")
}

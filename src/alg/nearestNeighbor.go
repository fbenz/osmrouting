package alg

import (
	"geo"
	"graph"
	"kdtree"
)

func LoadKdTree(base string, positions graph.Positions) (*kdtree.KdTree, error) {
	kdTreePermutation, err := graph.MmapFileUint32(base, kdtree.FilenameKdTree)
	if err != nil {
		return nil, err
	}
	return &kdtree.KdTree{kdTreePermutation, positions}, nil
}

func NearestNeighbor(kdTree *kdtree.KdTree, lat, lng float64, forward bool) (graph.Step, []graph.Way) {
	index := binarySearch(kdTree, kdTree.Nodes, lat, lng, true /* compareLat */)
	if index >= uint32(kdTree.Positions.Len()) {
		panic("nearestNeighbor found index is too large")
	}
	return kdTree.Positions.Step(int(index)), kdTree.Positions.Ways(int(index), forward)
}

func binarySearch(kdTree *kdtree.KdTree, nodes kdtree.Nodes, lat, lng float64, compareLat bool) uint32 {
	if len(nodes) == 0 {
		panic("nearestNeighbor.binarySearch")
	} else if len(nodes) == 1 {
		return nodes[0]
	}
	middle := len(nodes) / 2

	// exact hit
	if lat == kdTree.Positions.Lat(int(nodes[middle])) && lng == kdTree.Positions.Lng(int(nodes[middle])) {
		return nodes[middle]
	}

	// corner case where the nearest point can be on both sides of the middle
	if (compareLat && lat == kdTree.Positions.Lat(int(nodes[middle]))) || (!compareLat && lng == kdTree.Positions.Lng(int(nodes[middle]))) {
		// recursion on both halfs
		leftRecIndex := binarySearch(kdTree, nodes[:middle], lat, lng, !compareLat)
		rightRecIndex := binarySearch(kdTree, nodes[middle+1:], lat, lng, !compareLat)
		distMiddle := distance(lat, lng, kdTree.Positions.Lat(int(nodes[middle])), kdTree.Positions.Lng(int(nodes[middle])))
		distRecursionLeft := distance(lat, lng, kdTree.Positions.Lat(int(leftRecIndex)), kdTree.Positions.Lng(int(leftRecIndex)))
		distRecursionRight := distance(lat, lng, kdTree.Positions.Lat(int(rightRecIndex)), kdTree.Positions.Lng(int(rightRecIndex)))
		if distRecursionLeft < distRecursionRight {
			if distRecursionLeft < distMiddle {
				return leftRecIndex
			}
			return nodes[middle]
		}
		if distRecursionRight < distMiddle {
			return rightRecIndex
		}
		return nodes[middle]
	}

	var left bool
	if compareLat {
		left = lat < kdTree.Positions.Lat(int(nodes[middle]))
	} else {
		left = lng < kdTree.Positions.Lng(int(nodes[middle]))
	}
	if left {
		// stop if there is nothing left of the middle
		if middle == 0 {
			return nodes[middle]
		}
		// recursion on the left half
		recIndex := binarySearch(kdTree, nodes[:middle], lat, lng, !compareLat)
		// compare middle and result from the left
		distMiddle := distance(lat, lng, kdTree.Positions.Lat(int(nodes[middle])), kdTree.Positions.Lng(int(nodes[middle])))
		distRecursion := distance(lat, lng, kdTree.Positions.Lat(int(recIndex)), kdTree.Positions.Lng(int(recIndex)))
		if distMiddle < distRecursion {
			return nodes[middle]
		}
		return recIndex
	}
	// stop if there is nothing right of the middle
	if middle == len(nodes)-1 {
		return nodes[middle]
	}
	// recursion on the right half
	recIndex := binarySearch(kdTree, nodes[middle+1:], lat, lng, !compareLat)
	// compare middle and result from the right
	distMiddle := distance(lat, lng, kdTree.Positions.Lat(int(nodes[middle])), kdTree.Positions.Lng(int(nodes[middle])))
	distRecursion := distance(lat, lng, kdTree.Positions.Lat(int(recIndex)), kdTree.Positions.Lng(int(recIndex)))
	if distMiddle < distRecursion {
		return nodes[middle]
	}
	return recIndex
}

// TODO remove this function if everything uses geo.Coordinate
func distance(lat1, lng1, lat2, lng2 float64) float64 {
	return geo.Coordinate{Lat: lat1, Lng: lng1}.Distance(geo.Coordinate{Lat: lat2, Lng: lng2})
	//return geo.Distance(geo.Coordinate{Lat: lat1, Lng: lng1}, geo.Coordinate{Lat: lat2, Lng: lng2})
}

package kdtree

import (
	"ellipsoid"
	"errors"
	"fmt"
	"geo"
	"graph"
	"math"
	"mm"
	"path"
)

var (
	inf           float64
	e             ellipsoid.Ellipsoid
	clusterKdTree ClusterKdTree
)

type NN struct {
	kdTree *KdTree
	x      geo.Coordinate
	trans  graph.Transport
}

func init() {
	inf = math.Inf(1)
	e = ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter, ellipsoid.Longitude_is_symmetric, ellipsoid.Bearing_is_symmetric)
}

func LoadKdTree(clusterGraph *graph.ClusterGraph, base string) error {
	// Load the k-d tree of all clusters
	clusterKdTrees := make([]*KdTree, len(clusterGraph.Cluster))
	for i, g := range clusterGraph.Cluster {
		clusterDir := path.Join(base, fmt.Sprintf("/cluster%d", i+1))
		var encodedSteps []uint64
		var encodedCoordinates []int32
		err := mm.Open(path.Join(clusterDir, "kdtree.ftf"), &encodedSteps)
		if err != nil {
			return err
		}
		err = mm.Open(path.Join(clusterDir, "coordinates.ftf"), &encodedCoordinates)
		if err != nil {
			return err
		}
		clusterKdTrees[i] = &KdTree{
			Graph:              g,
			EncodedSteps:       encodedSteps,
			EncodedCoordinates: encodedCoordinates,
		}
	}

	// Load the k-d tree of the overlay graph
	var encodedSteps []uint64
	var encodedCoordinates []int32
	err := mm.Open(path.Join(base, "/overlay/kdtree.ftf"), &encodedSteps)
	if err != nil {
		return err
	}
	err = mm.Open(path.Join(base, "/overlay/coordinates.ftf"), &encodedCoordinates)
	if err != nil {
		return err
	}
	overlayKdTree := &KdTree{
		Graph:              clusterGraph.Overlay.GraphFile,
		EncodedSteps:       encodedSteps,
		EncodedCoordinates: encodedCoordinates,
	}

	// Load the bounding boxes of the clusters
	var bboxesFile []int32
	err = mm.Open(path.Join(base, "bboxes.ftf"), &bboxesFile)
	if err != nil {
		return err
	}
	if len(bboxesFile)/4 != clusterGraph.Overlay.ClusterCount() {
		return errors.New("size of bboxes file does not match cluster count")
	}
	bboxes := make([]geo.BBox, len(bboxesFile)/4)
	for i, _ := range bboxes {
		bboxes[i] = geo.DecodeBBox(bboxesFile[4*i : 4*i+4])
	}

	clusterKdTree = ClusterKdTree{
		Overlay: overlayKdTree,
		Cluster: clusterKdTrees,
		BBoxes:  bboxes,
	}
	return nil
}

var linear = 0

// NearestNeighbor returns -1 if the location is on the overlay graph
// No fail strategy: a nearest point on the overlay graph is always returned if no point
// is found in the clusters.
func NearestNeighbor(x geo.Coordinate, trans graph.Transport) Location {
	// first search on the overlay graph
	t := clusterKdTree.Overlay
	nn := &NN{
		kdTree: t,
		x:      x,
		trans:  trans,
	}
	bestStepIndex, coord, foundPoint := nn.binarySearch(0, t.EncodedStepLen()-1, true /* compareLat */)
	minDistance, _ := e.To(x.Lat, x.Lng, coord.Lat, coord.Lng)

	// then search on all clusters where the point is inside the bounding box of the cluster
	clusterIndex := -1
	for i, b := range clusterKdTree.BBoxes {
		if b.Contains(x) {
			t = clusterKdTree.Cluster[i]
			nn = &NN{
				kdTree: t,
				x:      x,
				trans:  trans,
			}
			stepIndex, coord, ok := nn.binarySearch(0, t.EncodedStepLen()-1, true /* compareLat */)
			dist, _ := e.To(x.Lat, x.Lng, coord.Lat, coord.Lng)

			if ok && (!foundPoint || dist < minDistance) {
				foundPoint = true
				minDistance = dist
				bestStepIndex = stepIndex
				clusterIndex = i
			}
		}
	}

	if clusterIndex >= 0 {
		kdTree := clusterKdTree.Cluster[clusterIndex]
		return Location{
			Graph:   kdTree.Graph,
			EC:      kdTree.EncodedStep(bestStepIndex),
			Cluster: clusterIndex,
		}
	}
	return Location{
		Graph:   clusterKdTree.Overlay.Graph,
		EC:      clusterKdTree.Overlay.EncodedStep(bestStepIndex),
		Cluster: clusterIndex,
	}
}

// binarySearch in one k-d tree. The index, the coordinate, and if the returned step/vertex
// is accessible are returned.
func (nn *NN) binarySearch(start, end int, compareLat bool) (int, geo.Coordinate, bool) {
	if end <= start {
		// end of the recursion
		startCoord, startAccessible := nn.decodeCoordinate(start)
		return start, startCoord, startAccessible
	}

	x := nn.x
	middle := (end-start)/2 + start
	middleCoord, middleAccessible := nn.decodeCoordinate(middle)

	// exact hit
	if middleAccessible && x.Lat == middleCoord.Lat && x.Lng == middleCoord.Lng {
		return middle, middleCoord, middleAccessible
	}

	middleDist := inf
	if middleAccessible {
		middleDist = distance(x, middleCoord)
	}

	// recursion one half and if no accessible point is returned also on the other half
	var left bool
	if compareLat {
		left = x.Lat < middleCoord.Lat
	} else {
		left = x.Lng < middleCoord.Lng
	}
	var recIndex int
	var recCoord geo.Coordinate
	var recAccessible bool
	bothHalfs := false
	if left {
		// left
		recIndex, recCoord, recAccessible = nn.binarySearch(start, middle-1, !compareLat)
		if !recAccessible {
			// other half -> right
			recIndex, recCoord, recAccessible = nn.binarySearch(middle+1, end, !compareLat)
			bothHalfs = true
		}
	} else {
		// right
		recIndex, recCoord, recAccessible = nn.binarySearch(middle+1, end, !compareLat)
		if !recAccessible {
			// other half -> left
			recIndex, recCoord, recAccessible = nn.binarySearch(start, middle-1, !compareLat)
			bothHalfs = true
		}
	}
	bestDistance := distance(x, recCoord)

	// we are finished if both have already been searched
	if bothHalfs {
		if middleDist < bestDistance {
			return middle, middleCoord, middleAccessible
		} else {
			return recIndex, recCoord, recAccessible
		}
	}

	distToPlane := 0.0
	if compareLat {
		distToPlane = (x.Lat - middleCoord.Lat) * (x.Lat - middleCoord.Lat)
	} else {
		distToPlane = (x.Lng - middleCoord.Lng) * (x.Lng - middleCoord.Lng)
	}

	var recIndex2 int
	var recCoord2 geo.Coordinate
	recAccessible2 := false
	// Test whether the current best distance circle crosses the plane.
	// We subtract 10 so that even with possible inaccuracies the nearest neighbor and not 
	// some near neighbor is found (10 is about 3.16m because of the squared distance).
	if bestDistance >= distToPlane-10 {
		// search on the other half
		if !left {
			// left
			recIndex2, recCoord2, recAccessible2 = nn.binarySearch(start, middle-1, !compareLat)
		} else {
			// right
			recIndex2, recCoord2, recAccessible2 = nn.binarySearch(middle+1, end, !compareLat)
		}
	}

	bestDistance2 := inf
	if recAccessible2 {
		bestDistance2 = distance(x, recCoord2)
	}

	if bestDistance < bestDistance2 {
		if middleDist < bestDistance {
			return middle, middleCoord, middleAccessible
		} else {
			return recIndex, recCoord, recAccessible
		}
	}
	if middleDist < bestDistance2 {
		return middle, middleCoord, middleAccessible
	}
	return recIndex2, recCoord2, recAccessible2
}

// decodeCoordinate returns the coordinate of the encoded vertex/step and if it is accessible by the
// given transport mode
func (nn *NN) decodeCoordinate(i int) (geo.Coordinate, bool) {
	t := nn.kdTree
	g := t.Graph
	// inlining of geo.DecodeCoordinate as the Go compiler does not do it
	coord := geo.Coordinate{
		Lat: float64(t.EncodedCoordinates[2*i]) / geo.OsmPrecision,
		Lng: float64(t.EncodedCoordinates[2*i+1]) / geo.OsmPrecision,
	}

	// decode the index and the offsets
	ec := t.EncodedStep(i)
	vertexIndex := ec >> (EdgeOffsetBits + StepOffsetBits)
	edgeOffset := uint32((ec >> StepOffsetBits) & MaxEdgeOffset)
	stepOffset := ec & MaxStepOffset
	vertex := graph.Vertex(vertexIndex)

	if edgeOffset == MaxEdgeOffset && stepOffset == MaxStepOffset {
		// it is a vertex and not a step
		return coord, g.VertexAccessible(vertex, nn.trans)
	}
	edge := graph.Edge(g.FirstOut[vertex] + edgeOffset)
	edgeAccessible := g.EdgeAccessible(edge, nn.trans)
	return coord, edgeAccessible
}

// distance returns the squared euclidian distance
func distance(x, y geo.Coordinate) float64 {
	return (x.Lat-y.Lat)*(x.Lat-y.Lat) + (x.Lng-y.Lng)*(x.Lng-y.Lng)
}

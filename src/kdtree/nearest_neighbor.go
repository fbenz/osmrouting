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
	e             ellipsoid.Ellipsoid
	clusterKdTree ClusterKdTree
)

func init() {
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

// NearestNeighbor returns -1 if the way is on the overlay graph
// No fail strategy: a nearest point on the overlay graph is always returned if no point
// is found in the clusters.
func NearestNeighbor(x geo.Coordinate, trans graph.Transport) Location {
	// first search on the overlay graph
	overlay := clusterKdTree.Overlay
	bestStepIndex, coordOverlay, foundPoint := binarySearch(overlay, x, 0, overlay.EncodedStepLen()-1,
		true /* compareLat */, trans)
	minDistance, _ := e.To(x.Lat, x.Lng, coordOverlay.Lat, coordOverlay.Lng)

	// then search on all clusters where the point is inside the bounding box of the cluster
	clusterIndex := -1
	for i, b := range clusterKdTree.BBoxes {
		if b.Contains(x) {
			kdTree := clusterKdTree.Cluster[i]
			stepIndex, coord, ok := binarySearch(kdTree, x, 0, kdTree.EncodedStepLen()-1, true /* compareLat */, trans)
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
		Graph:   overlay.Graph,
		EC:      overlay.EncodedStep(bestStepIndex),
		Cluster: clusterIndex,
	}
}

// binarySearch in one k-d tree. The index, the coordinate, and if the returned step/vertex
// is accessible are returned.
func binarySearch(kdTree *KdTree, x geo.Coordinate, start, end int, compareLat bool,
	trans graph.Transport) (int, geo.Coordinate, bool) {

	if end <= start {
		// end of the recursion
		startCoord, startAccessible := decodeCoordinate(kdTree, start, trans)
		return start, startCoord, startAccessible
	}

	middle := (end-start)/2 + start
	middleCoord, middleAccessible := decodeCoordinate(kdTree, middle, trans)

	// exact hit
	if middleAccessible && x.Lat == middleCoord.Lat && x.Lng == middleCoord.Lng {
		return middle, middleCoord, middleAccessible
	}

	middleDist := math.Inf(1)
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
		recIndex, recCoord, recAccessible = binarySearch(kdTree, x, start, middle-1, !compareLat, trans)
		if !recAccessible {
			// other half -> right
			recIndex, recCoord, recAccessible = binarySearch(kdTree, x, middle+1, end, !compareLat, trans)
			bothHalfs = true
		}
	} else {
		// right
		recIndex, recCoord, recAccessible = binarySearch(kdTree, x, middle+1, end, !compareLat, trans)
		if !recAccessible {
			// other half -> left
			recIndex, recCoord, recAccessible = binarySearch(kdTree, x, start, middle-1, !compareLat, trans)
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
	// test whether the current best distance circle crosses the plane
	if bestDistance >= distToPlane || distToPlane < 1e-6 {
		// search on the other half
		if !left {
			// left
			recIndex2, recCoord2, recAccessible2 = binarySearch(kdTree, x, start, middle-1, !compareLat, trans)
		} else {
			// right
			recIndex2, recCoord2, recAccessible2 = binarySearch(kdTree, x, middle+1, end, !compareLat, trans)
		}
	}

	bestDistance2 := math.Inf(1)
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
func decodeCoordinate(t *KdTree, i int, trans graph.Transport) (geo.Coordinate, bool) {
	g := t.Graph
	ec := t.EncodedStep(i)
	coord := geo.DecodeCoordinate(t.EncodedCoordinates[2*i], t.EncodedCoordinates[2*i+1])

	vertexIndex := ec >> (EdgeOffsetBits + StepOffsetBits)
	edgeOffset := uint32((ec >> StepOffsetBits) & MaxEdgeOffset)
	stepOffset := ec & MaxStepOffset
	vertex := graph.Vertex(vertexIndex)

	if edgeOffset == MaxEdgeOffset && stepOffset == MaxStepOffset {
		// it is a vertex and not a step
		return coord, g.VertexAccessible(vertex, trans)
	}
	edge := graph.Edge(g.FirstOut[vertex] + edgeOffset)
	edgeAccessible := g.EdgeAccessible(edge, trans)
	return coord, edgeAccessible
}

// distance returns the squared euclidian distance
func distance(x, y geo.Coordinate) float64 {
	return (x.Lat-y.Lat)*(x.Lat-y.Lat) + (x.Lng-y.Lng)*(x.Lng-y.Lng)
}

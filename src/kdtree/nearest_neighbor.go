package kdtree

import (
	"alg"
	"ellipsoid"
	"errors"
	"fmt"
	"geo"
	"graph"
	"log"
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
		clusterKdTrees[i] = &KdTree{Graph: g, EncodedSteps: encodedSteps, EncodedCoordinates: encodedCoordinates}
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
	overlayKdTree := &KdTree{Graph: clusterGraph.Overlay, EncodedSteps: encodedSteps, EncodedCoordinates: encodedCoordinates}

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

	clusterKdTree = ClusterKdTree{Overlay: overlayKdTree, Cluster: clusterKdTrees, BBoxes: bboxes}
	return nil
}

// NearestNeighbor returns -1 if the way is on the overlay graph
// No fail strategy: a nearest point on the overlay graph is always returned if no point
// is found in the clusters.
func NearestNeighbor(x geo.Coordinate, forward bool, trans graph.Transport) (int, []graph.Way) {
	edges := []graph.Edge(nil)
	steps := []geo.Coordinate(nil)

	// first search on the overlay graph
	overlay := clusterKdTree.Overlay
	bestStepIndex, coordOverlay, foundPoint := binarySearch(overlay, x, 0, overlay.EncodedStepLen()-1,
		true /* compareLat */, trans, &edges)
	minDistance, _ := e.To(x.Lat, x.Lng, coordOverlay.Lat, coordOverlay.Lng)

	// for debugging
	//bestCoord := coordOverlay

	// then search on all clusters where the point is inside the bounding box of the cluster
	clusterIndex := -1
	for i, b := range clusterKdTree.BBoxes {
		if b.Contains(x) {
			kdTree := clusterKdTree.Cluster[i]
			stepIndex, coord, ok := binarySearch(kdTree, x, 0, kdTree.EncodedStepLen()-1, true /* compareLat */, trans, &edges)
			dist, _ := e.To(x.Lat, x.Lng, coord.Lat, coord.Lng)

			if ok && (!foundPoint || dist < minDistance) {
				foundPoint = true
				minDistance = dist
				bestStepIndex = stepIndex
				clusterIndex = i

				// for debugging
				//bestCoord = coord
			}
		}
	}

	// for debugging: linear search
	/*minD := math.Inf(1)
	minI := -1
	minCluster := -1
	var minC geo.Coordinate
	for i, b := range clusterKdTree.BBoxes {
		if b.Contains(x) {
			kdTree := clusterKdTree.Cluster[i]
			for j := 0; j < kdTree.EncodedStepLen(); j++ {
				c, oki := decodeCoordinate(kdTree, j, trans, &edges)
				dd, _ := e.To(x.Lat, x.Lng, c.Lat, c.Lng)
				if oki && dd < minD {
					minD = dd
					minI = j
					minC = c
					minCluster = i
				}
			}
		}
	}
	fmt.Printf("nearest neighbor for %v\n", x)
	fmt.Printf("binary search: min distance: %v at %v in cluster %v  %v\n", minDistance, bestStepIndex, clusterIndex, bestCoord)
	fmt.Printf("linear search: min distance: %v at %v in cluster %v  %v\n", minD, minI, minCluster, minC)*/

	if clusterIndex >= 0 {
		kdTree := clusterKdTree.Cluster[clusterIndex]
		return clusterIndex, decodeWays(kdTree.Graph, kdTree.EncodedStep(bestStepIndex), forward, trans, &edges, &steps)
	}
	log.Printf("no matching bounding box found for (%v, %v)", x.Lat, x.Lng)
	return clusterIndex, decodeWays(overlay.Graph, overlay.EncodedStep(bestStepIndex), forward, trans, &edges, &steps)
}

// binarySearch in one k-d tree. The index, the coordinate, and if the returned step/vertex
// is accessible are returned.
func binarySearch(kdTree *KdTree, x geo.Coordinate, start, end int, compareLat bool,
	trans graph.Transport, edges *[]graph.Edge) (int, geo.Coordinate, bool) {

	if end <= start {
		// end of the recursion
		startCoord, startAccessible := decodeCoordinate(kdTree, start, trans, edges)
		return start, startCoord, startAccessible
	}

	middle := (end-start)/2 + start
	middleCoord, middleAccessible := decodeCoordinate(kdTree, middle, trans, edges)

	// exact hit
	if middleAccessible && x.Lat == middleCoord.Lat && x.Lng == middleCoord.Lng {
		return middle, middleCoord, middleAccessible
	}

	middleDist := math.Inf(1)
	if middleAccessible {
		middleDist, _ = e.To(x.Lat, x.Lng, middleCoord.Lat, middleCoord.Lng)
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
		recIndex, recCoord, recAccessible = binarySearch(kdTree, x, start, middle-1, !compareLat, trans, edges)
		if !recAccessible {
			// other half -> right
			recIndex, recCoord, recAccessible = binarySearch(kdTree, x, middle+1, end, !compareLat, trans, edges)
			bothHalfs = true
		}
	} else {
		// right
		recIndex, recCoord, recAccessible = binarySearch(kdTree, x, middle+1, end, !compareLat, trans, edges)
		if !recAccessible {
			// other half -> left
			recIndex, recCoord, recAccessible = binarySearch(kdTree, x, start, middle-1, !compareLat, trans, edges)
			bothHalfs = true
		}
	}
	bestDistance, _ := e.To(x.Lat, x.Lng, recCoord.Lat, recCoord.Lng)

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
		distToPlane, _ = e.To(middleCoord.Lat, x.Lng, x.Lat, x.Lng)
	} else {
		distToPlane, _ = e.To(x.Lat, middleCoord.Lng, x.Lat, x.Lng)
	}

	var recIndex2 int
	var recCoord2 geo.Coordinate
	recAccessible2 := false
	// test whether the current best distance circle crosses the plane
	//distToPlane -= 0.002
	if bestDistance >= distToPlane {
		// search on the other half
		if !left {
			// left
			recIndex2, recCoord2, recAccessible2 = binarySearch(kdTree, x, start, middle-1, !compareLat, trans, edges)
		} else {
			// right
			recIndex2, recCoord2, recAccessible2 = binarySearch(kdTree, x, middle+1, end, !compareLat, trans, edges)
		}
	}

	bestDistance2 := math.Inf(1)
	if recAccessible2 {
		bestDistance2, _ = e.To(x.Lat, x.Lng, recCoord2.Lat, recCoord2.Lng)
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
func decodeCoordinate(t *KdTree, i int, trans graph.Transport, edges *[]graph.Edge) (geo.Coordinate, bool) {
	g := t.Graph
	ec := t.EncodedStep(i)
	coord := geo.DecodeCoordinate(t.EncodedCoordinates[2*i], t.EncodedCoordinates[2*i+1])

	vertexIndex := ec >> (EdgeOffsetBits + StepOffsetBits)
	edgeOffset := (ec >> StepOffsetBits) & MaxEdgeOffset
	stepOffset := ec & MaxStepOffset
	vertex := graph.Vertex(vertexIndex)

	if edgeOffset == MaxEdgeOffset && stepOffset == MaxStepOffset {
		// it is a vertex and not a step
		return coord, g.VertexAccessible(vertex, trans)
	}

	var edge graph.Edge
	var edgeAccessible bool
	switch t := g.(type) {
	case *graph.GraphFile:
		(*edges) = t.VertexRawEdges(vertex, *edges)
		edge = (*edges)[edgeOffset]
		edgeAccessible = t.EdgeAccessible(edge, trans)
	case *graph.OverlayGraphFile:
		(*edges) = t.VertexRawEdges(vertex, *edges)
		edge = (*edges)[edgeOffset]
		edgeAccessible = t.EdgeAccessible(edge, trans)
	default:
		panic("unexpected graph implementation")
	}

	return coord, edgeAccessible
}

// decodeWays returns one or two partital edges that are therefore called ways. Even for a vertex a
// way is returned so that later code only has to consider ways. The other cases are explained below.
func decodeWays(g graph.Graph, ec uint64, forward bool, trans graph.Transport, edges *[]graph.Edge, steps *[]geo.Coordinate) []graph.Way {
	vertexIndex := ec >> (EdgeOffsetBits + StepOffsetBits)
	edgeOffset := (ec >> StepOffsetBits) & MaxEdgeOffset
	offset := ec & MaxStepOffset
	vertex := graph.Vertex(vertexIndex)

	if edgeOffset == MaxEdgeOffset && offset == MaxStepOffset {
		// The easy case, where we hit some vertex exactly.
		w := make([]graph.Way, 1)
		target := g.VertexCoordinate(vertex)
		w[0] = graph.Way{Length: 0, Vertex: vertex, Steps: nil, Target: target}
		return w
	}

	var edge graph.Edge
	oneway := false
	switch t := g.(type) {
	case *graph.GraphFile:
		(*edges) = t.VertexRawEdges(vertex, *edges)
		edge = (*edges)[edgeOffset]
		oneway = alg.GetBit(t.Oneway, uint(edge))
	case *graph.OverlayGraphFile:
		(*edges) = t.VertexRawEdges(vertex, *edges)
		edge = (*edges)[edgeOffset]
		oneway = alg.GetBit(t.Oneway, uint(edge))
	default:
		panic("unexpected graph implementation")
	}
	if trans == graph.Foot {
		oneway = false
	}

	t1 := vertex                       // start vertex
	t2 := g.EdgeOpposite(edge, vertex) // end vertex

	// now we can allocate the way corresponding to (edge,offset),
	// but there are three cases to consider:
	// - if the way is bidirectional we have to compute both directions,
	//   if forward == true the from the offset two both endpoints,
	//   and the reverse otherwise
	// - if the way is unidirectional then we have to compute the way
	//   from the StartPoint to offset if forward == false
	// - otherwise we have to compute the way from offset to the EndPoint
	// Strictly speaking only the second case needs an additional binary
	// search in the form of edge.StartPoint, but let's keep this simple
	// for now.
	(*steps) = g.EdgeSteps(edge, vertex, *steps)
	s := *steps

	b1 := make([]geo.Coordinate, len(s[:offset]))
	b2 := make([]geo.Coordinate, len(s[offset+1:]))
	copy(b1, s[:offset])
	copy(b2, s[offset+1:])
	l1 := geo.StepLength(s[:offset+1])
	l2 := geo.StepLength(s[offset:])
	t1Coord := g.VertexCoordinate(t1)
	t2Coord := g.VertexCoordinate(t2)
	d1, _ := e.To(t1Coord.Lat, t1Coord.Lng, s[0].Lat, s[0].Lng)
	d2, _ := e.To(t2Coord.Lat, t2Coord.Lng, s[len(s)-1].Lat, s[len(s)-1].Lng)
	l1 += d1
	l2 += d2
	target := s[offset]

	if !forward {
		reverse(b2)
	} else {
		reverse(b1)
	}

	var w []graph.Way
	if !oneway {
		w = make([]graph.Way, 2) // bidirectional
		w[0] = graph.Way{Length: l1, Vertex: t1, Steps: b1, Forward: forward, Target: target}
		w[1] = graph.Way{Length: l2, Vertex: t2, Steps: b2, Forward: forward, Target: target}
	} else {
		w = make([]graph.Way, 1) // one way
		if forward {
			w[0] = graph.Way{Length: l2, Vertex: t2, Steps: b2, Forward: forward, Target: target}
		} else {
			w[0] = graph.Way{Length: l1, Vertex: t1, Steps: b1, Forward: forward, Target: target}
		}
	}
	return w
}

func reverse(steps []geo.Coordinate) {
	for i, j := 0, len(steps)-1; i < j; i, j = i+1, j-1 {
		steps[i], steps[j] = steps[j], steps[i]
	}
}

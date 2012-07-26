package kdtree

import (
	"ellipsoid"
	"errors"
	"fmt"
	"geo"
	"graph"
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
	// TODO precompute coordinates for faster live queries?
	dummyCoordinates := make([]geo.Coordinate, 0)

	clusterKdTrees := make([]*KdTree, len(clusterGraph.Cluster))
	for i, g := range clusterGraph.Cluster {
		clusterDir := fmt.Sprintf("cluster%d/kdtree.ftf", i+1)
		var encodedSteps []uint32
		err := mm.Open(path.Join(base, clusterDir), &encodedSteps)
		if err != nil {
			return err
		}
		clusterKdTrees[i] = &KdTree{Graph: g, EncodedSteps: encodedSteps, Coordinates: dummyCoordinates}
	}

	var encodedSteps []uint32
	err := mm.Open(path.Join(base, "/overlay/kdtree.ftf"), &encodedSteps)
	if err != nil {
		return err
	}
	overlayKdTree := &KdTree{Graph: clusterGraph.Overlay, EncodedSteps: encodedSteps, Coordinates: dummyCoordinates}

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

	t := clusterKdTree.Overlay
	bestEncodedStep := binarySearch(t, t.EncodedSteps, x, true /* compareLat */, trans, &edges)
	coordOverlay := decodeCoordinate(t.Graph, bestEncodedStep, trans, &edges)
	minDistance, _ := e.To(x.Lat, x.Lng, coordOverlay.Lat, coordOverlay.Lng)

	clusterIndex := -1
	for i, b := range clusterKdTree.BBoxes {
		if b.Contains(x) {
			kdTree := clusterKdTree.Cluster[i]
			encodedStep := binarySearch(kdTree, kdTree.EncodedSteps, x, true /* compareLat */, trans, &edges)
			coord := decodeCoordinate(kdTree.Graph, encodedStep, trans, &edges)
			dist, _ := e.To(x.Lat, x.Lng, coord.Lat, coord.Lng)
			if dist < minDistance {
				minDistance = dist
				bestEncodedStep = encodedStep
				clusterIndex = i
			}
		}
	}

	g := clusterKdTree.Cluster[clusterIndex].Graph
	return clusterIndex, decodeWays(g, bestEncodedStep, forward, trans, &edges)
}

func binarySearch(kdTree *KdTree, nodes []uint32, x geo.Coordinate, compareLat bool,
	trans graph.Transport, edges *[]graph.Edge) uint32 {
	g := kdTree.Graph
	if len(nodes) == 0 {
		panic("nearestNeighbor: recursion to dead end")
	} else if len(nodes) == 1 {
		return nodes[0]
	}
	middle := len(nodes) / 2

	// exact hit
	middleCoord := decodeCoordinate(g, nodes[middle], trans, edges)
	if x.Lat == middleCoord.Lat && x.Lng == middleCoord.Lng {
		return nodes[middle]
	}

	// corner case where the nearest point can be on both sides of the middle
	if (compareLat && x.Lat == middleCoord.Lat) || (!compareLat && x.Lng == middleCoord.Lng) {
		// recursion on both halfs
		leftRecEnc := binarySearch(kdTree, nodes[:middle], x, !compareLat, trans, edges)
		rightRecEnc := binarySearch(kdTree, nodes[middle+1:], x, !compareLat, trans, edges)
		leftCoord := decodeCoordinate(g, leftRecEnc, trans, edges)
		rightCoord := decodeCoordinate(g, rightRecEnc, trans, edges)

		// TODO exact distance on Coordinates?
		distMiddle, _ := e.To(x.Lat, x.Lng, middleCoord.Lat, middleCoord.Lng)
		distRecursionLeft, _ := e.To(x.Lat, x.Lng, leftCoord.Lat, leftCoord.Lng)
		distRecursionRight, _ := e.To(x.Lat, x.Lng, rightCoord.Lat, rightCoord.Lng)
		if distRecursionLeft < distRecursionRight {
			if distRecursionLeft < distMiddle {
				return leftRecEnc
			}
			return nodes[middle]
		}
		if distRecursionRight < distMiddle {
			return rightRecEnc
		}
		return nodes[middle]
	}

	var left bool
	if compareLat {
		left = x.Lat < middleCoord.Lat
	} else {
		left = x.Lng < middleCoord.Lng
	}
	if left {
		// stop if there is nothing left of the middle
		if middle == 0 {
			return nodes[middle]
		}
		// recursion on the left half
		recEnc := binarySearch(kdTree, nodes[:middle], x, !compareLat, trans, edges)
		recCoord := decodeCoordinate(g, recEnc, trans, edges)

		// compare middle and result from the left
		distMiddle, _ := e.To(x.Lat, x.Lng, middleCoord.Lat, middleCoord.Lng)
		distRecursion, _ := e.To(x.Lat, x.Lng, recCoord.Lat, recCoord.Lng)
		if distMiddle < distRecursion {
			return nodes[middle]
		}
		return recEnc
	}
	// stop if there is nothing right of the middle
	if middle == len(nodes)-1 {
		return nodes[middle]
	}
	// recursion on the right half
	recEnc := binarySearch(kdTree, nodes[middle+1:], x, !compareLat, trans, edges)
	recCoord := decodeCoordinate(g, recEnc, trans, edges)

	// compare middle and result from the right
	distMiddle, _ := e.To(x.Lat, x.Lng, middleCoord.Lat, middleCoord.Lng)
	distRecursion, _ := e.To(x.Lat, x.Lng, recCoord.Lat, recCoord.Lng)
	if distMiddle < distRecursion {
		return nodes[middle]
	}
	return recEnc
}

func decodeCoordinate(g graph.Graph, ec uint32, trans graph.Transport, edges *[]graph.Edge) geo.Coordinate {
	vertexIndex := ec >> (EdgeOffsetBits + StepOffsetBits)
	edgeOffset := (ec >> StepOffsetBits) & MaxEdgeOffset
	stepOffset := ec & MaxStepOffset
	vertex := graph.Vertex(vertexIndex)
	if edgeOffset == MaxEdgeOffset && stepOffset == MaxStepOffset {
		// it is a vertex and not a step
		return g.VertexCoordinate(vertex)
	}

	(*edges) = g.VertexEdges(vertex, true /* out */, trans, *edges)
	for i, e := range *edges {
		if i == int(edgeOffset) {
			steps := g.EdgeSteps(e, vertex)
			return steps[stepOffset]
		}
		i++
	}
	panic("incorrect encoding: no matching edge found")
}

func decodeWays(g graph.Graph, ec uint32, forward bool, trans graph.Transport, edges *[]graph.Edge) []graph.Way {
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
	(*edges) = g.VertexEdges(vertex, true /* out */, trans, *edges)
	for i, e := range *edges {
		if i == int(edgeOffset) {
			edge = e
			break
		}
		i++
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
	steps := g.EdgeSteps(edge, vertex)
	b1 := make([]geo.Coordinate, len(steps[:offset]))
	b2 := make([]geo.Coordinate, len(steps[offset+1:]))
	copy(b1, steps[:offset])
	copy(b2, steps[offset+1:])
	l1 := geo.StepLength(steps[:offset+1])
	l2 := geo.StepLength(steps[offset:])
	t1Coord := g.VertexCoordinate(t1)
	t2Coord := g.VertexCoordinate(t2)
	d1, _ := e.To(t1Coord.Lat, t1Coord.Lng, steps[0].Lat, steps[0].Lng)
	d2, _ := e.To(t2Coord.Lat, t2Coord.Lng, steps[len(steps)-1].Lat, steps[len(steps)-1].Lng)
	l1 += d1
	l2 += d2
	target := steps[offset]

	if !forward {
		reverse(b2)
	} else {
		reverse(b1)
	}

	// TODO check oneway based on transport
	oneway := true

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

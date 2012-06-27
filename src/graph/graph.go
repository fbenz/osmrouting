package graph

import (
	"ellipsoid"
	"sort"
)

type Node uint // TODO uint or something else?
type Edge uint

// Way is a "partial edge" that is returned by the k-d tree
type Way struct {
	Length  float64
	Node    Node // StartPoint or EndPoint
	Steps   []Step
	Target  Step
	Forward bool
}

type Graph interface {
	NodeCount() int
	EdgeCount() int
	Positions() Positions

	NodeEdges(Node) (Edge, Edge)
	NodeLatLng(Node) (float64, float64)

	EdgeLength(Edge) float64
	EdgeStartPoint(Edge) Node
	EdgeEndPoint(Edge) Node
	EdgeReverse(Edge) (Edge, bool)
	EdgeSteps(Edge) []Step
}

// Positions is used for both creating the k-d tree in the preprocessing phase
// and for doing the nearest neighbor lookup during runtime.
type Positions interface {
	Len() int
	Lat(int) float64
	Lng(int) float64
	Step(int) Step
	Ways(int, bool) []Way // index, forward (i.e. looking at the edge in forward or in backward order)
}

type Step struct {
	Lat float64
	Lng float64
}

// Implementation

type graphFile struct {
	// ellipsoid for distance calculations
	geo ellipsoid.Ellipsoid
	// vertices maps a vertex index to the index of its first out edge
	vertices []uint32
	// edges maps edge indices to the index of the vertex endpoint
	edges []uint32
	// rev_edges maps an edge index to its reverse
	revEdges []uint32
	// map edge indices to distances
	distances []float64
	// map node indices to positions (lat: 2 * i, lng: 2 * i + 1)
	positions []float64
	// map an edge index to the inex of the first intermediate step
	steps []uint32
	// positions as interleaved lat/lng pairs as above
	stepPositions []float64
}

func Open(base string) (Graph, error) {
	graph := graphFile{}

	graph.geo = ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter,
		ellipsoid.Longitude_is_symmetric, ellipsoid.Bearing_is_symmetric)

	var err error
	graph.vertices, err = MmapFileUint32(base, "vertices.ftf")
	if err != nil {
		return nil, err
	}

	graph.edges, err = MmapFileUint32(base, "edges.ftf")
	if err != nil {
		return nil, err
	}

	graph.revEdges, err = MmapFileUint32(base, "rev_edges.ftf")
	if err != nil {
		return nil, err
	}

	graph.distances, err = MmapFileFloat64(base, "distances.ftf")
	if err != nil {
		return nil, err
	}

	graph.positions, err = MmapFileFloat64(base, "positions.ftf")
	if err != nil {
		return nil, err
	}

	graph.steps, err = MmapFileUint32(base, "steps.ftf")
	if err != nil {
		return nil, err
	}

	graph.stepPositions, err = MmapFileFloat64(base, "step_positions.ftf")
	if err != nil {
		return nil, err
	}

	return &graph, nil
}

func reverse(steps []Step) {
	for i, j := 0, len(steps)-1; i < j; i, j = i+1, j-1 {
		steps[i], steps[j] = steps[j], steps[i]
	}
}

// Interface Implementation

// Graph

func (g *graphFile) NodeCount() int {
	return len(g.vertices) - 1
}

func (g *graphFile) EdgeCount() int {
	return len(g.edges)
}

func (g *graphFile) Positions() Positions {
	return g
}

// Node

func (g *graphFile) NodeEdges(i Node) (Edge, Edge) {
	// The check is done anyway when accessing g.vertices
	/*if i >= Node(g.NodeCount()) {
		panic("Node access out of bounds.")
	}*/

	start := g.vertices[i]
	end := g.vertices[i+1] - 1
	return Edge(start), Edge(end)
}

func (g *graphFile) NodeLatLng(i Node) (float64, float64) {
	lat := g.positions[2*i]
	lng := g.positions[2*i+1]
	return lat, lng
}

// Edge

func (g *graphFile) EdgeLength(i Edge) float64 {
	return g.distances[i]
}

func (g *graphFile) EdgeStartPoint(i Edge) Node {
	j := sort.Search(len(g.vertices),
		func(k int) bool { return Edge(g.vertices[k]) > i }) - 1
	return Node(j)
}

func (g *graphFile) EdgeEndPoint(i Edge) Node {
	index := g.edges[i]
	return Node(index)
}

func (g *graphFile) EdgeReverse(i Edge) (Edge, bool) {
	index := g.revEdges[i]
	if Edge(index) == i {
		return Edge(index), false
	}
	return Edge(index), true
}

func (g *graphFile) EdgeSteps(i Edge) []Step {
	start := g.steps[i]
	stop := g.steps[i+1]
	revert := false
	if start == stop {
		revIndex := g.revEdges[i]
		if Edge(revIndex) != i {
			start = g.steps[revIndex]
			stop = g.steps[revIndex+1]
			revert = true
		}
	}
	size := stop - start
	steps := make([]Step, size)
	for j, _ := range steps {
		lat := g.stepPositions[2*(int(start)+j)]
		lng := g.stepPositions[2*(int(start)+j)+1]
		steps[j] = Step{lat, lng}
	}
	if revert {
		reverse(steps)
	}
	return steps
}

// Positions

func (g *graphFile) Len() int {
	return g.NodeCount() + len(g.stepPositions)/2
}

func (g *graphFile) Lat(i int) float64 {
	if i < g.NodeCount() {
		lat, _ := g.NodeLatLng(Node(i))
		return lat
	}
	i -= g.NodeCount()
	return g.stepPositions[2*i]
}

func (g *graphFile) Lng(i int) float64 {
	if i < g.NodeCount() {
		_, lng := g.NodeLatLng(Node(i))
		return lng
	}
	i -= g.NodeCount()
	return g.stepPositions[2*i+1]
}

func (g *graphFile) Step(i int) Step {
	if i < g.NodeCount() {
		lat, lng := g.NodeLatLng(Node(i))
		return Step{lat, lng}
	}
	i -= g.NodeCount()
	return Step{g.stepPositions[2*i], g.stepPositions[2*i+1]}
}

func wayLength(steps []Step, geo ellipsoid.Ellipsoid) float64 {
	if len(steps) == 0 {
		return 0.0
	}
	total := 0.0
	prev := steps[0]
	for _, step := range steps {
		distance, _ := geo.To(prev.Lat, prev.Lng, step.Lat, step.Lng)
		total += distance
		prev = step
	}
	return total
}

func (g *graphFile) Ways(i int, forward bool) []Way {
	if i < g.NodeCount() {
		// The easy case, where we hit some node exactly.
		w := make([]Way, 1)
		n := Node(i)
		lat, lng := g.NodeLatLng(n)
		target := Step{lat, lng}
		w[0] = Way{Length: 0, Node: n, Steps: nil, Target: target}
		return w
	}
	i -= g.NodeCount()
	// find the (edge, offset) pair for step i
	edge := Edge(sort.Search(len(g.steps),
		func(j int) bool { return uint(g.steps[j]) > uint(i) }) - 1)
	offset := uint32(i) - g.steps[edge]
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
	steps := g.EdgeSteps(edge)
	b1 := make([]Step, len(steps[:offset]))
	b2 := make([]Step, len(steps[offset+1:]))
	copy(b1, steps[:offset])
	copy(b2, steps[offset+1:])
	l1 := wayLength(steps[:offset+1], g.geo)
	l2 := wayLength(steps[offset:], g.geo)
	t1 := g.EdgeStartPoint(edge)
	t2 := g.EdgeEndPoint(edge)
	t1Lat, t1Lng := g.NodeLatLng(t1)
	t2Lat, t2Lng := g.NodeLatLng(t2)
	d1, _ := g.geo.To(t1Lat, t1Lng, steps[0].Lat, steps[0].Lng)
	d2, _ := g.geo.To(t2Lat, t2Lng, steps[len(steps)-1].Lat, steps[len(steps)-1].Lng)
	l1 += d1
	l2 += d2
	target := steps[offset]

	if !forward {
		reverse(b2)
	} else {
		reverse(b1)
	}

	var w []Way
	if _, ok := g.EdgeReverse(edge); ok {
		w = make([]Way, 2) // bidirectional
		w[0] = Way{Length: l1, Node: t1, Steps: b1, Forward: forward, Target: target}
		w[1] = Way{Length: l2, Node: t2, Steps: b2, Forward: forward, Target: target}
	} else {
		w = make([]Way, 1) // one way
		if forward {
			w[0] = Way{Length: l2, Node: t2, Steps: b2, Forward: forward, Target: target}
		} else {
			w[0] = Way{Length: l1, Node: t1, Steps: b1, Forward: forward, Target: target}
		}
	}
	return w
}

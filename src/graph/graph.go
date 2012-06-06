package graph

import (
	"os"
	"path"
	"reflect"
	"sort"
	"syscall"
	"unsafe"
	"ellipsoid"
	"fmt"
)

type Node interface {
	Edges() []Edge
	LatLng() (float64, float64)
}

type Edge interface {
	Length() float64
	StartPoint() Node // e.g. via binary search on the node array
	EndPoint() Node
	ReverseEdge() (Edge, bool)
	Steps() []Step
	// Label() string
}

// "partial edge" is returned by the k-d tree
type Way struct {
	Length float64
	Node   Node // StartPoint or EndPoint
	Steps  []Step
}

type Graph interface {
	NodeCount() int
	EdgeCount() int
	Node(uint) Node
	Edge(uint) Edge
	Positions() Positions
}

// Implementation sketch (wrapper around graph):
// The graph is loaded before and is given to this.
// Init: Both position files (vertexes and inner steps in an edge) are alread loaded
//   for the graph (Nodes.LatLng(), Edge.Steps() - Step.Lat/Lng).
//   Thus despite storing a pointer the graph, nothing to do here. 
// For every method where an index is given, there is a branch
//   if index < Graph.NodeCount()
//      work with Graph.Node(index)
//   else
//      work with the underlying Step array (index - Graph.NodeCount())
//      not possible efficiently with the current interface
//
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

func mapFile(base, name string) ([]byte, error) {
	file, err := os.Open(path.Join(base, name))
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	// Thanks to Windows compatibility file.Fd is not declared int...
	fdfu := file.Fd()
	fd := *(*int)(unsafe.Pointer(&fdfu))
	// This is bad. Slices have int size and capacity fields, which
	// means that we might truncate here. We can work around this issue
	// by using unsafe.Pointer internally and only convert to slices for
	// individual edge/step lists... But for now our files are small
	// and this works:
	size := int(info.Size())
	return syscall.Mmap(fd, 0, size, syscall.PROT_READ, syscall.MAP_PRIVATE)
}

func mapFileUint32(base, name string) ([]uint32, error) {
	m, err := mapFile(base, name)
	if err != nil {
		return nil, err
	}

	dh := (*reflect.SliceHeader)(unsafe.Pointer(&m))
	dh.Len /= 4
	dh.Cap /= 4
	return *(*[]uint32)(unsafe.Pointer(&m)), nil
}

func mapFileFloat64(base, name string) ([]float64, error) {
	m, err := mapFile(base, name)
	if err != nil {
		return nil, err
	}

	dh := (*reflect.SliceHeader)(unsafe.Pointer(&m))
	dh.Len /= 8
	dh.Cap /= 8
	return *(*[]float64)(unsafe.Pointer(&m)), nil
}

func Open(base string) (Graph, error) {
	graph := graphFile{}

	var err error
	graph.vertices, err = mapFileUint32(base, "vertices.ftf")
	if err != nil {
		return nil, err
	}

	graph.edges, err = mapFileUint32(base, "edges.ftf")
	if err != nil {
		return nil, err
	}

	graph.revEdges, err = mapFileUint32(base, "rev_edges.ftf")
	if err != nil {
		return nil, err
	}

	graph.distances, err = mapFileFloat64(base, "distances.ftf")
	if err != nil {
		return nil, err
	}

	graph.positions, err = mapFileFloat64(base, "positions.ftf")
	if err != nil {
		return nil, err
	}

	graph.steps, err = mapFileUint32(base, "steps.ftf")
	if err != nil {
		return nil, err
	}

	graph.stepPositions, err = mapFileFloat64(base, "step_positions.ftf")
	if err != nil {
		return nil, err
	}

	return &graph, nil
}

// Interface Implementation

type nodeReference struct {
	g     *graphFile
	index uint
}

type edgeReference struct {
	g     *graphFile
	index uint
}

// Graph

func (g *graphFile) NodeCount() int {
	return len(g.vertices) - 1
}

func (g *graphFile) EdgeCount() int {
	return len(g.edges)
}

func (g *graphFile) Node(i uint) Node {
	if i >= uint(g.NodeCount()) {
		panic("Node access out of bounds.")
	}
	return nodeReference{g, i}
}

func (g *graphFile) Edge(i uint) Edge {
	if i >= uint(g.EdgeCount()) {
		panic("Edge access out of bounds.")
	}
	return edgeReference{g, i}
}

func (g *graphFile) Positions() Positions {
	return g
}

// Node

func (ref nodeReference) Edges() []Edge {
	start := ref.g.vertices[ref.index]
	stop := ref.g.vertices[ref.index+1]
	degree := stop - start
	edges := make([]Edge, degree)
	for i, _ := range edges {
		edges[i] = edgeReference{ref.g, uint(ref.g.edges[int(start)+i])}
	}
	return edges
}

func (ref nodeReference) LatLng() (float64, float64) {
	lat := ref.g.positions[2*ref.index]
	lng := ref.g.positions[2*ref.index+1]
	return lat, lng
}

// Edge

func (ref edgeReference) Length() float64 {
	return ref.g.distances[ref.index]
}

func (ref edgeReference) StartPoint() Node {
	i := sort.Search(len(ref.g.vertices),
		func(i int) bool { return uint(ref.g.vertices[i]) > ref.index })
	return nodeReference{ref.g, uint(i - 1)}
}

func (ref edgeReference) EndPoint() Node {
	index := ref.g.edges[ref.index]
	return nodeReference{ref.g, uint(index)}
}

func (ref edgeReference) ReverseEdge() (Edge, bool) {
	index := ref.g.revEdges[ref.index]
	if uint(index) == ref.index {
		return ref, false
	}
	return edgeReference{ref.g, uint(index)}, true
}

func (ref edgeReference) Steps() []Step {
	start := ref.g.steps[ref.index]
	stop := ref.g.steps[ref.index+1]
	size := stop - start
	steps := make([]Step, size)
	for i, _ := range steps {
		lat := ref.g.stepPositions[2*(int(start)+i)]
		lng := ref.g.stepPositions[2*(int(start)+i)+1]
		steps[i] = Step{lat, lng}
	}
	return steps
}

// Positions

func (g *graphFile) Len() int {
	return g.NodeCount() + len(g.stepPositions) / 2
}

func (g *graphFile) Lat(i int) float64 {
	if i < g.NodeCount() {
		lat, _ := g.Node(uint(i)).LatLng()
		return lat
	}
	i -= g.NodeCount()
	return g.stepPositions[2*i]
}

func (g *graphFile) Lng(i int) float64 {
	if i < g.NodeCount() {
		_, lng := g.Node(uint(i)).LatLng()
		return lng
	}
	i -= g.NodeCount()
	return g.stepPositions[2*i+1]
}

func (g *graphFile) Step(i int) Step {
	index := g.steps[i]
	return Step{g.stepPositions[2*index], g.stepPositions[2*index+1]}
}

func wayLength(steps []Step, geo ellipsoid.Ellipsoid) float64 {
	if len(steps) == 0 {
		return 0.0
	}
	total := 0.0
	prev  := steps[0]
	for _, step := range steps {
		distance, _ := geo.To(prev.Lat, prev.Lng, step.Lat, step.Lng)
		total += distance
		prev = step
	}
	return total
}

func reverse(steps []Step) {
	for i, j := 0, len(steps)-1; i < j; i, j = i+1, j-1 {
		steps[i], steps[j] = steps[j], steps[i]
	}
}

func (g *graphFile) Ways(i int, forward bool) []Way {
    if i < g.NodeCount() {
        w := make([]Way, 1)
		w[0] = Way{Length: 0, Node: g.Node(uint(i)), Steps: nil}
        return w
    }
    i -= g.NodeCount()
    // find the (edge, offset) pair for step i
	edgeIndex := sort.Search(len(g.steps),
		func(j int) bool { return uint(g.steps[j]) > uint(i) }) - 1
	
	fmt.Printf("Edge: %v\n", edgeIndex)
	offset := uint32(i) - g.steps[edgeIndex]
	fmt.Printf("Offset: %v\n", offset)
	edge   := g.Edge(uint(edgeIndex))
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
	steps := edge.Steps()
	fmt.Printf("Steps: %v\n", steps)
	b1 := steps[:offset + 1]
	b2 := steps[offset:]
	l1 := wayLength(b1, g.geo)
	l2 := wayLength(b2, g.geo)
	t1 := edge.StartPoint()
	t2 := edge.EndPoint()
	
	if !forward {
		reverse(b1)
		reverse(b2)
	}
	
	var w []Way
	if _, ok := edge.ReverseEdge(); ok {
		w = make([]Way, 2) // bidirectional
		w[0] = Way{Length: l1, Node: t1, Steps: b1}
		w[1] = Way{Length: l2, Node: t2, Steps: b2}
	} else {
		w = make([]Way, 1) // one way
		if forward {
			w[0] = Way{Length: l2, Node: t2, Steps: b2}
		} else {
			w[0] = Way{Length: l1, Node: t1, Steps: b1}
		}
	}
	return w
}

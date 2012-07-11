// TODO:
// - Make the debugging stuff optional again...
// - Split this file and cleanup the individual passes.
// - Add missing features:
//   * We need to parse relations, since these are used to encode
//     access restrictions between different roads.
//     Look at the university main entrance for a nice example.
//   * Obviously, we need max_speed information. However, this
//     is simply ridiculously convoluted.
//     max_speed is implicit for many roads and depends both
//     on the country and on whether or not the road lies
//     in a residential area. This means that we will have to
//     parse the corresponding relations and then do a few point
//     in polygon tests for any road without max_speed...
//   * Access restrictions. Your car can't climb stairs.

package main

import (
	"alg"
	"encoding/binary"
	"ellipsoid"
	"flag"
	"fmt"
	"geo"
	"os"
	"osm"
	"umath"
)

// Street graph visitor, skips unimportant ways.
type StreetGraph struct {
	File   *os.File
	Access osm.AccessType
}

type StreetGraphVisitor struct {
	Access  osm.AccessType
	Visitor osm.Visitor
}

func (s *StreetGraphVisitor) VisitNode(node osm.Node) {
	s.Visitor.VisitNode(node)
}

func (s *StreetGraphVisitor) VisitRelation(relation osm.Relation) {
	s.Visitor.VisitRelation(relation)
}

func (s *StreetGraphVisitor) VisitWay(way osm.Way) {
	// Skip trivial ways
	if len(way.Nodes) < 2 {
		return
	}
	
	// Skip non-roads
	mask := osm.AccessMask(way)
	if mask & s.Access == 0 {
		return
	}
	
	// Now parse the oneway tag... if it's broken we cannot (safely) use
	// this street.
	safe := osm.NormalizeOneway(way)
	if !safe {
		return
	}
	
	s.Visitor.VisitWay(way)
}

func Traverse(graph StreetGraph, visitor osm.Visitor) error {
	filter := &StreetGraphVisitor{
		Access:  graph.Access,
		Visitor: visitor,
	}
	return osm.ParseFile(graph.File, filter)
}

// Output a slice of fixed size data in little endian format.
func Output(name string, data interface{}) error {
	output, err := os.Create(name)
	if err != nil {
		return err
	}
	err = binary.Write(output, binary.LittleEndian, data)
	if err != nil {
		return err
	}
	err = output.Close()
	if err != nil {
		return err
	}
	return nil
}

// PASS 1
// We want to find all nodes which are relevant for the street-graph.
// In particular we need to find all nodes which are either endpoints of
// highways/junctions *or* intersection points of different highways/junctions.
// The latter part makes this expensive. Basically, we need to flag all interior
// nodes we see and add every node we see more than once to the subgraph.
// Corner points are always part of the street graph.

type NodeIndices map[int64] uint32

type Subgraph struct {
	Indices NodeIndices
	Visited alg.BitVector
	Size    uint32
}

func (s *Subgraph) VisitNode(node osm.Node) {
}

func (s *Subgraph) VisitRelation(relation osm.Relation) {
}

func (s *Subgraph) VisitWay(way osm.Way) {
	// Add all flagged nodes along with the endpoints
	for i, nodeId := range way.Nodes {
		if _, ok := s.Indices[nodeId]; ok {
			// node is already in the graph
			continue
		}
		if i == 0 || i == len(way.Nodes) - 1 || s.Visited.Get(nodeId) {
			s.Indices[nodeId] = s.Size
			s.Size++
		}
	}
	
	// Flag all nodes in on the way
	for _, nodeId := range way.Nodes {
		s.Visited.Set(nodeId, true)
	}
}

func InducedSubgraph(graph StreetGraph) (*Subgraph, error) {
	visitor := &Subgraph{
		Indices: NodeIndices {},
		Visited: alg.NewBitVector(64),
		Size:    0,
	}
	
	err := Traverse(graph, visitor)
	if err != nil {
		return nil, err
	}
	
	// the bitvector is still needed for pass 3
	return visitor, nil
}

// Pass 2
// Gather the relevant node attributes: out-degrees and positions.
// At this point we could allow a small interlude which sorts the node
// indices, but we do not do this currently.

type NodeAttributes struct {
	*Subgraph
	Degrees   []uint32
	Positions []int32
}

func (v *NodeAttributes) VisitNode(node osm.Node) {
	if i, ok := v.Indices[node.Id]; ok {
		lat, lng := node.Position.Encode()
		v.Positions[2*i]   = lat
		v.Positions[2*i+1] = lng
	}
}

func (v *NodeAttributes) VisitRelation(relation osm.Relation) {
}

func (v *NodeAttributes) VisitWay(way osm.Way) {
	isOneway := way.Attributes["oneway"] == "true"
	
	prevIndex, ok := v.Indices[way.Nodes[0]]
	if !ok {
		panic("First vertex of a path is not in the graph")
	}
	
	broken := true // this just guards against parser bugs
	
	for _, osmId := range way.Nodes[1:] {
		nodeIndex, ok := v.Indices[osmId]
		if !ok {
			continue
		}
		broken = false
		
		// We always have an edge from the previous index to this one
		v.Degrees[prevIndex]++
		// The reverse edge only exists if this is a two-way road
		if !isOneway {
			v.Degrees[nodeIndex]++
		}
		
		prevIndex = nodeIndex
	}
	
	if broken {
		panic(fmt.Sprintf("Visited an edge without vertices: %v.\n", way))
	}
}

// sort.Interface
/*
func (v *NodeAttributes) Len() int {
	return int(v.Size)
}

func (v *NodeAttributes) Less(i, j int) bool {
	lat0, lng0 := v.Positions[2 * i], v.Positions[2 * i + 1]
	lat1, lng1 := v.Positions[2 * j], v.Positions[2 * j + 1]
	x0, y0 := uint32(lng0 + 180), uint32(lat0 + 90)
	x1, y1 := uint32(lng1 + 180), uint32(lat1 + 90)
	return HilbertLess(x0, y0, x1, y1)
}

func (v *NodeAttributes) Swap(i, j int) {
	v.Degrees[i], v.Degrees[j] = v.Degrees[j], v.Degrees[i]
	v.Positions[2 * i], v.Positions[2 * j] =
		v.Positions[2 * j], v.Positions[2 * i]
	v.Positions[2 * i + 1], v.Positions[2 * j + 1] =
		v.Positions[2 * j + 1], v.Positions[2 * i + 1]
}

func ReorderNodes(attr *NodeAttributes) {
	permutation := SortPermutation(attr)
	for k,i := range attr.Indices {
		attr.Indices[k] = uint32(permutation[i])
	}
	ApplyPermutation(attr, permutation)
}
*/

func ComputeNodeAttributes(graph StreetGraph, subgraph *Subgraph) ([]uint32, error) {
	visitor := &NodeAttributes{
		Subgraph:  subgraph,
		Degrees:   make([]uint32, subgraph.Size+1),
		Positions: make([]int32, 2*subgraph.Size),
	}
	
	err := Traverse(graph, visitor)
	if err != nil {
		return nil, err
	}

	// Write node attributes
	err = Output("positions.ftf", visitor.Positions)
	if err != nil {
		return nil, err
	}
	
	// Write node -> first edge table (that's the degree sum)
	var c uint32 = 0
	h := alg.NewHistogram("degrees")
	e := visitor.Degrees
	for i, d := range e {
		// The last "vertex" is a sentinel
		if uint32(i) < subgraph.Size {
			h.Add(fmt.Sprintf("%d", d))
		}
		e[i] = c
		c += d
	}
	
	// Print statistics
	h.Print()
	fmt.Printf("Edge Count: %d\n", c)
	
	err = Output("vertices.ftf", e)
	if err != nil {
		return nil, err
	}
	
	return e, nil
}

// Pass 3
// Gather edge attributes. The only vexing point here are the step positions.
// Since we do not have edge indices (edges are subdivisions of ways), we can't
// count the "step sizes" first and then allocate a single large file for the
// steps. Instead we need to keep an array of dynamic arrays for the steps.
// This sucks for multiple reasons. One, it uses more memory than it should.
// Two, the garbage collector will have to traverse this very large array of
// pointers.
// TODO: Find a better way.

type EncodedPoint struct {
	Lat, Lng int32
}

type EdgeAttributes struct {
	*Subgraph
	
	// ellipsoid, for distance calculations
	E ellipsoid.Ellipsoid
	// node locations, for the steps
	Positions map[int64] EncodedPoint
	
	// vertex -> edge index maps
	Current []uint32
	// edge -> vertex index map
	Edges []uint32
	// edge -> edge index map
	//Reverse []uint32
	// edge -> distance
	Distance []uint16
	// edge -> steps (could save indices instead of float64 pairs)
	Steps [][]byte
}

func edgeLength(steps []geo.Coordinate, e ellipsoid.Ellipsoid) uint16 {
	if len(steps) < 2 {
		panic(fmt.Sprintf("Missing steps: %v", steps))
	}

	prev := steps[0]
	total := 0.0
	for _, step := range steps[1:] {
		distance, _ := e.To(prev.Lat, prev.Lng, step.Lat, step.Lng)
		total += distance
		prev = step
	}
	return uint16(umath.Float64ToHalf(total))
}

func (v *EdgeAttributes) Position(nodeIndex int64) geo.Coordinate {
	p := v.Positions[nodeIndex]
	return geo.DecodeCoordinate(p.Lat, p.Lng)
}

func (v *EdgeAttributes) VisitNode(node osm.Node) {
	if v.Visited.Get(node.Id) {
		lat, lng := node.Position.Encode()
		v.Positions[node.Id] = EncodedPoint{lat, lng}
	}
}

func (v *EdgeAttributes) VisitRelation(relation osm.Relation) {
}

func (v *EdgeAttributes) VisitWay(way osm.Way) {
	isOneway := way.Attributes["oneway"] == "true"
	segmentStart := 0
	segmentIndex := v.Indices[way.Nodes[0]]
	for i, nodeId := range way.Nodes {
		if i == 0 {
			continue
		}
		if nodeIndex, ok := v.Indices[nodeId]; ok {
			// Record a new edge from vertex segmentIndex to nodeIndex
			edge := v.Current[segmentIndex]
			v.Edges[edge] = nodeIndex
			v.Current[segmentIndex]++

			// If this is a bidirectional road, also record the reverse edge
			rev_edge := edge
			if !isOneway {
				rev_edge = v.Current[nodeIndex]
				v.Edges[rev_edge] = segmentIndex
				v.Current[nodeIndex]++
				//v.Reverse[edge] = rev_edge
				//v.Reverse[rev_edge] = edge
			} else {
				//v.Reverse[edge] = edge
			}

			// Calculate all steps on the way
			edgeSteps := make([]geo.Coordinate, i-segmentStart+1)
			for j, stepId := range way.Nodes[segmentStart:i+1] {
				edgeSteps[j] = v.Position(stepId)
			}

			// Calculate the length of the current edge
			dist := edgeLength(edgeSteps, v.E)
			v.Distance[edge] = dist
			if !isOneway {
				v.Distance[rev_edge] = dist
			}

			// Finally, record the intermediate steps
			if len(edgeSteps) > 2 {
				v.Steps[edge] = geo.EncodeStep(edgeSteps[0], edgeSteps[1 : len(edgeSteps)-1])
				//v.Steps[edge] = edgeSteps[1 : len(edgeSteps)-1]
			} else {
				v.Steps[edge] = nil
			}

			if !isOneway {
				// This is always implicit and we do not save it
				v.Steps[rev_edge] = nil
			}
			
			segmentStart = i
			segmentIndex = nodeIndex
		}
	}
}

func ComputeEdgeAttributes(graph StreetGraph, subgraph *Subgraph, vertices []uint32) error {
	// Allocate space for the edge attributes
	numEdges := vertices[len(vertices)-1]
	attributes := &EdgeAttributes{
		Subgraph:  subgraph,
		Positions: map[int64]EncodedPoint{},
		Current:   vertices,
		Edges:     make([]uint32, numEdges),
		//Reverse:   make([]uint32, numEdges),
		Distance:  make([]uint16, numEdges),
		Steps:     make([][]byte, numEdges),
	}
	
	// We need to compute some distances in this pass
	attributes.E = ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter,
		ellipsoid.Longitude_is_symmetric, ellipsoid.Bearing_is_symmetric)

	// Perform the actual graph traversal
	err := Traverse(graph, attributes)
	if err != nil {
		return err
	}

	// Write all edge attributes to disk
	Output("edges.ftf",     attributes.Edges)
	//Output("rev_edges.ftf", attributes.Reverse)
	Output("distances.ftf", attributes.Distance)

	// Index the step arrays
	stepIndices := make([]uint32, numEdges+1)
	var current uint32 = 0
	for i, steps := range attributes.Steps {
		stepIndices[i] = current
		current += uint32(len(steps))
	}
	stepIndices[numEdges] = current // <- sentinel

	Output("steps.ftf", stepIndices)

	output, err := os.Create("step_positions.ftf")
	if err != nil {
		return err
	}
	for _, steps := range attributes.Steps {
		binary.Write(output, binary.LittleEndian, steps)
	}
	output.Close()

	return nil
}

func main() {
	inputFile  := flag.String("i", "", "input OSM PBF file")
	accessType := flag.String("f", "car", "access type (car, bike, foot)")
	flag.Parse()
	
	if len(*inputFile) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	file, err := os.Open(*inputFile)
	if err != nil {
		println("Unable to open file:", err.Error())
		os.Exit(1)
	}
	
	var access osm.AccessType
	switch *accessType {
	case "car":
		access = osm.AccessMotorcar
	case "bike":
		access = osm.AccessBicycle
	case "foot":
		access = osm.AccessFoot
	default:
		println("Unrecognized access type:", access)
		os.Exit(2)
	}
	graph := StreetGraph{file, access}

	println("Pass 1")

	subgraph, err := InducedSubgraph(graph)
	if err != nil {
		println("Error during pass1:", err.Error())
		os.Exit(3)
	}

	println("Pass 2")

	//indices := NodeIndices {}
	//for k,v := range subgraph.Indices {
	//	indices[k] = v
	//}
	vertices, err := ComputeNodeAttributes(graph, subgraph)
	//same := 0
	//for k,v := range subgraph.Indices {
	///	if indices[k] == v {
	//		same++
	//	}
	//}
	//fmt.Printf("same: %v, len: %v\n", same, len(indices) / 2)
	if err != nil {
		println("Error during pass2:", err.Error())
		os.Exit(4)
	}

	println("Pass 3")

	err = ComputeEdgeAttributes(graph, subgraph, vertices)
	if err != nil {
		println("Error during pass3:", err.Error())
		os.Exit(5)
	}
}

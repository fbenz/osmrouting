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
	"encoding/binary"
	"ellipsoid"
	"flag"
	"fmt"
	"os"
	"parser/pbf"
)

// Debugging

type NodeInspector struct{}

func (*NodeInspector) VisitNode(node pbf.Node) {
	fmt.Printf("Node %d:\n", node.Id)
	fmt.Printf(" - Lat/Lon: (%.5f, %.5f)\n", node.Lat, node.Lon)
	for key, val := range node.Attributes {
		fmt.Printf(" - %s: %s\n", key, val)
	}
}

func (*NodeInspector) VisitWay(way pbf.Way) {
	fmt.Printf("Way %d:\n", way.Id)
	for i, ref := range way.Nodes {
		fmt.Printf(" - Ref[%d] = %d\n", i, ref)
	}
	for key, val := range way.Attributes {
		fmt.Printf(" - %s: %s\n", key, val)
	}
}

type NodeCounter struct {
	nodeCount uint64
	wayCount  uint64
}

func (c *NodeCounter) VisitNode(node pbf.Node) {
	c.nodeCount++
}

func (c *NodeCounter) VisitWay(way pbf.Way) {
	c.wayCount++
}

// PASS 1
// We want to find all nodes which are relevant for the street-graph.
// In particular we need to find all nodes which are either endpoints of
// highways/junctions *or* intersection points of different highways/junctions.
// The latter part makes this expensive. Basically, we need to flag all interior
// nodes we see and add every node we see more than once to the subgraph.
// Corner points are always part of the street graph.

type subgraphNodes map[int64]uint32

type subgraph struct {
	indices subgraphNodes
	visited map[int64] bool
	high    uint32
}

func (s *subgraph) VisitNode(node pbf.Node) {
}

func (s *subgraph) VisitWay(way pbf.Way) {
	if len(way.Nodes) > 1 {
		i := way.Nodes[0]
		j := way.Nodes[len(way.Nodes)-1]
		if _, ok := s.indices[i]; !ok {
			s.indices[i] = s.high
			s.high++
		}
		if _, ok := s.indices[j]; !ok {
			s.indices[j] = s.high
			s.high++
		}
	}
	
	// Flag interior nodes
	if len(way.Nodes) > 2 {
		for _, nodeIndex := range way.Nodes[1:len(way.Nodes)-1] {
			if s.visited[nodeIndex] {
				// Intersection node, add it to the graph
				// (unless it's already in the graph)
				if _, ok := s.indices[nodeIndex]; !ok {
					s.indices[nodeIndex] = s.high
					s.high++
				}
			} else {
				// Flag the node
				s.visited[nodeIndex] = true
			}
		}
	}
}

func subgraphInduction(graph pbf.Graph) (*subgraph, error) {
	var nodes subgraphNodes = subgraphNodes(map[int64]uint32{})
	var visited map[int64]bool = map[int64]bool {}
	var subgraph *subgraph = &subgraph{nodes, visited, 0}
	err := graph.Traverse(subgraph)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Found a street-graph with %d nodes.\n", subgraph.high)
	return subgraph, nil
}

// Pass 2

type nodeAttributes struct {
	graph     *subgraph
	degrees   []uint32
	positions []float64
	reverseIndex map[int] int64
	count int
}

func (v *nodeAttributes) VisitNode(node pbf.Node) {
	if i, ok := v.graph.indices[node.Id]; ok {
		v.positions[2*i]   = node.Lat
		v.positions[2*i+1] = node.Lon
		//fmt.Printf("Visit: %d\n", i)
		v.reverseIndex[int(i)] = node.Id
		v.count++
	}
}

func (v *nodeAttributes) VisitWay(way pbf.Way) {
	isOneway := way.Attributes["oneway"] == "true"
	//segmentStart := 0
	segmentIndex, ok := v.graph.indices[way.Nodes[0]]
	if !ok {
		panic("First vertex of a path is not in the graph!?")
	}
	borked := true
	for _, nodeId := range way.Nodes[1:] {
		if nodeIndex, ok := v.graph.indices[nodeId]; ok {
			borked = false
			v.degrees[segmentIndex]++
			if !isOneway {
				v.degrees[nodeIndex]++
			}
			segmentIndex = nodeIndex
			//segmentStart = i
		}
	}
	if borked {
		fmt.Printf("Visited an edge without vertices: %v.\n", way)
	}
}

func nodeAttribution(graph pbf.Graph, subgraph *subgraph) ([]uint32, []float64, error) {
	positions := make([]float64, 2*subgraph.high)
	degrees := make([]uint32, subgraph.high+1)
	for i, _ := range degrees {
		degrees[i] = 0
	}
	filter := &nodeAttributes{subgraph, degrees, positions, map[int]int64 {}, 0}
	err := graph.Traverse(filter)
	if err != nil {
		return nil, nil, err
	}

	println("Writing node positions")

	// Write node attributes
	output, err := os.Create("positions.ftf")
	if err != nil {
		return nil, nil, err
	}
	binary.Write(output, binary.LittleEndian, positions)
	output.Close()

	println("Writing node edge pointers")

	// Write node -> edge start pointers
	vertices, err := os.Create("vertices.ftf")
	if err != nil {
		return nil, nil, err
	}
	var current uint32 = 0
	var minDegree uint32 = degrees[0]
	var maxDegree uint32 = degrees[0]
	histogram := map[uint32] int {}
	missing := 0
	zeros   := 0
	for i, d := range degrees {
		// The last "vertex" is a sentinel and should not appear in the
		// statistics...
		if uint32(i) < subgraph.high {
			if _, ok := filter.reverseIndex[i]; !ok {
				missing++
			} else if d == 0 {
				fmt.Printf("Degree 0 node: %d\n", filter.reverseIndex[i])
				zeros++
			}
			if _, ok := histogram[d]; !ok {
				histogram[d] = 1
			} else {
				histogram[d]++
			}
			if i < int(subgraph.high) {
				if d < minDegree {
					minDegree = d
				}
				if d > maxDegree {
					maxDegree = d
				}
			}
		}
		degrees[i] = current
		current += d
	}
	fmt.Printf("Visited %d nodes\n", filter.count)
	fmt.Printf("Missing nodes: %d\n", missing)
	fmt.Printf("Degress 0 nodes: %d\n", zeros)
	fmt.Printf("Edge count: %d\n", current)
	fmt.Printf("Node count: %d\n", subgraph.high)
	fmt.Printf("Average degree: %.4f\n", float64(current)/float64(subgraph.high))
	fmt.Printf("Minimum degree: %d\n", minDegree)
	fmt.Printf("Maximum degree: %d\n", maxDegree)
	fmt.Printf("Degree histogram: %d\n", histogram)
	//fmt.Printf("Degrees: %v\n", degrees)
	binary.Write(vertices, binary.LittleEndian, degrees)
	vertices.Close()
	println("Success.")
	return degrees, positions, nil
}

// Pass 3

type step struct {
	lat float64
	lon float64
}

type edgeAttributes struct {
	// ellipsoid for distance calculations
	geo ellipsoid.Ellipsoid
	nodePositions []float64
	// focus on the street graph
	graph *subgraph
	// node locations
	locations map[int64]step
	// vertex -> edge index maps
	current []uint32
	// edge -> vertex index map
	edges []uint32
	// edge -> edge index map
	reverse []uint32
	// edge -> distance
	distance []float64
	// edge -> steps
	steps [][]step
}

func edgeLength(steps []step, geo ellipsoid.Ellipsoid) float64 {
	if len(steps) < 2 {
		fmt.Printf("%v\n", steps)
		panic("Missing steps")
		return 0.0
	}

	prev := steps[0]
	total := 0.0
	for _, step := range steps[1:] {
		distance, _ := geo.To(prev.lat, prev.lon, step.lat, step.lon)
		total += distance
		prev = step
	}
	return total
}

func (v *edgeAttributes) VisitNode(node pbf.Node) {
	v.locations[node.Id] = step{node.Lat, node.Lon}
}

func (v *edgeAttributes) VisitWay(way pbf.Way) {
	isOneway := way.Attributes["oneway"] == "true"
	segmentStart := 0
	segmentIndex := v.graph.indices[way.Nodes[0]]
	for i, nodeId := range way.Nodes {
		if i == 0 {
			continue
		}
		if nodeIndex, ok := v.graph.indices[nodeId]; ok {
			// Record a new edge from vertex segmentIndex to nodeIndex
			edge := v.current[segmentIndex]
			v.edges[edge] = nodeIndex
			v.current[segmentIndex]++

			// If this is a bidirectional road, also record the reverse edge
			rev_edge := edge
			if !isOneway {
				rev_edge = v.current[nodeIndex]
				v.edges[rev_edge] = segmentIndex
				v.current[nodeIndex]++
				v.reverse[edge] = rev_edge
				v.reverse[rev_edge] = edge
			} else {
				v.reverse[edge] = edge
			}

			// Calculate all steps on the way
			edgeSteps := make([]step, i-segmentStart+1)
			for j, stepId := range way.Nodes[segmentStart:i+1] {
				edgeSteps[j] = v.locations[stepId]
			}
			
			nlat := v.nodePositions[2 * segmentIndex]
			nlon := v.nodePositions[2 * segmentIndex + 1]
			vlat := v.locations[way.Nodes[segmentStart]].lat
			vlon := v.locations[way.Nodes[segmentStart]].lon
			if nlat != vlat || nlon != vlon {
				fmt.Printf("Node Positions are wrong:\n")
				fmt.Printf(" - should: (%.2f, %.2f)\n", nlat, nlon)
				fmt.Printf(" - is:     (%.2f, %.2f)\n", vlat, vlon)
				fmt.Printf(" - node id: %d\n", segmentIndex)
				fmt.Printf(" - osm id:  %d\n", way.Nodes[segmentStart])
				panic("No point in continuing.")
			}

			// Calculate the length of the current edge
			dist := edgeLength(edgeSteps, v.geo)
			v.distance[edge] = dist
			if !isOneway {
				v.distance[rev_edge] = dist
			}
			
			// Sanity check
			tlat := v.nodePositions[2 * nodeIndex]
			tlon := v.nodePositions[2 * nodeIndex + 1]
			dlat := v.locations[nodeId].lat
			dlon := v.locations[nodeId].lon
			if tlat != dlat || tlon != dlon {
				fmt.Printf("Node Positions are wrong:\n")
				fmt.Printf(" - should: (%.2f, %.2f)\n", tlat, tlon)
				fmt.Printf(" - is:     (%.2f, %.2f)\n", dlat, dlon)
				fmt.Printf(" - node id: %d\n", nodeIndex)
				fmt.Printf(" - osm id:  %d\n", nodeId)
				panic("No point in continuing.")
			}
			line, _ := v.geo.To(nlat, nlon, dlat, dlon)
			if line > dist + 0.1 {
				fmt.Printf("Wormhole\n")
				fmt.Printf(" - from: (%.2f, %.2f)\n", nlat, nlon)
				fmt.Printf(" - to:   (%.2f, %.2f)\n", dlat, dlon)
				fmt.Printf(" - dist: %.4f\n", dist)
				fmt.Printf(" - line: %.4f\n", line)
				panic("No point in continuing.")
			}

			// Finally, record the intermediate steps
			if len(edgeSteps) > 2 {
				v.steps[edge] = edgeSteps[1 : len(edgeSteps)-1]
			} else {
				v.steps[edge] = []step {}
			}

			if !isOneway {
				// This is always implicit and we do not save it
				v.steps[rev_edge] = []step {}
			}
			
			segmentStart = i
			segmentIndex = nodeIndex
		}
	}
}

func TestDistances(attributes *edgeAttributes, vertices []uint32) {
	nodeCount := len(vertices) - 1
	for i := 0; i < nodeCount; i++ {
		nodeLat := attributes.nodePositions[2 * i]
		nodeLng := attributes.nodePositions[2 * i + 1]
		start := vertices[i]
		stop  := vertices[i + 1]
		for edgeIndex := start; edgeIndex < stop; edgeIndex++ {
			tip := attributes.edges[edgeIndex]
			tipLat := attributes.nodePositions[2 * tip]
			tipLng := attributes.nodePositions[2 * tip + 1]
			dist   := attributes.distance[edgeIndex]
			line, _ := attributes.geo.To(nodeLat, nodeLng, tipLat, tipLng)
			if line > dist + 0.01 {
				fmt.Printf("Wormhole:\n")
				fmt.Printf(" - from: %.4f, %.4f\n", nodeLat, nodeLng)
				fmt.Printf(" - to:   %.4f, %.4f\n", tipLat, tipLng)
				fmt.Printf(" - in distance: %.2f m\n", dist)
				fmt.Printf(" - line dist:   %.2f m\n", line)
				panic("Something is really wrong.")
			}
		}
	}
}

func edgeAttribution(graph pbf.Graph, subgraph *subgraph, vertices []uint32, positions []float64) error {
	// Allocate space for the edge attributes
	numEdges := vertices[len(vertices)-1]
	highPointers := make([]uint32, len(vertices) - 1)
	copy(highPointers, vertices)
	attributes := &edgeAttributes{
		graph:     subgraph,
		nodePositions: positions,
		locations: map[int64]step{},
		current:   highPointers,
		edges:     make([]uint32, numEdges),
		reverse:   make([]uint32, numEdges),
		distance:  make([]float64, numEdges),
		steps:     make([][]step, numEdges),
	}
	
	// We need to compute some distances in this pass
	attributes.geo = ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter,
		ellipsoid.Longitude_is_symmetric, ellipsoid.Bearing_is_symmetric)

	// Perform the actual graph traversal
	graph.Traverse(attributes)
		
	// Check that we hit all the edges
	for i, high := range highPointers {
		if high != vertices[i + 1] {
			fmt.Printf("Missed a vertex at index %d\n", i)
			fmt.Printf("Degree should be: %d\n", vertices[i + 1] - vertices[i])
			fmt.Printf("       is:        %d\n", highPointers[i] - vertices[i])
		}
	}
	
	// Check that the distances make sense
	TestDistances(attributes, vertices)

	// Write all edge attributes to disk
	output, err := os.Create("edges.ftf")
	if err != nil {
		return err
	}
	binary.Write(output, binary.LittleEndian, attributes.edges)
	output.Close()

	output, err = os.Create("rev_edges.ftf")
	if err != nil {
		return err
	}
	binary.Write(output, binary.LittleEndian, attributes.reverse)
	output.Close()

	output, err = os.Create("distances.ftf")
	if err != nil {
		return err
	}
	binary.Write(output, binary.LittleEndian, attributes.distance)
	output.Close()

	// Index the step arrays
	stepIndices := make([]uint32, numEdges+1)
	var current uint32 = 0
	for i, steps := range attributes.steps {
		stepIndices[i] = current
		current += uint32(len(steps))
	}
	stepIndices[numEdges] = current // <- sentinel

	output, err = os.Create("steps.ftf")
	if err != nil {
		return err
	}
	binary.Write(output, binary.LittleEndian, stepIndices)
	output.Close()

	output, err = os.Create("step_positions.ftf")
	if err != nil {
		return err
	}
	for _, steps := range attributes.steps {
		binary.Write(output, binary.LittleEndian, steps)
	}
	output.Close()

	return nil
}

func main() {
	inputFile  := flag.String("i", "input.osm.pbf", "input OSM PBF file")
	accessType := flag.String("f", "car", "access type (car, bike, foot)")
	//outputFile := flag.String("o", "output.map", "output graph map file")
	flag.Parse()

	file, err := os.Open(*inputFile)
	if err != nil {
		println("Unable to open file:", err.Error())
		os.Exit(1)
	}
	
	var access pbf.AccessType
	switch *accessType {
	case "car":
		access = pbf.AccessMotorcar
	case "bike":
		access = pbf.AccessBicycle
	case "foot":
		access = pbf.AccessFoot
	default:
		println("Unrecognized access type:", access)
		os.Exit(2)
	}
	graph := pbf.NewGraph(file, access)

	println("Pass 1")

	subgraph, err := subgraphInduction(graph)
	if err != nil {
		println("Error during pass1:", err.Error())
		os.Exit(3)
	}

	println("Pass 2")

	vertices, positions, err := nodeAttribution(graph, subgraph)
	if err != nil {
		println("Error during pass2:", err.Error())
		os.Exit(4)
	}

	println("Pass 3")

	err = edgeAttribution(graph, subgraph, vertices, positions)
	if err != nil {
		println("Error during pass3:", err.Error())
		os.Exit(5)
	}
}

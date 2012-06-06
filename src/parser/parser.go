package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"parser/pbf"
	"math"
)

func traverseGraph(file *os.File, visitor pbf.Visitor) error {
	_, err := file.Seek(0, 0)
	if err != nil {
		return err
	}

	pbf.VisitRoutes(file, visitor)
	return nil
}

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

type subgraphNodes map[int64]uint32

type subgraph struct {
	indices subgraphNodes
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
}

func subgraphInduction(file *os.File) (*subgraph, error) {
	var nodes subgraphNodes = subgraphNodes(map[int64]uint32{})
	var graph *subgraph = &subgraph{nodes, 0}
	err := traverseGraph(file, graph)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Found a street-graph with %d nodes.\n", graph.high)
	return graph, nil
}

// Pass 2

type nodeAttributes struct {
	graph     *subgraph
	degrees   []uint32
	positions []float64
}

func (v *nodeAttributes) VisitNode(node pbf.Node) {
	if i, ok := v.graph.indices[node.Id]; ok {
		v.positions[2*i] = node.Lat
		v.positions[2*i+1] = node.Lon
	}
}

func (v *nodeAttributes) VisitWay(way pbf.Way) {
	isOneway := way.Attributes["oneway"] == "true"
	//segmentStart := 0
	segmentIndex := v.graph.indices[way.Nodes[0]]
	for _, nodeId := range way.Nodes[1:] {
		if nodeIndex, ok := v.graph.indices[nodeId]; ok {
			v.degrees[segmentIndex]++
			if !isOneway {
				v.degrees[nodeIndex]++
			}
			segmentIndex = nodeIndex
			//segmentStart = i
		}
	}
}

func nodeAttribution(file *os.File, graph *subgraph) ([]uint32, error) {
	positions := make([]float64, 2*graph.high)
	degrees := make([]uint32, graph.high+1)
	for i, _ := range degrees {
		degrees[i] = 0
	}
	filter := &nodeAttributes{graph, degrees, positions}
	err := traverseGraph(file, filter)
	if err != nil {
		return nil, err
	}

	println("Writing node positions")

	// Write node attributes
	output, err := os.Create("positions.ftf")
	if err != nil {
		return nil, err
	}
	binary.Write(output, binary.LittleEndian, positions)
	output.Close()

	println("Writing node edge pointers")

	// Write node -> edge start pointers
	vertices, err := os.Create("vertices.ftf")
	if err != nil {
		return nil, err
	}
	var current uint32 = 0
	var minDegree uint32 = degrees[0]
	var maxDegree uint32 = degrees[0]
	for i, d := range degrees {
		if d == 0 {
			fmt.Printf("Degree 0 vertex: %d\n", i)
		}
		degrees[i] = current
		current += d
		if i < int(graph.high) {
			if d < minDegree {
				minDegree = d
			}
			if d > maxDegree {
				maxDegree = d
			}
		}
	}
	fmt.Printf("Edge count: %d\n", current)
	fmt.Printf("Node count: %d\n", graph.high)
	fmt.Printf("Average degree: %.4f\n", float64(current)/float64(graph.high))
	fmt.Printf("Minimum degree: %d\n", minDegree)
	fmt.Printf("Maximum degree: %d\n", maxDegree)
	//fmt.Printf("Degrees: %v\n", degrees)
	binary.Write(vertices, binary.LittleEndian, degrees)
	vertices.Close()
	println("Success.")
	return degrees, nil
}

// Pass 3

type step struct {
	lat float64
	lon float64
}

type edgeAttributes struct {
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

func distance(from, to step) float64 {
	// Great circle distance - probably overkill,
	// a euclidean approximation would do...
	fromLat := from.lat * math.Pi / 180.0
	fromLng := from.lon * math.Pi / 180.0
	toLat   := to.lat   * math.Pi / 180.0
	toLng   := to.lon   * math.Pi / 180.0
	deltaLat1 := math.Sin((toLat - fromLat) / 2.0)
	deltaLng1 := math.Sin((toLng - fromLng) / 2.0)
	deltaLat2 := deltaLat1 * deltaLat1
	deltaLng2 := deltaLng1 * deltaLng1
	delta := math.Sqrt(deltaLat2 + math.Cos(fromLat) * math.Cos(fromLng) * deltaLng2)
	return 6378.388 * 2 * math.Asin(delta)
}

func edgeLength(steps []step) float64 {
	if len(steps) < 2 {
		return 0.0
	}

	prev := steps[0]
	total := 0.0
	for _, step := range steps[1:] {
		total += distance(prev, step)
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
	for i, nodeId := range way.Nodes[1:] {
		if nodeIndex, ok := v.graph.indices[nodeId]; ok {
			// Record a new edge from vertex segmentStart to nodeIndex
			edge := v.current[segmentStart]
			v.edges[edge] = nodeIndex
			v.current[segmentStart]++

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
			for j, stepId := range way.Nodes[segmentStart:i] {
				edgeSteps[j] = v.locations[stepId]
			}

			// Calculate the length of the current edge
			dist := edgeLength(edgeSteps)
			v.distance[edge] = dist
			if !isOneway {
				v.distance[rev_edge] = dist
			}

			// Finally, record the intermediate steps
			if len(edgeSteps) > 2 {
				v.steps[edge] = edgeSteps[1 : len(edgeSteps)-1]
			} else {
				v.steps[edge] = make([]step, 0)
			}

			if !isOneway {
				// This is always implicit and we do not save it
				v.steps[rev_edge] = nil
			}
		}
	}
}

func edgeAttribution(file *os.File, graph *subgraph, vertices []uint32) error {
	// Allocate space for the edge attributes
	numEdges := vertices[len(vertices)-1]
	attributes := &edgeAttributes{
		graph:     graph,
		locations: map[int64]step{},
		current:   vertices,
		edges:     make([]uint32, numEdges),
		reverse:   make([]uint32, numEdges),
		distance:  make([]float64, numEdges),
		steps:     make([][]step, numEdges),
	}

	// Perform the actual graph traversal
	traverseGraph(file, attributes)

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
	inputFile := flag.String("i", "input.osm.pbf", "input OSM PBF file")
	//outputFile := flag.String("o", "output.map", "output graph map file")
	flag.Parse()

	file, err := os.Open(*inputFile)
	if err != nil {
		println("Unable to open file:", err.Error())
		os.Exit(1)
	}

	println("Pass 1")

	graph, err := subgraphInduction(file)
	if err != nil {
		println("Error during pass1:", err.Error())
		os.Exit(2)
	}

	println("Pass 2")

	vertices, err := nodeAttribution(file, graph)
	if err != nil {
		println("Error during pass2:", err.Error())
		os.Exit(3)
	}

	println("Pass 3")

	err = edgeAttribution(file, graph, vertices)
	if err != nil {
		println("Error during pass3:", err.Error())
		os.Exit(4)
	}
}

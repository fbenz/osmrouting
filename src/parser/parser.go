package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"parser/pbf"
)

type NodeInspector struct {}

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

type SubgraphNodes map[int64] bool

func (s SubgraphNodes) VisitNode(node pbf.Node) {
}

func (s SubgraphNodes) VisitWay(way pbf.Way) {
	if len(way.Nodes) > 0 {
		s[way.Nodes[0]] = true
		s[way.Nodes[len(way.Nodes) - 1]] = true
	}
}

type SubgraphFilter struct {
	nodes  SubgraphNodes
	client pbf.Visitor
}

func (s *SubgraphFilter) VisitNode(node pbf.Node) {
	if s.nodes[node.Id] {
		s.client.VisitNode(node)
	}
}

func (s *SubgraphFilter) VisitWay(way pbf.Way) {
	s.client.VisitWay(way)
}

func visitSubgraph(file *os.File, client pbf.Visitor) {
	var focus SubgraphNodes = SubgraphNodes(map[int64] bool {})
	pbf.VisitRoutes(file, focus)
	file.Seek(0, 0)
	
	var filter *SubgraphFilter = &SubgraphFilter{focus, client}
	pbf.VisitRoutes(file, filter)
}

// Implementation

func traverseGraph(file *os.File, visitor pbf.Visitor) error {
	_, err := file.Seek(0, 0)
	if err != nil {
		return err
	}
	
	pbf.VisitRoutes(file, visitor)
	return nil
}

// PASS 1

type subgraphNodes map[int64] uint64

type subgraph struct {
	indices subgraphNodes
	high uint64
}

func (s *subgraph) VisitNode(node pbf.Node) {
}

func (s *subgraph) VisitWay(way pbf.Way) {
	if len(way.Nodes) > 1 {
		i := way.Nodes[0]
		j := way.Nodes[len(way.Nodes) - 1]
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
	var nodes subgraphNodes = subgraphNodes(map[int64] uint64 {})
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
	graph *subgraph
	degrees   []uint32
	positions []float64
}

func (v *nodeAttributes) VisitNode(node pbf.Node) {
	if i, ok := v.graph.indices[node.Id]; ok {
		v.positions[2 * i]     = node.Lat
		v.positions[2 * i + 1] = node.Lon
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

func nodeAttribution(file *os.File, graph *subgraph) error {
	positions := make([]float64, 2 * graph.high)
	degrees   := make([]uint32,  graph.high + 1)
	for i, _ := range degrees {
		degrees[i] = 0
	}
	filter := &nodeAttributes{graph, degrees, positions}
	err := traverseGraph(file, filter)
	if err != nil {
		return err
	}
	
	println("Writing node positions")
	
	// Write node attributes
	output, err := os.Create("positions.ftf")
	if err != nil {
		return err
	}
	binary.Write(output, binary.LittleEndian, positions)
	output.Close()
	
	println("Writing node edge pointers")
	
	// Write node -> edge start pointers
	vertices, err := os.Create("vertices.ftf")
	if err != nil {
		return err
	}
	var current   uint32 = 0
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
	fmt.Printf("Average degree: %.4f\n", float64(current) / float64(graph.high))
	fmt.Printf("Minimum degree: %d\n", minDegree)
	fmt.Printf("Maximum degree: %d\n", maxDegree)
	//fmt.Printf("Degrees: %v\n", degrees)
	binary.Write(vertices, binary.LittleEndian, degrees)
	vertices.Close()
	println("Success.")
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
	
	err = nodeAttribution(file, graph)
	if err != nil {
		println("Error during pass2:", err.Error())
		os.Exit(3)
	}

/*
	blockCount := 0
	for {
		block, err := pbf.ReadBlock(file)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		
		if block.Kind == pbf.OSMHeader {
			print("Header Block")
		} else {
			print("Data Block")
		}
		fmt.Printf(" - Size: %d\n", len(block.Data))
		blockCount++
	}
	fmt.Printf("# of blocks: %d\n", blockCount)
*/
	//visitor := &NodeInspector{}
	//visitor := &NodeCounter{0,0}
	//err = pbf.VisitGraph(file, visitor)
	//err = pbf.VisitRoutes(file, visitor)
	//visitSubgraph(file, visitor)
	//fmt.Printf("Visited %d nodes and %d ways\n", visitor.nodeCount, visitor.wayCount)
	//if err != nil {
	//	panic(err)
	//}
}

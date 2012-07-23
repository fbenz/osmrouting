
package main

import (
	"alg"
	"log"
	"os"
	"osm"
)

// Compared to the visited or positions array the vertices are very sparse and
// it makes no sense to use a veb tree for this.
type NodeIndices map[int64] uint32

// Street graph visitor, skips unimportant ways.
type StreetGraph struct {
	File    *os.File
	Access   osm.AccessType
	Size     uint32
	Indices  NodeIndices
	Visited  alg.BitVector
}

func (s *StreetGraph) VisitNode(node osm.Node) {
}

func (s *StreetGraph) VisitRelation(relation osm.Relation) {
}

func (s *StreetGraph) VisitWay(way osm.Way) {
	// Skip trivial ways
	if len(way.Nodes) < 2 {
		return
	}
	
	// Skip non-roads
	mask := osm.AccessMask(way)
	if mask & s.Access == 0 {
		return
	}
	safe := osm.NormalizeOneway(way)
	if !safe && mask == osm.AccessMotorcar {
		return
	}
	
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

func NewStreetGraph(file *os.File, access osm.AccessType) *StreetGraph {
	graph := &StreetGraph{
		File:    file,
		Access:  access,
		Indices: NodeIndices {},
		Visited: alg.NewBitVector(64),
		Size:    0,
	}
	err := osm.ParseFile(file, graph)
	if err != nil {
		log.Fatal(err.Error())
	}
	return graph
}

type StreetGraphVisitor struct {
	Access  osm.AccessType
	Visitor osm.Visitor
	Nodes   alg.BitVector
}

func (s *StreetGraphVisitor) VisitNode(node osm.Node) {
	if s.Nodes.Get(node.Id) {
		s.Visitor.VisitNode(node)
	}
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
	// this street for car routing.
	safe := osm.NormalizeOneway(way)
	if !safe {
		if mask == osm.AccessMotorcar {
			return
		} else if mask & osm.AccessMotorcar != 0 {
			way.Attributes["motorcar"] = "no"
		}
	}
	
	s.Visitor.VisitWay(way)
}

func (s *StreetGraph) Visit(visitor osm.Visitor) {
	filter := &StreetGraphVisitor{
		Access:  s.Access,
		Visitor: visitor,
		Nodes:   s.Visited,
	}
	err := osm.ParseFile(s.File, filter)
	if err != nil {
		log.Fatal(err.Error())
	}
}

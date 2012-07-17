
package main

import (
	"alg"
	"fmt"
	"mm"
	"osm"
)

// Pass 2
// Gather the relevant node attributes: out-degrees and positions.
// At this point we could allow a small interlude which sorts the node
// indices, but we do not do this currently.

type NodeAttributes struct {
	*StreetGraph
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
	prevIndex, _ := v.Indices[way.Nodes[0]]
	
	for _, osmId := range way.Nodes[1:] {
		nodeIndex, ok := v.Indices[osmId]
		if !ok {
			continue
		}
		
		// We always have an edge from the previous index to this one
		v.Degrees[prevIndex]++
		// The reverse edge only exists if this is a two-way road
		if !isOneway {
			v.Degrees[nodeIndex]++
		}
		
		prevIndex = nodeIndex
	}
}

func NewNodeAttributes(graph *StreetGraph) (*NodeAttributes, error) {
	attr := &NodeAttributes{StreetGraph: graph}
	
	err  := mm.Create("vertices.ftf", int(graph.Size+1), &attr.Degrees)
	if err != nil {
		return nil, err
	}
	
	err = mm.Create("positions.ftf", int(2*graph.Size), &attr.Positions)
	if err != nil {
		return nil, err
	}
	
	return attr, nil
}

func PrintStatistics(attr *NodeAttributes) {
	h   := alg.NewHistogram("degrees")
	m   := uint32(0)
	min := attr.Degrees[0]
	max := min
	
	for _, d := range attr.Degrees[:len(attr.Degrees)-1] {
		h.Add(fmt.Sprintf("%d", d))
		m += d
		if d < min {
			min = d
		} else if d > max {
			max = d
		}
	}
	
	// Print statistics
	h.Print()
	fmt.Printf("\n")
	fmt.Printf("Street Graph:\n")
	fmt.Printf(" - |V| = %v\n", attr.Size)
	fmt.Printf(" - |E| = %v\n", m)
	fmt.Printf(" - average degree: %.2f\n", float64(m) / float64(attr.Size))
	fmt.Printf(" - minimum degree: %v\n", min)
	fmt.Printf(" - maximum degree: %v\n", max)
}

func WriteNodeAttributes(attr *NodeAttributes) ([]uint32, error) {
	err := mm.Close(&attr.Positions)
	if err != nil {
		return nil, err
	}
	
	// Write node -> first edge table (that's the degree sum)
	c := uint32(0)
	for i, d := range attr.Degrees {
		attr.Degrees[i] = c
		c += d
	}
	
	// We need to preserve the vertices for the third pass, but we really
	// shouldn't keep the file mapping around. Instead we copy everything
	// to the go heap and then close the mapping.
	vertices := make([]uint32, attr.Size+1)
	copy(vertices, attr.Degrees)
	err = mm.Close(&attr.Degrees)
	if err != nil {
		return nil, err
	}
	
	return vertices, nil
}

func ComputeNodeAttributes(graph *StreetGraph) ([]uint32, error) {
	attr, err := NewNodeAttributes(graph)
	if err != nil {
		return nil, err
	}

	err = graph.Visit(attr)
	if err != nil {
		return nil, err
	}

	PrintStatistics(attr)

	return WriteNodeAttributes(attr)
}

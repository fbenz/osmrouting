
package main

import (
	"alg"
	"fmt"
	"ellipsoid"
	"geo"
	"mm"
	"osm"
)

// Pass 3
// Gather edge attributes.

type EdgeAttributes struct {
	*StreetGraph
	
	// ellipsoid, for distance calculations
	E ellipsoid.Ellipsoid
	// node locations, for the steps
	Positions Positions
	
	// Allocator for the step arrays
	Region  *mm.Region
	
	// vertex -> edge index maps
	Current   []uint32
	// edge -> vertex index map
	Edges     []uint32
	// edge -> distance
	Distances []uint16
	// edge -> step indices
	Steps     [][]byte
}

func EdgeLength(steps []geo.Coordinate, e ellipsoid.Ellipsoid) uint16 {
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
	return alg.Float64ToHalf(total)
}

func (v *EdgeAttributes) VisitNode(node osm.Node) {
	v.Positions.Set(node.Id, node.Position)
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
			}

			// Calculate all steps on the way
			edgeSteps := make([]geo.Coordinate, i-segmentStart+1)
			for j, stepId := range way.Nodes[segmentStart:i+1] {
				edgeSteps[j] = v.Positions.Get(stepId)
			}

			// Calculate the length of the current edge
			dist := EdgeLength(edgeSteps, v.E)
			v.Distances[edge] = dist
			if !isOneway {
				v.Distances[rev_edge] = dist
			}

			// Finally, record the intermediate steps
			if len(edgeSteps) > 2 {
				step := geo.EncodeStep(edgeSteps[0], edgeSteps[1 : len(edgeSteps)-1])
				v.Region.Allocate(len(step), &v.Steps[edge])
				copy(v.Steps[edge], step)
			}

			segmentStart = i
			segmentIndex = nodeIndex
		}
	}
}

func NewEdgeAttributes(graph *StreetGraph, vertices []uint32) (*EdgeAttributes, error) {
	numEdges := int(vertices[len(vertices)-1])
	attr := &EdgeAttributes{
		StreetGraph: graph,
		Positions:   NewPositions(64),
		Current:     vertices,
		Region:      mm.NewRegion(0),
	}
	
	err := mm.Create("edges.ftf", numEdges, &attr.Edges)
	if err != nil {
		return nil, err
	}
	err = mm.Create("distances.ftf", numEdges, &attr.Distances)
	if err != nil {
		return nil, err
	}
	err = mm.Allocate(numEdges+1, &attr.Steps)
	if err != nil {
		return nil, err
	}
	
	attr.E = ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter,
		ellipsoid.Longitude_is_symmetric, ellipsoid.Bearing_is_symmetric)
	
	return attr, nil
}

func WriteEdgeAttributes(attr *EdgeAttributes) error {
	fmt.Printf("Writing edge attributes\n")
	
	err := mm.Close(&attr.Edges)
	if err != nil {
		return err
	}
	
	err = mm.Close(&attr.Distances)
	if err != nil {
		return err
	}
	
	// Compute the step indices
	var steps []uint32
	err = mm.Create("steps.ftf", len(attr.Steps), &steps)
	if err != nil {
		return err
	}
	c := uint32(0)
	for i, step := range attr.Steps {
		steps[i] = c
		c += uint32(len(step))
	}
	
	// Output the compressed steps
	var step_positions []byte
	err = mm.Create("step_positions.ftf", int(c), &step_positions)
	if err != nil {
		return err
	}
	for i, step := range attr.Steps {
		copy(step_positions[steps[i]:], step)
	}
	
	err = mm.Close(&steps)
	if err != nil {
		return err
	}
	return mm.Close(&step_positions)
}

func ComputeEdgeAttributes(graph *StreetGraph, vertices []uint32) error {
	attr, err := NewEdgeAttributes(graph, vertices)
	if err != nil {
		return err
	}

	// Perform the actual graph traversal
	fmt.Printf("Edge Attribute traversal\n")
	err = graph.Visit(attr)
	if err != nil {
		return err
	}

	return WriteEdgeAttributes(attr)
}


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
	CurrentOut []uint32
	FirstIn    []uint32
	// edge -> vertex index map
	Edges      []uint32
	// edge -> edge index map
	NextIn     []uint32
	
	// edge -> distance (float16)
	Distances  []uint16
	MaxSpeeds  []uint16
	// edge -> encoded steps
	Steps      [][]byte
	
	// access bitvectors
	Oneway     []byte // TODO: treat oneway bike/car differently
	AccessCar  []byte
	AccessFoot []byte
	AccessBike []byte
	Ferries    []byte
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
	
	w := alg.Float64ToHalf(total)
	if alg.IsInfHalf(w) {
		//fmt.Printf("Edge length %v overflows half, rounding to %v.\n",
		//	total, alg.MaxHalfFloat)
		w = alg.MaxHalf
	} else if w == 0 {
		//fmt.Printf("Edge length %v underflows half, rounding to %v.\n",
		//	total, alg.MinHalfFloat)
		w = alg.MinHalf
	}
	return w
}

func (v *EdgeAttributes) VisitNode(node osm.Node) {
	v.Positions.Set(node.Id, node.Position)
}

func (v *EdgeAttributes) VisitRelation(relation osm.Relation) {
}

// Record a new edge from vertex i to j
func (v *EdgeAttributes) NewEdge(i, j uint32) uint32 {
	// Out edge i->
	edge := v.CurrentOut[i]
	v.Edges[edge] = i ^ j
	v.CurrentOut[i]++

	// In edge ->j
	if v.FirstIn[j] != 0xffffffff {
		v.NextIn[edge] = v.FirstIn[j]
	} else {
		v.NextIn[edge] = edge
	}
	v.FirstIn[j] = edge

	return edge
}

func (v *EdgeAttributes) NewStep(nodes []int64, edge uint32) {
	// Calculate all steps on the way
	step := make([]geo.Coordinate, len(nodes))
	for j, id := range nodes {
		step[j] = v.Positions.Get(id)
	}

	// Calculate the length of the current edge
	v.Distances[edge] = EdgeLength(step, v.E)

	// Record the intermediate steps (if any)
	if len(step) > 2 {
		encode := geo.EncodeStep(step[0], step[1:])
		v.Region.Allocate(len(encode), &v.Steps[edge])
		copy(v.Steps[edge], encode)
	}
}

func SetBit(ary []byte, i uint32) {
	ary[i / 8] |= 1 << (i % 8)
}

func (v *EdgeAttributes) SetExtendedAttributes(way osm.Way, edge uint32) {
	// Osm Attributes
	// Store MaxSpeed in m/s instead of km/h, since Distances are in meters.
	speed := osm.MaxSpeed(way) * 0.277778
	if speed == 0 {
		speed = 1 // Shouldn't happen, but let's be on the safe side.
	}
	v.MaxSpeeds[edge] = alg.Float64ToHalf(speed)
	
	// Bitvectors
	if way.Attributes["oneway"] == "true" {
		SetBit(v.Oneway, edge)
	}
	
	mask := osm.AccessMask(way)
	if mask & osm.AccessMotorcar != 0 {
		SetBit(v.AccessCar, edge)
	}
	if mask & osm.AccessBicycle != 0 {
		SetBit(v.AccessBike, edge)
	}
	if mask & osm.AccessFoot != 0 {
		SetBit(v.AccessFoot, edge)
	}
	
	if way.Attributes["route"] == "ferry" {
		SetBit(v.Ferries, edge)
	}
}

func (v *EdgeAttributes) VisitWay(way osm.Way) {
	//isOneway := way.Attributes["oneway"] == "true"
	segmentStart := 0
	segmentIndex := v.Indices[way.Nodes[0]]
	for i, nodeId := range way.Nodes {
		if i == 0 {
			continue
		}
		if nodeIndex, ok := v.Indices[nodeId]; ok {
			// avoid self loops
			if segmentIndex == nodeIndex {
				continue
			}
			edge := v.NewEdge(segmentIndex, nodeIndex)
			v.NewStep(way.Nodes[segmentStart:i+1], edge)
			v.SetExtendedAttributes(way, edge)
			segmentStart = i
			segmentIndex = nodeIndex
		}
	}
}

func NewEdgeAttributes(graph *StreetGraph, vertices []uint32) *EdgeAttributes {
	numVertices := len(vertices) - 1
	numEdges := int(vertices[numVertices])
	attr := &EdgeAttributes{
		StreetGraph: graph,
		Positions:   NewPositions(64),
		CurrentOut:  vertices,
		Region:      mm.NewRegion(0),
	}
	
	Create("vertices-in.ftf", numVertices, &attr.FirstIn)
	Create("edges-next.ftf", numEdges, &attr.NextIn)
	Create("edges.ftf", numEdges, &attr.Edges)
	Create("distances.ftf", numEdges, &attr.Distances)
	Create("maxspeeds.ftf", numEdges, &attr.MaxSpeeds)
	Allocate(numEdges+1, &attr.Steps)
	
	bvSize := (numEdges + 7) / 8
	Create("oneway.ftf",      bvSize, &attr.Oneway)
	Create("access-car.ftf",  bvSize, &attr.AccessCar)
	Create("access-bike.ftf", bvSize, &attr.AccessBike)
	Create("access-foot.ftf", bvSize, &attr.AccessFoot)
	Create("ferries.ftf",     bvSize, &attr.Ferries)
	
	for i, _ := range attr.FirstIn {
		attr.FirstIn[i] = 0xffffffff
	}
	
	attr.E = ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter,
		ellipsoid.Longitude_is_symmetric, ellipsoid.Bearing_is_symmetric)
	
	return attr
}

func WriteEdgeAttributes(attr *EdgeAttributes) {
	Close(&attr.FirstIn)
	Close(&attr.NextIn)
	Close(&attr.Edges)
	Close(&attr.Distances)
	Close(&attr.MaxSpeeds)
	
	Close(&attr.Oneway)
	Close(&attr.AccessCar)
	Close(&attr.AccessBike)
	Close(&attr.AccessFoot)
	Close(&attr.Ferries)
	
	// Compute the step indices
	var steps []uint32
	Create("steps.ftf", len(attr.Steps), &steps)
	c := uint32(0)
	for i, step := range attr.Steps {
		steps[i] = c
		c += uint32(len(step))
	}
	
	// Output the compressed steps
	var step_positions []byte
	Create("step_positions.ftf", int(c), &step_positions)
	for i, step := range attr.Steps {
		copy(step_positions[steps[i]:], step)
	}
	
	Close(&steps)
	Close(&step_positions)
	attr.Region.Free()
}

func ComputeEdgeAttributes(graph *StreetGraph, vertices []uint32) {
	attr := NewEdgeAttributes(graph, vertices)
	graph.Visit(attr)
	WriteEdgeAttributes(attr)
}

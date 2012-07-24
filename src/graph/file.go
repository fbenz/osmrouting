package graph

import (
	"geo"
	"math"
	"mm"
	"path"
)

type GraphFile struct {
	// vertex -> first out/in edge
	FirstOut      []uint32
	FirstIn       []uint32
	// positions (at index 2 * i, 2 * i + 1)
	Coordinates   []int32
	
	// Accessibility bit vectors
	Access        [TransportMax][]byte
	AccessEdge    [TransportMax][]byte
	Oneway        []byte // should be distinguished by transport type
	
	// edge -> next edge (or to the same edge if this is the last in edge)
	NextIn        []uint32
	// for edge {u,v}, this array contains u^v
	Edges         []uint32
	
	// edge weights
	Weights       [MetricMax][]uint16
	
	// edge -> first step
	Steps         []uint32
	StepPositions []byte
}

type GraphFileEdgeIterator struct {
	Graph   *GraphFile
	Vertex  Vertex
	Forward bool
	Access  []byte
	Oneway  []byte
	Current Edge
	Out     bool
	Done    bool
}

// I/O

func OpenGraphFile(base string, ignoreErrors bool) (*GraphFile, error) {
	g := &GraphFile{}
	files := []struct{name string; p interface{}} {
		{"vertices.ftf",       &g.FirstOut},
		{"vertices-in.ftf",    &g.FirstIn},
		{"positions.ftf",      &g.Coordinates},
		{"vaccess-car.ftf",    &g.Access[Car]},
		{"vaccess-bike.ftf",   &g.Access[Bike]},
		{"vaccess-foot.ftf",   &g.Access[Foot]},
		{"access-car.ftf",     &g.AccessEdge[Car]},
		{"access-bike.ftf",    &g.AccessEdge[Bike]},
		{"access-foot.ftf",    &g.AccessEdge[Foot]},
		{"oneway.ftf",         &g.Oneway},
		{"edges-next.ftf",     &g.NextIn},
		{"edges.ftf",          &g.Edges},
		{"distances.ftf",      &g.Weights[Distance]},
		{"steps.ftf",          &g.Steps},
		{"step_positions.ftf", &g.StepPositions},
	}
	
	for _, file := range files {
		err := mm.Open(path.Join(base, file.name), file.p)
		if err != nil && !ignoreErrors {
			return nil, err
		}
	}
	return g, nil
}

func CloseGraphFile(g *GraphFile) error {
	files := []interface{} {
		&g.FirstOut, &g.FirstIn, &g.Coordinates,
		&g.Access[Car], &g.Access[Bike], &g.Access[Foot],
		&g.AccessEdge[Car], &g.AccessEdge[Bike], &g.AccessEdge[Foot],
		&g.Oneway, &g.NextIn, &g.Edges, &g.Weights[Distance],
		&g.Steps, &g.StepPositions,
	}
	for _, p := range files {
		err := mm.Close(p)
		if err != nil {
			return err
		}
	}
	return nil
}

// Graph Interface

func (g *GraphFile) VertexCount() int {
	return len(g.FirstIn)
}

func (g *GraphFile) EdgeCount() int {
	return len(g.Edges)
}

func GetBit(ary []byte, i uint) bool {
	return ary[i / 8] & (1 << (i % 8)) != 0
}

func (g *GraphFile) VertexAccessible(v Vertex, t Transport) bool {
	return GetBit(g.Access[t], uint(v))
}

func (g *GraphFile) VertexCoordinate(v Vertex) geo.Coordinate {
	lat := g.Coordinates[2 * int(v)]
	lng := g.Coordinates[2 * int(v) + 1]
	return geo.DecodeCoordinate(lat, lng)
}

func (g *GraphFile) VertexEdges(v Vertex, forward bool, t Transport) []Edge {
	result := make([]Edge, 0)
	// Add the out edges for v
	for i := g.FirstOut[v]; i < g.FirstOut[v+1]; i++ {
		// If we are iterating over the in edges and this is a oneway
		// road there is no corresponding in edge.
		if !forward && t < Foot && GetBit(g.Oneway, uint(i)) {
			continue
		}
		// Furthermore, the edge might be inaccessible to begin with.
		if t < TransportMax && !GetBit(g.AccessEdge[t], uint(i)) {
			continue
		}
		// Otherwise we can take this edge
		result = append(result, Edge(i))
	}
	
	// The in edges are stored as a linked list. -1 means no in edges.
	i := g.FirstIn[v]
	if i == 0xffffffff {
		return result
	}
	
	for {
		// If we are iterating over the out edges and this is a oneway
		// road there is no corresponding out edge.
		if forward && t < Foot && GetBit(g.Oneway, uint(i)) {
			goto NextEdge
		}
		// Access restrictions.
		if t < TransportMax && !GetBit(g.AccessEdge[t], uint(i)) {
			goto NextEdge
		}
		result = append(result, Edge(i))
		// Continue with the next in edge, if any.
NextEdge:
		if i == g.NextIn[i] {
			break
		}
		i = g.NextIn[i]
	}
	return result
}

func (g *GraphFile) VertexEdgeIterator(v Vertex, forward bool, t Transport) EdgeIterator {
	oneway := g.Oneway
	if t == Foot || t == TransportMax {
		oneway = nil
	}
	access := []byte(nil)
	if t < TransportMax {
		access = g.AccessEdge[t]
	}
	return &GraphFileEdgeIterator {
		// Static fields:
		Graph:   g,
		Vertex:  v,
		Forward: forward,
		Access:  access,
		Oneway:  oneway,
		// Mutable fields:
		Current: Edge(g.FirstOut[v]),
		Out:     true,
		Done:    false,
	}
}

func (i *GraphFileEdgeIterator) Accessible(e Edge) bool {
	if i.Access != nil && !GetBit(i.Access, uint(e)) {
		return false
	}
	if i.Out == i.Forward || i.Oneway == nil {
		return true
	}
	return !GetBit(i.Oneway, uint(e))
}

func (i *GraphFileEdgeIterator) Next() (Edge, bool) {
	g := i.Graph
	
	// Iterate over the out edges first
	if i.Out {
		l := g.FirstOut[i.Vertex + 1]
		for c := i.Current; uint32(c) < l; c++ {
			if i.Accessible(c) {
				i.Current = c + 1
				return c, true
			}
		}
		// Finished with the out edges
		i.Out = false
		i.Current = Edge(g.FirstIn[i.Vertex])
		if i.Current == Edge(-1) {
			i.Done = true
		}
	}
	
	if i.Done {
		return 0, false
	}
	
	c := i.Current
	for {
		if g.NextIn[c] != uint32(c) {
			i.Current = Edge(g.NextIn[c])
		} else {
			i.Done = true
		}
		
		if i.Accessible(c) {
			c = i.Current
			return c, true
		} else if i.Done {
			return 0, false
		}
		
		c = i.Current
	}
	return 0, false
}

func (g *GraphFile) EdgeOpposite(e Edge, from Vertex) Vertex {
	return Vertex(g.Edges[e]) ^ from
}

func (g *GraphFile) EdgeSteps(e Edge, from Vertex) []geo.Coordinate {
	// In order to decode the step positions we need the starting vertex.
	// Additionally, if this vertex is not "from", we will need to reverse
	// the steps positions before returning.
	firstEdge := g.FirstOut[from]
	lastEdge  := g.FirstOut[from+1]
	forward   := firstEdge <= uint32(e) && uint32(e) < lastEdge
	start     := from
	if !forward {
		start = g.EdgeOpposite(e, from)
	}
	
	firstStep := g.Steps[e]
	lastStep  := g.Steps[e+1]
	initial   := g.VertexCoordinate(start)
	step      := geo.DecodeStep(initial, g.StepPositions[firstStep:lastStep])
	
	if !forward {
		for i, j := 0, len(step)-1; i < j; i, j = i+1, j-1 {
			step[i], step[j] = step[j], step[i]
		}
	}
	
	return step
}

func HalfToFloat32(a uint16) float32 {
	s := uint32(a & 0x8000) << 16
	e := uint32(a >> 10) & 0x1f
	m := uint32(a & 0x3ff)
	
	if e == 0 {
		// +/- 0, we don't produce denormals
		// (mainly because we would have to turn them into a
		// normalized number here and that's costly)
		return math.Float32frombits(s)
	} else if e == 31 {
		if m == 0 {
			// Infinity
			return math.Float32frombits(s | 0x7f800000)
		} else {
			// NaN
			return math.Float32frombits(0x7f800000 | (m << 13))
		}
	}
	
	return math.Float32frombits(s | ((e + 112) << 23) | (m << 13))
}

func HalfToFloat64(a uint16) float64 {
	return float64(HalfToFloat32(a))
}

func (g *GraphFile) EdgeWeight(e Edge, t Transport, m Metric) float64 {
	return HalfToFloat64(g.Weights[m][e])
}

package graph

import (
	"alg"
	"geo"
	"mm"
	"path"
)

const Sentinel uint32 = 0xffffffff

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
	
	// Ugly hack: we can't have too many open files...
	bitvectors := []*[]byte {
		&g.Access[Car], &g.Access[Bike], &g.Access[Foot],
		&g.AccessEdge[Car], &g.AccessEdge[Bike], &g.AccessEdge[Foot],
		&g.Oneway,
	}
	for _, bv := range bitvectors {
		p := *bv
		*bv = make([]byte, len(p))
		copy(*bv, p)
		err := mm.Close(&p)
		if err != nil {
			return nil, err
		}
	}
	
	return g, nil
}

func CloseGraphFile(g *GraphFile) error {
	files := []interface{} {
		&g.FirstOut, &g.FirstIn, &g.Coordinates,
		//&g.Access[Car], &g.Access[Bike], &g.Access[Foot],
		//&g.AccessEdge[Car], &g.AccessEdge[Bike], &g.AccessEdge[Foot],
		//&g.Oneway,
		&g.NextIn, &g.Edges, &g.Weights[Distance],
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

func (g *GraphFile) VertexAccessible(v Vertex, t Transport) bool {
	return alg.GetBit(g.Access[t], uint(v))
}

func (g *GraphFile) VertexCoordinate(v Vertex) geo.Coordinate {
	lat := g.Coordinates[2 * int(v)]
	lng := g.Coordinates[2 * int(v) + 1]
	return geo.DecodeCoordinate(lat, lng)
}

func (g *GraphFile) VertexEdges(v Vertex, forward bool, t Transport, buf []Edge) []Edge {
	// This is rather nice: buf[:0] sets the length to 0 but does not change the capacity.
	// In effect calls to append will not allocate a new array if the capacity is already
	// sufficient. This is much faster than using an iterator, since every interface call
	// is indirect.
	result := buf[:0]
	
	// So, at this point you are probably thinking:
	// "What The Fuck? Are you implementing common compiler optimizations by hand?"
	// To which I am forced to answer that yes, I am. I wish this were a joke, but this
	// saved 8 seconds (~25%) of the running time in the scc finding program.
	// TODO: Implement loop unswitching, cse and some algebraic identities in the go compiler.
	
	// Add the out edges for v
	first := g.FirstOut[v]
	last  := g.FirstOut[v+1]
	access := g.AccessEdge[t]
	if forward || t == Foot {
		// No need to consider the oneway flags
		for i := first; i < last; i++ {
			index := i >> 3
			bit := byte(1 << (i & 7))
			if access[index] & bit == 0 {
				continue
			}
			result = append(result, Edge(i))
		}
	} else {
		// Consider the oneway flags...
		oneway := g.Oneway
		for i := first; i < last; i++ {
			index := i >> 3
			bit := byte(1 << (i & 7))
			if access[index] & bit == 0 || oneway[index] & bit != 0 {
				continue
			}
			result = append(result, Edge(i))
		}
	}
	
	// The in edges are stored as a linked list.
	i := g.FirstIn[v]
	if i == Sentinel {
		return result
	}
	
	if !forward || t == Foot {
		// As above, no need to consider the oneway flags
		for {
			index := i >> 3
			bit := byte(1 << (i & 7))
			if access[index] & bit != 0 {
				result = append(result, Edge(i))
			}
			if i == g.NextIn[i] {
				break
			}
			i = g.NextIn[i]
		}
	} else {
		// Need to consider the oneway flags.
		oneway := g.Oneway
		for {
			index := i >> 3
			bit := byte(1 << (i & 7))
			if access[index] & bit != 0 && oneway[index] & bit == 0 {
				result = append(result, Edge(i))
			}
			if i == g.NextIn[i] {
				break
			}
			i = g.NextIn[i]
		}
	}
	
	return result
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

func (g *GraphFile) EdgeWeight(e Edge, t Transport, m Metric) float64 {
	return alg.HalfToFloat64(g.Weights[m][e])
}

// Dijkstra interface

func (g *GraphFile) VertexNeighbors(v Vertex, forward bool, t Transport, m Metric, buf []Dart) []Dart {
	// This is copy pasted from Vertex Edges, and in a perfect world there would never
	// be a reason to do something like this. Alas the world of go is not perfect yet.
	result := buf[:0]
	
	// Add the out edges for v
	first := g.FirstOut[v]
	last  := g.FirstOut[v+1]
	access := g.AccessEdge[t]
	if forward || t == Foot {
		// No need to consider the oneway flags
		for i := first; i < last; i++ {
			index := i >> 3
			bit := byte(1 << (i & 7))
			if access[index] & bit == 0 {
				continue
			}
			u := Vertex(g.Edges[i]) ^ v
			w := alg.HalfToFloat32(g.Weights[m][i])
			result = append(result, Dart{u, w})
		}
	} else {
		// Consider the oneway flags...
		oneway := g.Oneway
		for i := first; i < last; i++ {
			index := i >> 3
			bit := byte(1 << (i & 7))
			if access[index] & bit == 0 || oneway[index] & bit != 0 {
				continue
			}
			u := Vertex(g.Edges[i]) ^ v
			w := alg.HalfToFloat32(g.Weights[m][i])
			result = append(result, Dart{u, w})
		}
	}
	
	// The in edges are stored as a linked list.
	i := g.FirstIn[v]
	if i == Sentinel {
		return result
	}
	
	if !forward || t == Foot {
		// As above, no need to consider the oneway flags
		for {
			index := i >> 3
			bit := byte(1 << (i & 7))
			if access[index] & bit != 0 {
				u := Vertex(g.Edges[i]) ^ v
				w := alg.HalfToFloat32(g.Weights[m][i])
				result = append(result, Dart{u, w})
			}
			if i == g.NextIn[i] {
				break
			}
			i = g.NextIn[i]
		}
	} else {
		// Need to consider the oneway flags.
		oneway := g.Oneway
		for {
			index := i >> 3
			bit := byte(1 << (i & 7))
			if access[index] & bit != 0 && oneway[index] & bit == 0 {
				u := Vertex(g.Edges[i]) ^ v
				w := alg.HalfToFloat32(g.Weights[m][i])
				result = append(result, Dart{u, w})
			}
			if i == g.NextIn[i] {
				break
			}
			i = g.NextIn[i]
		}
	}
	
	return result
}

// Raw Interface (used to implement other tools working with GraphFiles)

func (g *GraphFile) EdgeAccessible(v Edge, t Transport) bool {
	return alg.GetBit(g.AccessEdge[t], uint(v))
}

func (g *GraphFile) VertexRawEdges(v Vertex, buf []Edge) []Edge {
	result := buf[:0]
	
	for i := g.FirstOut[v]; i < g.FirstOut[v+1]; i++ {
		result = append(result, Edge(i))
	}
	
	i := g.FirstIn[v]
	if i == Sentinel {
		return result
	}
	
	for {
		result = append(result, Edge(i))
		if i == g.NextIn[i] {
			break
		}
		i = g.NextIn[i]
	}
	
	return result
}

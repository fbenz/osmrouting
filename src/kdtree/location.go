/*
 * Copyright 2014 Florian Benz, Steven Schäfer, Bernhard Schommer
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kdtree

import (
	"geo"
	"graph"
)

type Location struct {
	Graph   *graph.GraphFile
	EC      uint64
	Cluster int
}

func (l Location) Vertex() graph.Vertex {
	return graph.Vertex(l.EC >> (EdgeOffsetBits + StepOffsetBits))
}

func (l Location) EdgeOffset() uint32 {
	return uint32((l.EC >> StepOffsetBits) & MaxEdgeOffset)
}

func (l Location) StepOffset() int {
	return int(l.EC & MaxStepOffset)
}

func (l Location) IsVertex() bool {
	return l.EdgeOffset() == MaxEdgeOffset && l.StepOffset() == MaxStepOffset
}

func (l Location) Edge() graph.Edge {
	vertex := l.Vertex()
	edgeOffset := l.EdgeOffset()
	stepOffset := l.StepOffset()
	if edgeOffset == MaxEdgeOffset && stepOffset == MaxStepOffset {
		return graph.Edge(-1)
	}
	return graph.Edge(l.Graph.FirstOut[vertex] + edgeOffset)
}

func (l Location) Decode(forward bool, transport graph.Transport, steps *[]geo.Coordinate) []graph.Way {
	g := l.Graph
	vertex := l.Vertex()
	edge := l.Edge()
	offset := l.StepOffset()

	if int(edge) == -1 {
		// The easy case, where we hit some vertex exactly.
		target := g.VertexCoordinate(vertex)
		way := graph.Way{Length: 0, Vertex: vertex, Steps: nil, Target: target}
		return []graph.Way{way}
	}

	oneway := g.EdgeOneway(edge, transport)
	t1 := vertex                       // start vertex
	t2 := g.EdgeOpposite(edge, vertex) // end vertex

	// now we can allocate the way corresponding to (edge,offset),
	// but there are three cases to consider:
	// - if the way is bidirectional we have to compute both directions,
	//   if forward == true the from the offset two both endpoints,
	//   and the reverse otherwise
	// - if the way is unidirectional then we have to compute the way
	//   from the StartPoint to offset if forward == false
	// - otherwise we have to compute the way from offset to the EndPoint
	(*steps) = g.EdgeSteps(edge, vertex, *steps)
	s := *steps

	b1 := make([]geo.Coordinate, len(s[:offset]))
	b2 := make([]geo.Coordinate, len(s[offset+1:]))
	copy(b1, s[:offset])
	copy(b2, s[offset+1:])
	l1 := geo.StepLength(s[:offset+1])
	l2 := geo.StepLength(s[offset:])
	t1Coord := g.VertexCoordinate(t1)
	t2Coord := g.VertexCoordinate(t2)
	d1, _ := e.To(t1Coord.Lat, t1Coord.Lng, s[0].Lat, s[0].Lng)
	d2, _ := e.To(t2Coord.Lat, t2Coord.Lng, s[len(s)-1].Lat, s[len(s)-1].Lng)
	l1 += d1
	l2 += d2
	target := s[offset]

	if !forward {
		reverse(b2)
	} else {
		reverse(b1)
	}

	var w []graph.Way
	if !oneway {
		w = make([]graph.Way, 2) // bidirectional
		w[0] = graph.Way{Length: l1, Vertex: t1, Steps: b1, Forward: forward, Target: target}
		w[1] = graph.Way{Length: l2, Vertex: t2, Steps: b2, Forward: forward, Target: target}
	} else {
		w = make([]graph.Way, 1) // one way
		if forward {
			w[0] = graph.Way{Length: l2, Vertex: t2, Steps: b2, Forward: forward, Target: target}
		} else {
			w[0] = graph.Way{Length: l1, Vertex: t1, Steps: b1, Forward: forward, Target: target}
		}
	}
	return w
}

func reverse(steps []geo.Coordinate) {
	for i, j := 0, len(steps)-1; i < j; i, j = i+1, j-1 {
		steps[i], steps[j] = steps[j], steps[i]
	}
}

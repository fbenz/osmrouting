package route

import (
	"graph"
)

type Element struct {
	vertex   graph.Vertex
	index    int
	priority float64
	p        graph.Vertex
	ep       graph.Edge
}

func NewElement(vertex graph.Vertex, priority float64) *Element {
	return &Element{
		vertex:   vertex,
		index:    -1,
		priority: priority,
		p:        vertex,
	}
}

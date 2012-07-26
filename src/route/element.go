package route

import (
	"newgraph"
	//"fmt"
)

type Element struct {
	vertex   newgraph.Vertex
	index    int
	priority float64
	p        newgraph.Vertex
	ep       newgraph.Edge
}

func NewElement(vertex newgraph.Vertex, priority float64) *Element {
	/*if d < 100 {
		fmt.Printf("%v vs. %v\n", priority, int32(priority * 1e4))
	}*/
	return &Element{
		vertex:   vertex,
		index:    -1,
		priority: priority,
		p:        vertex,
	}
}

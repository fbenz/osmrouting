package alg

import (
	"graph"
)

type Element struct {
	node graph.Node
	index int
	priority float64
	d float64
	p graph.Node
	ep graph.Edge
}

func NewElement(node graph.Node, priority, d float64) *Element {
	return &Element{
		node: node,
		index: -1,
		priority: priority,
		d: d,
		p: node,
	}
}

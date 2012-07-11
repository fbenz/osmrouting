package alg

import (
	"graph"
	//"fmt"
)

type Element struct {
	node graph.Node
	index int
	priority int32
	d float64
	p graph.Node
	ep graph.Edge
}

func NewElement(node graph.Node, priority, d float64) *Element {
	/*if d < 100 {
		fmt.Printf("%v vs. %v\n", priority, int32(priority * 1e4))
	}*/
	return &Element{
		node: node,
		index: -1,
		priority: int32(priority),
		d: d,
		p: node,
	}
}

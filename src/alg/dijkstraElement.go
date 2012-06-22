package alg

import (
	"graph"
)

type DijkstraElement struct {
	node graph.Node
	index int
	priority float64
	d float64
	p graph.Node
	ep graph.Edge
}

func NewDijkstraElement(node graph.Node, priority, d float64) *DijkstraElement {
	return &DijkstraElement{
		node: node,
		index: -1,
		priority: priority,
		d: d,
		p: node,
	}
}

func (e *DijkstraElement) Index() int {
	return e.index
}

func (e *DijkstraElement) SetIndex(i int) {
	e.index = i
}

func (e *DijkstraElement) Priority() float64 {
	return e.priority
}

func (e *DijkstraElement) SetPriority(p float64) {
	e.priority = p
}

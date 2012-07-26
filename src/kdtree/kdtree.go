// TODO move to graph?
package kdtree

import (
	"geo"
	"graph"
)

// encoded step: vertex index (16bit) + edge offset (8bit) + step offset (8bit)
type KdTree struct {
	Graph        graph.Graph
	EncodedSteps []uint32
	Coordinates  []geo.Coordinate
}

func (t KdTree) Len() int {
	return len(t.EncodedSteps)
}

func (t KdTree) Swap(i, j int) {
	t.Coordinates[i], t.Coordinates[j] = t.Coordinates[j], t.Coordinates[i]
	t.EncodedSteps[i], t.EncodedSteps[j] = t.EncodedSteps[j], t.EncodedSteps[i]
}

// TODO move to graph?
package kdtree

import (
	"geo"
	"graph"
)

const (
	VertexIndexBits = 16
	EdgeOffsetBits  = 5
	StepOffsetBits  = 11
	MaxVertexIndex  = 0xFFFF
	MaxEdgeOffset   = 0x1F
	MaxStepOffset   = 0x7FF
)

// encoded step: vertex index (16bit) + edge offset (8bit) + step offset (8bit)
type KdTree struct {
	Graph        graph.Graph
	EncodedSteps []uint32
	Coordinates  []geo.Coordinate
}

type ClusterKdTree struct {
	Overlay *KdTree
	Cluster []*KdTree
	BBoxes  []geo.BBox
}

func (t KdTree) Len() int {
	return len(t.EncodedSteps)
}

func (t KdTree) Swap(i, j int) {
	t.Coordinates[i], t.Coordinates[j] = t.Coordinates[j], t.Coordinates[i]
	t.EncodedSteps[i], t.EncodedSteps[j] = t.EncodedSteps[j], t.EncodedSteps[i]
}

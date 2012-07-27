// TODO move to graph?
package kdtree

import (
	"geo"
	"graph"
)

const (
	TotalBits       = 34
	TypeSize        = 64
	VertexIndexBits = 18
	EdgeOffsetBits  = 5
	StepOffsetBits  = 11
	MaxVertexIndex  = 0x3FFFF
	MaxEdgeOffset   = 0x1F
	MaxStepOffset   = 0x7FF
)

// encoded step: vertex index (18bit) + edge offset (8bit) + step offset (8bit)
type KdTree struct {
	Graph        graph.Graph
	EncodedSteps []uint64
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
	tmp := t.EncodedStep(j)
	t.SetEncodedStep(j, t.EncodedStep(i))
	t.SetEncodedStep(i, tmp)
}

// TODO not perfect as it might be off by one
func (t KdTree) EncodedStepSize() int {
	return (len(t.EncodedSteps) * TypeSize) / TotalBits
}

func (t KdTree) EncodedStep(i int) uint64 {
	index := i * TotalBits / TypeSize
	offset := i * TotalBits % TypeSize
	if offset+TotalBits <= TypeSize {
		// contained in one uint64
		shift := uint(TypeSize - (offset + TotalBits))
		mask := (uint64(1) << (TotalBits + 1)) - 1
		return (t.EncodedSteps[index] >> shift) & mask
	}
	// split over two uint64
	first := uint(TypeSize - (offset + TotalBits))
	second := uint(TotalBits - first)

	fMask := (uint64(1) << (first + 1)) - 1
	result := (t.EncodedSteps[index] & fMask) << second

	sShift := TypeSize - second
	sMask := (uint64(1) << (second + 1)) - 1
	result |= (t.EncodedSteps[index+1] >> sShift) & sMask
	return result
}

func (t KdTree) SetEncodedStep(i int, s uint64) {
	index := i * TotalBits / TypeSize
	offset := i * TotalBits % TypeSize
	if offset+TotalBits <= TypeSize {
		// contained in one uint64
		shift := uint(TypeSize - (offset + TotalBits))
		mask := (uint64(1) << (TotalBits + 1)) - 1
		t.EncodedSteps[index] ^= t.EncodedSteps[index] & (mask << shift)
		t.EncodedSteps[index] |= s << shift
	}
	// split over two uint64
	first := uint(TypeSize - (offset + TotalBits))
	second := uint(TotalBits - first)

	fMask := (uint64(1) << (first + 1)) - 1
	t.EncodedSteps[index] ^= t.EncodedSteps[index] & fMask
	t.EncodedSteps[index] |= s >> second

	sShift := TypeSize - second
	sMask := (uint64(1) << (second + 1)) - 1
	t.EncodedSteps[index+1] ^= t.EncodedSteps[index+1] & (sMask << sShift)
	t.EncodedSteps[index+1] ^= s << sShift
}

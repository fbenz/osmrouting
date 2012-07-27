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
	MaxVertexIndex  = 0x3FFFF // (1 << VertexIndexBits) - 1
	MaxEdgeOffset   = 0x1F    // (1 << EdgeOffsetBits) - 1
	MaxStepOffset   = 0x7FF   // (1 << StepOffsetBits) - 1
)

// encoded step: vertex index (18bit) + edge offset (8bit) + step offset (8bit)
type KdTree struct {
	Graph        graph.Graph
	EncodedSteps []uint64
	Coordinates  []geo.Coordinate
	// It is inefficient to create a sub slice of EncodedSteps due to the used encoding.
	// Thus, we use start and end pointer instead. 
	EncodedStepsStart int
	EncodedStepsEnd   int
}

type ClusterKdTree struct {
	Overlay *KdTree
	Cluster []*KdTree
	BBoxes  []geo.BBox
}

func (t *KdTree) Len() int {
	return len(t.Coordinates)
}

func (t *KdTree) Swap(i, j int) {
	t.Coordinates[i], t.Coordinates[j] = t.Coordinates[j], t.Coordinates[i]
	tmp := t.EncodedStep(j)
	t.SetEncodedStep(j, t.EncodedStep(i))
	t.SetEncodedStep(i, tmp)
}

func (t *KdTree) EncodedStepLen() int {
	if t.EncodedStepsEnd > 0 {
		return t.EncodedStepsEnd + 1
	}
	l := (len(t.EncodedSteps) * TypeSize) / TotalBits
	if l > 0 && t.EncodedStep(l-1) == (1<<TotalBits)-1 {
		return l - 1
	}
	return l
}

func (t *KdTree) EncodedStep(i int) uint64 {
	index := t.EncodedStepsStart + i*TotalBits/TypeSize
	offset := i * TotalBits % TypeSize
	if offset+TotalBits <= TypeSize {
		// contained in one uint64
		mask := (uint64(1) << TotalBits) - 1
		return (t.EncodedSteps[index] >> uint(offset)) & mask
	}
	// split over two uint64
	first := uint(TypeSize - offset)
	second := uint(TotalBits - first)

	fMask := ((uint64(1) << first) - 1)
	result := ((t.EncodedSteps[index] >> uint(offset)) & fMask) << second

	//sShift := TypeSize - second
	sMask := (uint64(1) << second) - 1
	result |= t.EncodedSteps[index+1] & sMask
	return result
}

func (t *KdTree) SetEncodedStep(i int, s uint64) {
	index := t.EncodedStepsStart + i*TotalBits/TypeSize
	offset := i * TotalBits % TypeSize
	if offset+TotalBits <= TypeSize {
		// contained in one uint64
		mask := (uint64(1) << TotalBits) - 1
		t.EncodedSteps[index] ^= t.EncodedSteps[index] & (mask << uint(offset))
		t.EncodedSteps[index] |= s << uint(offset)
	} else {
		// split over two uint64
		first := uint(TypeSize - offset)
		second := uint(TotalBits - first)

		fMask := (uint64(1) << first) - 1
		t.EncodedSteps[index] ^= t.EncodedSteps[index] & (fMask << uint(offset))
		t.EncodedSteps[index] |= (s >> second) << uint(offset)

		sMask := (uint64(1) << second) - 1
		t.EncodedSteps[index+1] ^= t.EncodedSteps[index+1] & sMask
		t.EncodedSteps[index+1] |= s & sMask
	}
}

func (t *KdTree) AppendEncodedStep(s uint64) {
	l := t.EncodedStepLen()
	index := l * TotalBits / TypeSize
	offset := l * TotalBits % TypeSize
	if index >= len(t.EncodedSteps) {
		t.EncodedSteps = append(t.EncodedSteps, (1<<64)-1)
	}
	if offset+TotalBits >= TypeSize && index+1 >= len(t.EncodedSteps) {
		t.EncodedSteps = append(t.EncodedSteps, (1<<64)-1)
	}
	t.SetEncodedStep(l, s)
}

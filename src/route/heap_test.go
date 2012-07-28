package route

import (
	"graph"
	"testing"
	"math/rand"
)

const (
	NumTests = 100
	MinSize = 50
	MaxSize = 1000
)

func TestHeapSort(t *testing.T) {
	h := &Heap{}
	for i := 0; i < NumTests; i++ {
		n := rand.Intn(MaxSize - MinSize) + MinSize
		h.Reset(n)
		// Add a random permuation to the heap.
		p := rand.Perm(n)
		for j, x := range p {
			h.Push(graph.Vertex(j), float32(x)) // / float32(n))
		}
		
		// Ensure that the heap property holds.
		for j := 0; j < n; j++ {
			parent := h.Items[j].Priority
			for k := 1; k <= BranchingFactor; k++ {
				if BranchingFactor * j + k >= n {
					break
				}
				child := h.Items[BranchingFactor * j + k].Priority
				if parent > child {
					t.Errorf("Heap property violated: p[%v] = %v > %v = p[%v].",
						j, parent, child, BranchingFactor * j + k)
				}
			}
		}
		
		// Ensure that what we get out of it is sorted.
		prev := float32(-1)
		for !h.Empty() {
			_, curr := h.Pop()
			if curr < prev {
				t.Errorf("Inversion in Heap.Pop, prev: %v, curr: %v.", prev, curr)
			}
			prev = curr
		}
	}
}

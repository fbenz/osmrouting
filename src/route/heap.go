package route

import (
	//"fmt"
	"graph"
	"reflect"
	"unsafe"
)

const (
	D = 2
	BranchingFactor = 1 << D
	CacheLineSize = 64 // bytes
)

// The color of a vertex represents its traversal state:
// - White is unvisited
// - Gray  is visited, but not yet finished
// - Black is processed
type Color int
const (
	White Color = iota
	Black
	Gray
)

// Ensure that a heap item is always 64 bits wide.
// Less would be better, but not at the expense of alignment.
type Item struct {
	Priority float32
	Vertex   uint32
}

type Heap struct {
	// Map vertices to items. More specifically, Index[v] == 0, 1
	// represents a vertex of color White and Black respectively.
	// If Index[v] >= 2, the vertex is at Index[v] - 2 in the Items array.
	Index []int
	// The array with all the heap elements.
	Items []Item
}

// Allocation

func (h *Heap) Reset(vertexCount int) {
	// We might have to allocate a new index unless the current index is large
	// enough for vertexCount elements. Otherwise, we have to clear it, but do
	// not allocate new storage.
	if h.Index == nil || cap(h.Index) < vertexCount {
		//fmt.Printf("Reallocating the Heap Index with capacity %v.\n", vertexCount)
		h.Index = make([]int, vertexCount)
	} else {
		h.Index = h.Index[:vertexCount]
		for i := range h.Index {
			h.Index[i] = 0
		}
	}
	
	// The Items array starts out empty, so we never need to clear it. On the
	// other hand, the allocation is more complicated since we have to ensure
	// the proper alignment.
	if h.Items == nil || cap(h.Items) < vertexCount {
		itemsPerCacheLine := CacheLineSize / 8
		//fmt.Printf("Reallocating the Heap with capacity %v.\n", vertexCount+itemsPerCacheLine)
		items := make([]Item, vertexCount + itemsPerCacheLine)
		data := (*reflect.SliceHeader)(unsafe.Pointer(&items)).Data
		// Round up and back off.
		skip := (CacheLineSize - (data & (CacheLineSize-1))) / 8 - 1
		h.Items = items[skip:skip]
	} else {
		h.Items = h.Items[:0]
	}
}

// Algorithms

func (h *Heap) move(item Item, to int) {
	h.Index[int(item.Vertex)] = to+2
	h.Items[to] = item
}

func (h *Heap) up(index int, item Item) {
	for index > 0 {
		parentIndex := (index - 1) >> D
		parentItem  := h.Items[parentIndex]
		if parentItem.Priority <= item.Priority {
			break
		}
		h.move(parentItem, index)
		index = parentIndex
	}
	h.move(item, index)
}

func (h *Heap) down(index int, item Item) {
	if len(h.Items) > 1 {
		// Avoid doing too many bounds checks by first processing children in
		// large batches and handling the remaining case later on.
		// Note: compile with -gcflags '-B', otherwise the compiler will make
		// a mess of this.
		child := (index << D) + 1
		for child + BranchingFactor <= len(h.Items) {
			// Compute the child with minimum priority.
			min := child
			minPriority := h.Items[child].Priority
			for i := 1; i < BranchingFactor; i++ {
				priority := h.Items[child + i].Priority
				if priority < minPriority {
					min = child + i
					minPriority = priority
				}
			}
			// If the heap property holds we are done.
			if minPriority >= item.Priority {
				h.move(item, index)
				return
			}
			// Otherwise shift the minimum child up one level and repeat.
			h.move(h.Items[min], index)
			index = min
			child = (index << D) + 1
		}
		
		// Handle the leftovers.
		if child < len(h.Items) {
			// Find the child of minimum priority among the last array
			// elements, [child:].
			min := child
			minPriority := h.Items[child].Priority
			for i := min + 1; i < len(h.Items); i++ {
				priority := h.Items[i].Priority
				if priority < minPriority {
					min = i
					minPriority = priority
				}
			}
			// The rest is as above, except for the fact that we are
			// done in any case.
			if item.Priority > minPriority {
				h.move(h.Items[min], index)
				index = min
			}
		}
	}
	h.move(item, index)
}

// Interface

func (h *Heap) Empty() bool {
	return len(h.Items) == 0
}

func (h *Heap) Color(vertex graph.Vertex) Color {
	index := h.Index[int(vertex)]
	if index < 2 {
		return Color(index)
	}
	return Gray
}

// Pre-Condition: Color(vertex) == Gray
func (h *Heap) Priority(vertex graph.Vertex) float32 {
	return h.Items[h.Index[int(vertex)]-2].Priority
}

// Pre-Condition: !h.Empty()
func (h *Heap) Top() float32 {
	return h.Items[0].Priority
}

// Pre-Condition: h.Color == White
func (h *Heap) Push(vertex graph.Vertex, prio float32) {
	h.Items = h.Items[:len(h.Items)+1] // Add an additional slot
	h.up(len(h.Items)-1, Item{prio, uint32(vertex)})
}

// Pre-Condition: h.Color(vertex) == Gray, h.Priority(vertex) >= prio
func (h *Heap) DecreaseKey(vertex graph.Vertex, prio float32) {
	index := h.Index[int(vertex)]-2
	h.up(index, Item{prio, uint32(vertex)})
}

// Pre-Conditions: !h.Empty()
// Post-Condition: h.Color(vertex) == Black
func (h *Heap) Pop() (graph.Vertex, float32) {
	root := h.Items[0]
	h.Index[int(root.Vertex)] = int(Black)
	if len(h.Items) > 1 {
		last := h.Items[len(h.Items)-1]
		h.Items = h.Items[:len(h.Items)-1]
		h.down(0, last)
	} else {
		h.Items = h.Items[:0]
	}
	return graph.Vertex(root.Vertex), root.Priority
}
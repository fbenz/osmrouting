package alg

import (
	"container/heap"
	"graph"
	"testing"
)

func compareNodes(t *testing.T, expected, actual graph.Node) {
	if expected != actual {
		t.Fatalf("Returned wrong element: expected value %v but was %v", expected, actual)
	}
}

func TestOrderOfReturnedElements(t *testing.T) {
	elem1 := NewElement(1 /* node */, 0 /* priority */, 0)
	elem2 := NewElement(2 /* node */, 5 /* priority */, 0)
	elem3 := NewElement(3 /* node */, 2 /* priority */, 0)
	elem4 := NewElement(4 /* node */, 4 /* priority */, 0)

	q := NewPriorityQueue(4 /* initialCapacity */)
	heap.Push(&q, elem1)
	heap.Push(&q, elem2)
	heap.Push(&q, elem3)
	heap.Push(&q, elem4)
	compareNodes(t, 1, (heap.Pop(&q)).(*Element).node)
	compareNodes(t, 3, (heap.Pop(&q)).(*Element).node)
	compareNodes(t, 4, (heap.Pop(&q)).(*Element).node)
	compareNodes(t, 2, (heap.Pop(&q)).(*Element).node)
}

func TestIncreaseSize(t *testing.T) {
	initialCapacity := 4
	n := 100

	q := NewPriorityQueue(initialCapacity)
	for i := 0; i < n; i++ {
		heap.Push(&q, NewElement(1 /* value */, 0 /* priority */, 0))
	}
	if q.Len() < 100 {
		t.Fatalf("Queue is too small")
	}
}

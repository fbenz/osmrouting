package alg

import (
	"container/heap"
	"testing"
)

func compareInt(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Fatalf("Returned wrong element: expected value %v but was %v", expected, actual)
	}
}

func TestOrderOfReturnedElements(t *testing.T) {
	elem1 := NewElement(1 /* value */, 0 /* priority */)
	elem2 := NewElement(2 /* value */, 5 /* priority */)
	elem3 := NewElement(3 /* value */, 2 /* priority */)
	elem4 := NewElement(4 /* value */, 4 /* priority */)

	q := New(4 /* initialCapacity */)
	heap.Push(&q, elem1)
	heap.Push(&q, elem2)
	heap.Push(&q, elem3)
	heap.Push(&q, elem4)
	compareInt(t, 1, (heap.Pop(&q)).(*Element).Value.(int))
	compareInt(t, 3, (heap.Pop(&q)).(*Element).Value.(int))
	compareInt(t, 4, (heap.Pop(&q)).(*Element).Value.(int))
	compareInt(t, 2, (heap.Pop(&q)).(*Element).Value.(int))
}

func TestIncreaseSize(t *testing.T) {
	initialCapacity := 4
	n := 100

	q := New(initialCapacity)
	for i := 0; i < n; i++ {
		heap.Push(&q, NewElement(1 /* value */, 0 /* priority */))
	}
	if q.Len() < 100 {
		t.Fatalf("Queue is too small")
	}
}

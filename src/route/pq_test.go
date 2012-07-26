package route

import (
	"graph"
	"testing"
)

func compareNodes(t *testing.T, expected, actual graph.Vertex) {
	if expected != actual {
		t.Fatalf("Returned wrong element: expected value %v but was %v", expected, actual)
	}
}

func TestOrderOfReturnedElements(t *testing.T) {
	elem1 := NewElement(1 /* node */, 0 /* priority */)
	elem2 := NewElement(2 /* node */, 5 /* priority */)
	elem3 := NewElement(3 /* node */, 2 /* priority */)
	elem4 := NewElement(4 /* node */, 4 /* priority */)

	q := NewPriorityQueue(4 /* initialCapacity */)
	PushElement(&q, elem1)
	PushElement(&q, elem2)
	PushElement(&q, elem3)
	PushElement(&q, elem4)
	compareNodes(t, 1, ExtractMin(&q).vertex)
	compareNodes(t, 3, ExtractMin(&q).vertex)
	compareNodes(t, 4, ExtractMin(&q).vertex)
	compareNodes(t, 2, ExtractMin(&q).vertex)
}

func TestIncreaseSize(t *testing.T) {
	initialCapacity := 4
	n := 100

	q := NewPriorityQueue(initialCapacity)
	for i := 0; i < n; i++ {
		PushElement(&q, NewElement(1 /* value */, 0 /* priority */))
	}
	if Len(&q) < 100 {
		t.Fatalf("Queue is too small")
	}
}

func TestBuildFromArray(t *testing.T) {
	q := make([]*Element, 0, 100)
	for i := 99; i >= 0; i-- {
		priority := float64(99 - i)
		q = append(q, NewElement(1, priority))
	}
	pq := ArrayToPriorityQueue(&q)
	for i := 0; i < 100; i++ {
		e := ExtractMin(pq)
		if e.priority != float64(i) {
			t.Fatalf("Element not ordered")
		}
	}
}

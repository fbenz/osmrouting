// Package pq implements a min-priority queue that grows if needed
//
// Example for two elements:
// 	    pqueue := pq.New(2)
//      x := pq.NewElement("x", 5)
//      y := pq.NewElement("y", 1)
//      heap.Push(&pqueue, x)
//      heap.Push(&pqueue, y)
//      heap.Pop(&pqueue) // returns y
//      heap.Pop(&pqueue) // returns x

package pq

import (
    "container/heap"
)

type Element struct {
    priority int
    // The index is needed by ChangePriority and is maintained by the heap.Interface methods.
    index int // The index of the element in the heap.

    Value interface{}
}

func NewElement(value interface{}, priority int) *Element {
    return &Element{priority, -1, value}
}

func (e *Element) Priority() int {
    return e.priority
}

// A PriorityQueue implements heap.Interface and holds Elements.
type PriorityQueue []*Element

func New(initialCapacity int) PriorityQueue {
    return make(PriorityQueue, 0, initialCapacity)
}

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Empty() bool { return len(pq) == 0 }

// less is built such that a pop returns the element with the lowest priority.
func (pq PriorityQueue) Less(i, j int) bool {
    return pq[i].priority < pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
    pq[i], pq[j] = pq[j], pq[i]
    pq[i].index = i
    pq[j].index = j
}

// Is used by heap.Interface methods and should not be called directly.
func (pq *PriorityQueue) Push(x interface{}) {
    // Push and Pop use pointer receivers because they modify the slice's length,
    // not just its contents.
    a := *pq
    n := len(a)
    element := x.(*Element)
    element.index = n
    a = append(a, element)
    *pq = a
}

// Is used by heap.Interface methods and should not be called directly.
func (pq *PriorityQueue) Pop() interface{} {
    a := *pq
    n := len(a)
    element := a[n-1]
    element.index = -1
    *pq = a[0 : n-1]
    return  element
}

func (pq *PriorityQueue) ChangePriority(element *Element, priority int) {
    heap.Remove(pq, element.index)
    element.priority = priority
    heap.Push(pq, element)
}


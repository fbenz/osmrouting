// Implementation of a Priority Queue with a 4-Heap
//
// The operations on a 4-Heap in contrast to a binary heap are
// first_child(i) = i * 4 + 1
// last_child(i) = i * 4 + 4
// parent(i) = (i - 1) / 4

package route

// Implementation of a Priority Queue with a 4 Heap
type PriorityQueue []*Element

// The Length of the Priority Queue
func Len(pq *PriorityQueue) int { return len(*pq) }

// Is the Priority Queue empty?
func Empty(pq *PriorityQueue) bool { return len(*pq) == 0 }

// Return a new empty Priority Queue
func NewPriorityQueue(initialCapacity int) PriorityQueue {
	return make(PriorityQueue, 0, initialCapacity)
}

// Return a new Priority Queue from an array
func ArrayToPriorityQueue(init *[]*Element) *PriorityQueue {
	a := PriorityQueue(*init)
	for k := len(a) - 1; k >= 0; k-- {
		a[k].index = k
		heapify(&a, k)
	}
	return &a
}

// Restore the Heap propertiy if A[i] larger then children, should never be called from the outside
func heapify(pq *PriorityQueue, index int) {
	smallest := index
	for k := 1; k <= 4; k++ {
		if index*4+k < len(*pq) && (*pq)[smallest].priority > (*pq)[index*4+k].priority {
			smallest = index*4 + k
		}
	}
	if smallest != index {
		(*pq)[smallest], (*pq)[index] = (*pq)[index], (*pq)[smallest]
		(*pq)[smallest].index = smallest
		(*pq)[index].index = index
		heapify(pq, smallest)
	}
}

func shiftup(pq *PriorityQueue, index int) {
	if index == 0 { // Nothing to do here
		return
	} else {
		parent := (index - 1) / 4
		if (*pq)[index].priority < (*pq)[parent].priority {
			(*pq)[index], (*pq)[parent] = (*pq)[parent], (*pq)[index]
			(*pq)[index].index = index
			(*pq)[parent].index = parent
			shiftup(pq, parent)
		} else {
			return
		}
	}
}

func ChangePriority(pq *PriorityQueue, element *Element, priority float64) {
	if element.priority == priority { // Later to be removed
		return
	} else if element.priority < priority {
		element.priority = priority
		heapify(pq, element.index)
	} else {
		element.priority = priority
		shiftup(pq, element.index)
	}
}

func ExtractMin(pq *PriorityQueue) *Element {
	n := len(*pq)
	element := (*pq)[0]
	element.index = -1
	(*pq)[0], (*pq)[n-1] = (*pq)[n-1], (*pq)[0]
	(*pq)[0].index = 0
	(*pq) = (*pq)[0 : n-1]
	heapify(pq, 0)
	return element
}

func PushElement(pq *PriorityQueue, element *Element) {
	n := len(*pq)
	element.index = n
	*pq = append(*pq, element)
	shiftup(pq, n)
}

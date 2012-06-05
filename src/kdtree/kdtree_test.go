// TODO test the new implementation

package kdtree

import (
	"math/rand"
	"testing"
	"time"
)

const (
	DataSetSize = 100000 // more important to the performance
	Repeats     = 10000
)

var nodeData = createData(DataSetSize, false)
var repeatsNodeData = createData(DataSetSize, true)

func createData(size int, repeats bool) NodeDataSlice {
	rnd := rand.New(rand.NewSource(time.Now().Unix()))

	nodeData := make(NodeDataSlice, size)
	for i := 0; i < size; i++ {
		lat, lng := rnd.Float64(), rnd.Float64()
		nodeData[i] = NodeData{lat, lng}

		// insert coordinates that have the same lat/lng (ugly corner case)
		if repeats {
			if rnd.Intn(5) == 0 {
				up := i + 2 + rnd.Intn(100)
				for i++; i < up && i < size; i++ {
					nodeData[i] = NodeData{lat, rnd.Float64()}
				}
				i--
			}
			if rnd.Intn(5) == 0 {
				up := i + 2 + rnd.Intn(100)
				for i++; i < up && i < size; i++ {
					nodeData[i] = NodeData{rnd.Float64(), lng}
				}
				i--
			}
		}
	}
	return nodeData
}

func TestKdTree(t *testing.T) {
	tree := NewKdTree(nodeData)

	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < Repeats; i++ {
		refIndex := rnd.Intn(DataSetSize)
		x := nodeData[refIndex]

		index := tree.Search(x)

		if index != refIndex {
			t.Fatalf("Returned wrong index: expected %v but was %v", refIndex, index)
		}
	}
}

func TestKdTreeRepeats(t *testing.T) {
	tree := NewKdTree(repeatsNodeData)

	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < Repeats; i++ {
		refIndex := rnd.Intn(DataSetSize)
		x := repeatsNodeData[refIndex]

		index := tree.Search(x)

		if index != refIndex {
			t.Fatalf("Returned wrong index: expected %v but was %v", refIndex, index)
		}
	}
}

func BenchmarkCreate(b *testing.B) {
	b.StopTimer()
	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	data := make(NodeDataSlice, b.N)
	for i := 0; i < b.N; i++ {
		lat, lng := rnd.Float64(), rnd.Float64()
		data[i] = NodeData{lat, lng}
	}
	b.StartTimer()

	tree := NewKdTree(data)

	if len(tree.Nodes) != b.N {
		b.Fatalf("Tree not created successfully")
	}
}

func BenchmarkLockups(b *testing.B) {
	b.StopTimer()
	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	tree := NewKdTree(nodeData)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		refIndex := rnd.Intn(DataSetSize)
		x := nodeData[refIndex]
		index := tree.Search(x)
		if index != refIndex {
			b.Fatalf("Returned wrong index: expected %v but was %v", refIndex, index)
		}
	}
}

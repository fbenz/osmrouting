package kdtree

import (
	"geo"
	"graph"
	"math"
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

func createData(size int, repeats bool) []geo.Coordinate {
	rnd := rand.New(rand.NewSource(time.Now().Unix()))

	nodeData := make([]geo.Coordinate, size)
	for i := 0; i < size; i++ {
		lat, lng := rnd.Float64(), rnd.Float64()
		nodeData[i] = geo.Coordinate{lat, lng}

		// insert coordinates that have the same lat/lng (ugly corner case)
		if repeats {
			if rnd.Intn(5) == 0 {
				up := i + 2 + rnd.Intn(100)
				for i++; i < up && i < size; i++ {
					nodeData[i] = geo.Coordinate{lat, rnd.Float64()}
				}
				i--
			}
			if rnd.Intn(5) == 0 {
				up := i + 2 + rnd.Intn(100)
				for i++; i < up && i < size; i++ {
					nodeData[i] = geo.Coordinate{rnd.Float64(), lng}
				}
				i--
			}
		}
	}
	return nodeData
}

func TestEncoding(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	maxNum := int64(math.Pow(2, TotalBits) - 1)
	data := make([]uint64, DataSetSize)
	for i, _ := range data {
		data[i] = uint64(rnd.Int63n(maxNum))
	}

	encodedSteps := make([]uint64, DataSetSize*TypeSize/TotalBits)

	g := make([]graph.Graph, 1)
	coordinates := make([]geo.Coordinate, 0)
	tree := KdTree{Graph: g[0], EncodedSteps: encodedSteps, Coordinates: coordinates}

	for i, _ := range data {
		//if i < 2 {
		tree.SetEncodedStep(i, data[i])
		//}
	}

	for i, _ := range data {
		if tree.EncodedStep(i) != data[i] {
			t.Fatalf("encoding didn't respect identity: expected %v but was %v at position %d\n", data[i], tree.EncodedStep(i), i)
		}
	}
}

func TestAppend(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	maxNum := int64(math.Pow(2, TotalBits) - 1)

	encodedSteps := make([]uint64, 0)
	g := make([]graph.Graph, 1)
	coordinates := make([]geo.Coordinate, 0)
	tree := KdTree{Graph: g[0], EncodedSteps: encodedSteps, Coordinates: coordinates}

	for i := 0; i < DataSetSize; i++ {
		s := uint64(rnd.Int63n(maxNum))
		tree.AppendEncodedStep(s)
		if tree.EncodedStepLen() != i+1 {
			t.Fatalf("wrong length: expected %d but was %d\n", i+1, tree.EncodedStepLen())
		}
		if tree.EncodedStep(i) != s {
			t.Fatalf("wrong content at position %d: expected %d but was %d\n", i, s, tree.EncodedStep(i))
		}
	}
}

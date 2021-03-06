/*
 * Copyright 2014 Florian Benz, Steven Schäfer, Bernhard Schommer
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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

	var g *graph.GraphFile
	tree := KdTree{Graph: g, EncodedSteps: encodedSteps}

	for i, _ := range data {
		tree.SetEncodedStep(i, data[i])
	}

	for i, _ := range data {
		if tree.EncodedStep(i) != data[i] {
			t.Fatalf("encoding didn't respect identity: expected %v but was %v at position %d\n", data[i], tree.EncodedStep(i), i)
		}
	}
}

func TestEncoding2(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	maxNum := int64(math.Pow(2, TotalBits) - 1)
	data := make([]uint64, DataSetSize)
	for i, _ := range data {
		data[i] = uint64(rnd.Int63n(maxNum))
	}

	encodedSteps := make([]uint64, DataSetSize*TypeSize/TotalBits)

	var g *graph.GraphFile
	tree := KdTree{Graph: g, EncodedSteps: encodedSteps}

	for _, i := range rnd.Perm(DataSetSize) {
		tree.SetEncodedStep(i, data[i])
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
	var g *graph.GraphFile
	tree := KdTree{Graph: g, EncodedSteps: encodedSteps}

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

func TestAppend2(t *testing.T) {
	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	maxNum := int64(math.Pow(2, TotalBits) - 1)
	data := make([]uint64, DataSetSize)
	for i, _ := range data {
		data[i] = uint64(rnd.Int63n(maxNum))
	}

	encodedSteps := make([]uint64, 0, DataSetSize*TypeSize/TotalBits)

	var g *graph.GraphFile
	tree := KdTree{Graph: g, EncodedSteps: encodedSteps}

	for i, _ := range data {
		tree.AppendEncodedStep(data[i])
	}

	if tree.EncodedStepLen() != len(data) {
		t.Fatalf("wrong length: expected %d but was %d\n", len(data), tree.EncodedStepLen())
	}

	for i, _ := range data {
		if tree.EncodedStep(i) != data[i] {
			t.Fatalf("encoding didn't respect identity: expected %v but was %v at position %d\n", data[i], tree.EncodedStep(i), i)
		}
	}
}

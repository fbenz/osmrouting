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

package main

import (
	"alg"
	"flag"
	"fmt"
	"geo"
	"log"
	"mm"
	"os"
	"graph"
	"path"
)

var (
	// command line flags
	InputFile    string
	InputCluster string
	ValidateKdTree bool
)

func init() {
	flag.StringVar(&InputFile, "i", "", "input graph file")
	flag.StringVar(&InputCluster, "ic", "", "input graph cluster")
	flag.BoolVar(&ValidateKdTree, "check-kdtree", true, "validate kdtree files")
}

// Helper function to ensure that a coordinate is valid.
func ValidateCoordinate(c geo.Coordinate) {
	if c.Lat < -90 || c.Lat > 90 || c.Lng < -180 || c.Lng > 180 {
		log.Fatalf("Coordinate out of range: %v.", c)
	}
}

// Ensure that the out edges are monotone, start at 0 and end at a sentinel entry
// containing the number of edges. Also ensure that the file size is correct.
// These checks imply that all indexes are valid for the edge array.
func ValidateOutEdges(g *graph.GraphFile) {
	if len(g.FirstOut) != g.VertexCount()+1 {
		log.Fatalf("FirstOut array truncated, len is %v, should be %v.",
			len(g.FirstOut), g.VertexCount()+1)
	}
	
	p := uint32(0)
	for i, j := range g.FirstOut {
		if j < p {
			log.Fatalf("FirstOut array is not monotone at i: %v.\n", i)
		}
		p = j
	}
	
	if g.FirstOut[0] != 0 {
		log.Fatalf("FirstOut array starts at %v, instead of 0.",
			g.FirstOut[0])
	}
	
	if g.FirstOut[len(g.FirstOut)-1] != uint32(g.EdgeCount()) {
		log.Fatalf("FirstOut array ends with %v, instead of %v.",
			g.FirstOut[len(g.FirstOut)-1], g.EdgeCount())
	}
}

// The in edges are stored as a linked list. We have to ensure that
// all edge indices are valid and that every list ends with a cycle
// (in retrospect, this is not a good design, it should end with a
// sentinel...).
// Additionally, we ensure that there is no sharing - it doesn't make
// sense to share edges because of the way EdgeOpposite works.
func ValidateInEdges(g *graph.GraphFile) {
	if len(g.FirstIn) != g.VertexCount() {
		log.Fatalf("FirstIn array truncated, len is %v, should be %v.",
			len(g.FirstIn), g.VertexCount())
	}
	
	if len(g.NextIn) != g.EdgeCount() {
		log.Fatalf("NextIn array truncated, len is %v, should be %v.",
			len(g.NextIn), g.EdgeCount())
	}
	
	visited := make([]byte, (g.EdgeCount() + 7) / 8)
	for _, curr := range g.FirstIn {
		if curr == graph.Sentinel {
			continue
		}
		
		// Traverse the list starting at curr.
		for {
			if curr >= uint32(g.EdgeCount()) {
				log.Fatalf("In Edge %v is out of range, EdgeCount: %v.",
					curr, g.EdgeCount())
			}

			if alg.GetBit(visited, uint(curr)) {
				log.Fatalf("In Edge %v was visited twice.", curr)
			}
			alg.SetBit(visited, uint(curr))
			
			if curr == g.NextIn[curr] {
				break
			}
			curr = g.NextIn[curr]
		}
	}
	
	// Every edge is the in edge of some vertex.
	for i := 0; i < g.EdgeCount(); i++ {
		if !alg.GetBit(visited, uint(i)) {
			log.Fatalf("Missing in edge at index %v.", i)
		}
	}
}

// Ensure that the coordinates are in the correct interval.
func ValidateCoordinates(g *graph.GraphFile) {
	if len(g.Coordinates) != 2 * g.VertexCount() {
		log.Fatalf("Coordinates array truncated, len is %v, should be %v.",
			len(g.Coordinates), 2 * g.VertexCount())
	}
	
	for i := 0; i < g.VertexCount(); i++ {
		lat := g.Coordinates[2 * i]
		lng := g.Coordinates[2 * i + 1]
		ValidateCoordinate(geo.DecodeCoordinate(lat, lng))
	}
}

// Ensure that the attribute arrays are large enough.
func ValidateBitmaps(g *graph.GraphFile) {
	vsize := (g.VertexCount() + 7) / 8
	esize := (g.EdgeCount() + 7) / 8
	arrays := []struct{
		name     string
		expected int
		actual   int
	} {
		{"access car",  vsize, len(g.Access[graph.Car])},
		{"access bike", vsize, len(g.Access[graph.Bike])},
		{"access foot", vsize, len(g.Access[graph.Foot])},
		{"edge access car",  esize, len(g.AccessEdge[graph.Car])},
		{"edge access bike", esize, len(g.AccessEdge[graph.Bike])},
		{"edge access foot", esize, len(g.AccessEdge[graph.Foot])},
		{"oneway",  esize, len(g.Oneway)},
		{"ferries", esize, len(g.Ferries)},
	}
	for _, ary := range arrays {
		if ary.actual != ary.expected {
			log.Fatalf("Bitvector '%v' truncated, len is %v, should be %v.",
				ary.name, ary.actual, ary.expected)
		}
	}
}

// Ensure that first^edge is a valid vertex for any edge in the graph.
func ValidateEdges(g *graph.GraphFile) {
	if len(g.Edges) != g.EdgeCount() {
		log.Fatalf("Edges array truncated, len is %v, should be %v.",
			len(g.Edges), g.EdgeCount())
	}
	
	for i := 0; i < g.VertexCount(); i++ {
		for j := g.FirstOut[i]; j < g.FirstOut[i+1]; j++ {
			u := int(g.Edges[j]) ^ i
			if u < 0 || int(u) >= g.VertexCount() {
				log.Fatalf("Edge target out of range: Edges[%v] = %v (= %v ^ %v).",
					j, g.Edges[j], i, u)
			}
		}
	}
}

// An edge weight may not be 0, +-Inf, or NaN.
func ValidateWeights(g *graph.GraphFile) {
	dist := g.Distances
	if len(dist) != g.EdgeCount() {
		log.Fatalf("Distance array truncated, len is %v, should be %v.",
			len(dist), g.EdgeCount())
	}

	for i := 0; i < g.EdgeCount(); i++ {
		w := dist[i]

		if alg.IsInfHalf(w) {
			log.Fatalf("Edge %v has distance Infinity.", i)
		} else if alg.IsNanHalf(w) {
			log.Fatalf("Edge %v has distance NaN.", i)
		}

		d := alg.HalfToFloat32(w)
		if d <= 0.0 {
			log.Fatalf("Edge %v has distance %v <= 0.", i, d)
		}
	}
	
	// Show a histogram with the max speed values.
	//histogram := alg.NewHistogram("max speed")
	//for i := 0; i < g.EdgeCount(); i++ {
	//	w := g.MaxSpeeds[i]
	//	histogram.Add(fmt.Sprintf("%d", w))
	//}
	//histogram.Print()
}

// Check monotonicity for the steps array.
func ValidateSteps(g *graph.GraphFile) {
	if len(g.Steps) != g.EdgeCount()+1 {
		log.Fatalf("Steps array truncated, len is %v, should be %v.",
			len(g.Steps), g.EdgeCount()+1)
	}
	
	p := uint32(0)
	for i, j := range g.Steps {
		if j < p {
			log.Fatalf("Steps array is not monotone at i: %v.\n", i)
		}
		p = j
	}
	
	if g.Steps[0] != 0 {
		log.Fatalf("Steps array starts at %v, instead of 0.",
			g.Steps[0])
	}
	
	if g.Steps[len(g.Steps)-1] != uint32(len(g.StepPositions)) {
		log.Fatalf("Steps array ends with %v, instead of %v.",
			g.Steps[len(g.Steps)-1], len(g.StepPositions))
	}
}

func ValidateGraphFile(g *graph.GraphFile) {
	fmt.Printf(" * Vertex Count: %v\n", g.VertexCount())
	fmt.Printf(" * Edge Count:   %v\n", g.EdgeCount())
	println(" * Validate Out Edges")
	ValidateOutEdges(g)
	println(" * Validate In Edges")
	ValidateInEdges(g)
	println(" * Validate Coordinates")
	ValidateCoordinates(g)
	println(" * Validate Bitmaps")
	ValidateBitmaps(g)
	println(" * Validate Edges")
	ValidateEdges(g)
	println(" * Validate Weights")
	ValidateWeights(g)
	println(" * Validate Steps")
	ValidateSteps(g)
}


func ValidateKdTreeCoordinates(base string) {
	// For historical reasons there is no convenient way to load a kd tree,
	// so we have to do it manually. The format is as follows:
	// - coordinates.ftf: An array of encoded coordinates, we check that the
	//                    range is correct.
	// - kdtree.ftf: An bitarray of encoded points, currently unchecked.
	var coordinates []int32
	err := mm.Open(path.Join(base, "coordinates.ftf"), &coordinates)
	if err != nil {
		log.Fatalf("Could not open coordinates file: %s", err.Error())
	}
	
	if len(coordinates) % 2 != 0 {
		log.Fatalf("Coordinates array truncated, len is %v, should be even.",
			len(coordinates))
	}
	
	for i := 0; i < len(coordinates)/2; i++ {
		lat := coordinates[2*i]
		lng := coordinates[2*i+1]
		ValidateCoordinate(geo.DecodeCoordinate(lat, lng))
	}
}

func ValidateKdTreeBounds(base string) {
	// The bounding boxes are stored as an array of encoded bboxes.
	// Ensure that every box is non-empty and that every coordinate is
	// in range.
	var boxes []int32
	err := mm.Open(path.Join(base, "bboxes.ftf"), &boxes)
	if err != nil {
		log.Fatalf("Could not open bboxes file: %s", err.Error())
	}
	
	if len(boxes) % 4 != 0 {
		log.Fatalf("BBoxes array truncated, len is %v, should be a multiple of 4.",
			len(boxes))
	}
	
	for i := 0; i < len(boxes)/4; i++ {
		box := geo.DecodeBBox(boxes[4*i:4*i+4])
		ValidateCoordinate(box.Min)
		ValidateCoordinate(box.Max)
		if box.Min.Lat >= box.Max.Lat || box.Min.Lng >= box.Max.Lng {
			log.Fatalf("Empty BBox found: %v.", box)
		}
	}
}

func main() {
	flag.Parse()
	if InputCluster == "" && InputFile == "" {
		flag.Usage()
		os.Exit(1)
	}
	
	if InputCluster != "" {
		println("Open cluster graph.")
		h, err := graph.OpenClusterGraph(InputCluster, false)
		if err != nil {
			println(err.Error())
			os.Exit(1)
		}
		
		println("Validate Cluster")
		for i, g := range h.Cluster {
			fmt.Printf("Cluster %v/%v\n", i+1, len(h.Cluster))
			ValidateGraphFile(g)
			if ValidateKdTree {
				println(" * Validate KdTree")
				base := path.Join(InputCluster, fmt.Sprintf("cluster%v", i+1))
				ValidateKdTreeCoordinates(base)
			}
		}
		println("Validate Overlay")
		ValidateGraphFile(h.Overlay.GraphFile)
		println(" * Validate KdTree")
		ValidateKdTreeCoordinates(path.Join(InputCluster, "overlay"))
		println("Validate Bounding Boxes")
		ValidateKdTreeBounds(InputCluster)
	} else {
		println("Open graph.")
		g, err := graph.OpenGraphFile(InputFile, false)
		if err != nil {
			println(err.Error())
			os.Exit(1)
		}
		ValidateGraphFile(g)
		if ValidateKdTree {
			println(" * Validate KdTree")
			ValidateKdTreeCoordinates(InputFile)
		}
	}
}

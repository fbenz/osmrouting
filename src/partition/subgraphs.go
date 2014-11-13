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

// Writes the subgraphs resulting from the partitioning on disc

package main

import (
	"fmt"
	"graph"
	"log"
	"os"
	"path"
	"time"
)

// Concurrent creation of the subgraphs
func (pi *PartitionInfo) createSubgraphs(g *graph.GraphFile, base string) {
	time1 := time.Now()

	ready := make(chan int, MaxThreads)
	for i := 0; i < MaxThreads; i++ {
		go pi.createSubgraphsPartly(ready, g, base, i)
	}
	for i := 0; i < MaxThreads; i++ {
		<-ready
	}

	time2 := time.Now()
	fmt.Printf("Creating subgraphs: %v s\n", time2.Sub(time1).Seconds())
}

// Creates and writes pi.Count / MaxThreads subgraphs
func (pi *PartitionInfo) createSubgraphsPartly(ready chan<- int, g *graph.GraphFile, base string, start int) {
	vertexIndices := make([]int, g.VertexCount())
	for p := start; p < pi.Count; p += MaxThreads {
		// reset, -1 entries are excluded from the subgraph
		for i, _ := range vertexIndices {
			vertexIndices[i] = -1
		}

		// number the border vertices first
		for i, x := range pi.BorderVertices[p] {
			vertexIndices[x] = i
			pi.Table[x] = -1 // so that it is not considered again in the next loop
		}

		// then number all remaining vertices
		subVertexCount := len(pi.BorderVertices[p])
		for i := 0; i < g.VertexCount(); i++ {
			if pi.Table[i] == p { // and not border vertex, due to the -1 in the loop before
				vertexIndices[i] = subVertexCount
				subVertexCount++
			}
		}

		// restore pi.Table, remove the -1 for the border vertices
		for _, x := range pi.BorderVertices[p] {
			pi.Table[x] = p
		}

		// create directory
		dir := path.Join(base, fmt.Sprintf("/cluster%d", p+1))
		err := os.Mkdir(dir, os.ModeDir|os.ModePerm)
		if err != nil {
			log.Fatal("Creating dir for subgraph: ", err)
		}
		// write graph to disk
		err = g.WriteSubgraph(dir, vertexIndices, vertexIndices)
		if err != nil {
			log.Fatal("Writing the subgraph: ", err)
		}
	}

	ready <- 1
}

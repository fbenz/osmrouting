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

package graph

import (
	"fmt"
	"geo"
	"math"
	"mm"
	"path"
	//"sort"
)

type OverlayGraphFile struct {
	*GraphFile
	Cluster          []uint32      // cluster id -> vertex indices
	VertexIndices    []int         // vertex indices -> cluster id
	Matrices         [][][]float32 // transport mode -> metric -> (cluster id, i, j) -> weight
	ClusterEdgeCount int           // combined boundary edge count of the clusters 
	EdgeCounts       []int         // cluster id -> id of first edge inside the cluster
}

// I/O

func computeVertexIndices(g *OverlayGraphFile) {
	g.VertexIndices = make([]int, g.VertexCount())
	for i := 0; i < g.ClusterCount(); i++ {
		for j := g.Cluster[i]; j < g.Cluster[i+1]; j++ {
			g.VertexIndices[j] = i
		}
	}
}

func computeEdgeCounts(g *OverlayGraphFile) {
	g.EdgeCounts = make([]int, g.ClusterCount()+1)
	g.EdgeCounts[0] = g.GraphFile.EdgeCount()
	for i := 0; i < g.ClusterCount(); i++ {
		g.EdgeCounts[i+1] = g.EdgeCounts[i] + g.ClusterSize(i)*g.ClusterSize(i)
	}
}

func loadAllMatrices(g *OverlayGraphFile, base string) error {
	g.Matrices = make([][][]float32, TransportMax)
	for t := 0; t < int(TransportMax); t++ {
		g.Matrices[t] = make([][]float32, MetricMax)
		for m := 0; m < int(MetricMax); m++ {
			var matrixFile []float32
			fileName := fmt.Sprintf("matrices.trans%d.metric%d.ftf", t+1, m+1)
			err := mm.Open(path.Join(base, fileName), &matrixFile)
			if err != nil {
				return err
			}
			g.Matrices[t][m] = matrixFile
		}
	}
	return nil
}

func OpenOverlay(base string, loadMatrices, ignoreErrors bool) (*OverlayGraphFile, error) {
	overlayBaseDir := path.Join(base, "/overlay")
	g, err := OpenGraphFile(overlayBaseDir, ignoreErrors)
	if err != nil && !ignoreErrors {
		return nil, err
	}

	overlay := &OverlayGraphFile{GraphFile: g}
	files := []struct {
		name string
		p    interface{}
	}{
		{"partitions.ftf", &overlay.Cluster},
	}

	for _, file := range files {
		err = mm.Open(path.Join(overlayBaseDir, file.name), file.p)
		if err != nil && !ignoreErrors {
			return nil, err
		}
	}

	computeVertexIndices(overlay)
	computeEdgeCounts(overlay)
	if loadMatrices {
		err = loadAllMatrices(overlay, base)
		if err != nil && !ignoreErrors {
			return nil, err
		}
	}

	for i := 0; i < overlay.ClusterCount(); i++ {
		overlay.ClusterEdgeCount += overlay.ClusterSize(i) * overlay.ClusterSize(i)
	}

	return overlay, nil
}

func CloseOverlay(overlay *OverlayGraphFile) error {
	err := CloseGraphFile(overlay.GraphFile)
	if err != nil {
		return err
	}

	files := []interface{}{
		&overlay.Cluster,
	}

	for _, p := range files {
		err = mm.Close(p)
		if err != nil {
			return err
		}
	}

	return nil
}

// Graph Interface

func (g *OverlayGraphFile) EdgeCount() int {
	// Count edges and matrices...
	return g.GraphFile.EdgeCount() + g.ClusterEdgeCount
}

func (g *OverlayGraphFile) VertexNeighbors(v Vertex, forward bool, t Transport, m Metric, buf []Dart) []Dart {
	result := g.GraphFile.VertexNeighbors(v, forward, t, m, buf)
	// Add the shortcuts
	cluster, indexInCluster := g.VertexCluster(v)
	clusterStart := g.EdgeCounts[cluster] - g.GraphFile.EdgeCount()
	clusterSize := g.ClusterSize(cluster)
	clusterIndex := int(g.Cluster[cluster])
	matrix := g.Matrices[t][m]
	inf := float32(math.Inf(1))

	// ensure that the result slice has sufficient capacity
	writeIndex := len(result)
	if cap(result) >= clusterSize+len(result) {
		result = result[:len(result)+clusterSize]
	} else {
		realloc := make([]Dart, len(result)+clusterSize)
		copy(realloc, result)
		result = realloc
	}

	if forward {
		// out edges
		outEdges := clusterStart + int(indexInCluster)*clusterSize
		for i := 0; i < clusterSize; i++ {
			u := Vertex(clusterIndex)
			w := matrix[outEdges]
			clusterIndex++
			outEdges++

			if i == int(indexInCluster) || w == inf {
				continue
			}
			//result[writeIndex] = Dart{u,w}
			result[writeIndex].Vertex = u
			result[writeIndex].Weight = w
			writeIndex++
			//result = append(result, Dart{u, w})
		}
	} else {
		// in edges
		inEdges := clusterStart + int(indexInCluster)
		for i := 0; i < clusterSize; i++ {
			u := Vertex(clusterIndex)
			w := matrix[inEdges]
			clusterIndex++
			inEdges += clusterSize

			if i == int(indexInCluster) || w == inf {
				continue
			}
			//result[writeIndex] = Dart{u,w}
			result[writeIndex].Vertex = u
			result[writeIndex].Weight = w
			writeIndex++
			//result = append(result, Dart{u, w})
		}
	}
	return result[:writeIndex]
}

func (g *OverlayGraphFile) IsCutEdge(e Edge) bool {
	return int(e) < g.GraphFile.EdgeCount()
}

func (g *OverlayGraphFile) EdgeSteps(e Edge, from Vertex, buf []geo.Coordinate) []geo.Coordinate {
	// Return nil unless the edge is a cross partition edge.
	// In this case, defer to the normal Graph interface.
	if g.IsCutEdge(e) {
		return g.GraphFile.EdgeSteps(e, from, buf)
	}
	return nil
}

func (g *OverlayGraphFile) EdgeWeight(e Edge, t Transport, m Metric) float64 {
	// Return the normal weight if e is a cross partition edge,
	// otherwise return the precomputed weight for t and m.
	if g.IsCutEdge(e) {
		return g.GraphFile.EdgeWeight(e, t, m)
	}
	edgeIndex := int(e) - g.GraphFile.EdgeCount()
	return float64(g.Matrices[t][m][edgeIndex])
}

// Overlay Interface

func (g *OverlayGraphFile) ClusterCount() int {
	return len(g.Cluster) - 1
}

// actually: boundary vertex count
func (g *OverlayGraphFile) ClusterSize(i int) int {
	// cluster id -> number of vertices
	return int(g.Cluster[i+1] - g.Cluster[i])
}

func (g *OverlayGraphFile) VertexCluster(v Vertex) (int, Vertex) {
	// overlay vertex id -> cluster id, cluster vertex id
	i := g.VertexIndices[v]
	return i, v - Vertex(g.Cluster[i])
}

func (g *OverlayGraphFile) ClusterVertex(i int, v Vertex) Vertex {
	// cluster id, cluster vertex id -> overlay vertex id
	return Vertex(g.Cluster[i]) + v
}

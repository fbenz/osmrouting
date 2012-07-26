package graph

import (
	"fmt"
	"geo"
	"mm"
	"path"
)

// TODO metric matrices
type OverlayGraphFile struct {
	*GraphFile
	Cluster       []uint16          // cluster id -> vertex indices
	VertexIndices []int             // vertex indices -> cluster id
	Matrices      [][][][][]float32 // transport mode -> metric -> cluster id -> i -> j -> weight
}

// I/O

func computeVertexIndices(g *OverlayGraphFile) {
	n, c := g.VertexCount(), 0
	g.VertexIndices = make([]int, n)
	for i := 0; i < n; i++ {
		for c+1 < len(g.Cluster) && i >= int(g.Cluster[c+1]) {
			c++
		}
		g.VertexIndices[i] = c
	}
}

func loadAllMatrices(g *OverlayGraphFile, base string) error {
	g.Matrices = make([][][][][]float32, TransportMax)
	for t := 0; t < int(TransportMax); t++ {
		g.Matrices[t] = make([][][][]float32, MetricMax)
		for m := 0; m < int(MetricMax); m++ {
			g.Matrices[t][m] = make([][][]float32, g.VertexCount())

			var matrixFile []float32
			fileName := fmt.Sprintf("matrices.trans%d.metric%d.ftf", t+1, m+1)
			err := mm.Open(path.Join(base, fileName), &matrixFile)
			if err != nil {
				return err
			}

			pos := 0
			for c := 0; c < g.ClusterCount(); c++ {
				clusterSize := g.ClusterSize(c)
				g.Matrices[t][m][c] = make([][]float32, clusterSize)
				for i := 0; i < clusterSize; i++ {
					g.Matrices[t][m][c][i] = make([]float32, clusterSize)
					for j := 0; j < clusterSize; j++ {
						g.Matrices[t][m][c][i][j] = matrixFile[pos]
						pos++
					}
				}
			}

			err = mm.Close(&matrixFile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func OpenOverlay(base string, loadMatrices, ignoreErrors bool) (*OverlayGraphFile, error) {
	g, err := OpenGraphFile(base, ignoreErrors)
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
		err = mm.Open(path.Join(base, file.name), file.p)
		if err != nil && !ignoreErrors {
			return nil, err
		}
	}

	computeVertexIndices(overlay)
	if loadMatrices {
		err = loadAllMatrices(overlay, base)
		if err != nil && !ignoreErrors {
			return nil, err
		}
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
	return g.GraphFile.EdgeCount() // + TODO
}

func (g *OverlayGraphFile) VertexEdges(v Vertex, forward bool, t Transport, buf []Edge) []Edge {
	// Add the cut edges
	result := g.GraphFile.VertexEdges(v, forward, t, buf)
	// Add the precomputed edges.
	return result
}

func (g *OverlayGraphFile) IsCutEdge(e Edge) bool {
	return int(e) < g.GraphFile.EdgeCount()
}

func (g *OverlayGraphFile) EdgeSteps(e Edge, from Vertex) []geo.Coordinate {
	// Return nil unless the edge is a cross partition edge.
	// In this case, defer to the normal Graph interface.
	if g.IsCutEdge(e) {
		return nil
	}
	return g.GraphFile.EdgeSteps(e, from)
}

func (g *OverlayGraphFile) EdgeWeight(e Edge, t Transport, m Metric) float64 {
	// Return the normal weight if e is a cross partition edge,
	// otherwise return the precomputed weight for t and m.
	if g.IsCutEdge(e) {
		return g.GraphFile.EdgeWeight(e, t, m)
	}
	// TODO
	return 0.0
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

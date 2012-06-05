
package alg

import (
    "../graph"
    "testing"
)

type TestEdge struct {
    start uint
    end uint
    weight int
}

func (e TestEdge) Startpoint() uint { return e.start }
func (e TestEdge) Endpoint() uint { return e.end }
func (e TestEdge) Weight() int { return e.weight }

type TestGraph struct {
    edges [][]graph.Edge
}

func NewTestGraph(edges [][]graph.Edge) *TestGraph {
    return &TestGraph{edges}
}

func (g *TestGraph) Outgoing(i uint) []graph.Edge {
    return g.edges[i]
}

func checkPathEntry(t *testing.T, pos int, expected, actual uint) {
    if expected != actual {
        t.Fatalf("Wrong path entry at position %v: expected %v but was %v", pos, expected, actual)
    }
}

func TestDijkstraSimple(t *testing.T) {
    n := 4
    edges := make([][]graph.Edge, n)
    // two paths from 0 to 3
    // first: 0 - 1 - 3, length 7
    // second: 0 - 2 - 3, length 8
    edges[0] = []graph.Edge{TestEdge{0, 1, 3}, TestEdge{0, 2, 2}}
    edges[1] = []graph.Edge{TestEdge{1, 3, 4}}
    edges[2] = []graph.Edge{TestEdge{2, 3, 6}}
    edges[3] = []graph.Edge{}
    g := NewTestGraph(edges)
    
    distance, path := Dijkstra(g, 0, 3)
    
    if distance != 7 {
        t.Fatalf("Wrong distance: expected 7 but was %v", distance)
    }
    
    if path.Len() != 3 {
        t.Fatalf("Wrong path length: expected 3 but was %v", path.Len())
    }
    
    pathSlice := make([]uint, 3)
    for e, i := path.Front(), 0; e != nil; e, i = e.Next(), i+1 {
	    pathSlice[i] = e.Value.(uint)
    }
    checkPathEntry(t, 0, 0, pathSlice[0])
    checkPathEntry(t, 1, 1, pathSlice[1])
    checkPathEntry(t, 2, 3, pathSlice[2])
}


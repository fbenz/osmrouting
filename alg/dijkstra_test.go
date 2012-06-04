
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

func TestDijkstra(t *testing.T) {
    n := 4
    edges := make([][]graph.Edge, n)
    // two paths from 0 to 3
    // first: 0 - 1 - 3, length 7
    // second: 0 - 1 - 3, length 8
    edges[0] = []graph.Edge{TestEdge{0, 1, 3}, TestEdge{0, 2, 2}}
    edges[1] = []graph.Edge{TestEdge{1, 3, 4}}
    edges[2] = []graph.Edge{TestEdge{2, 3, 6}}
    edges[3] = []graph.Edge{}
    g := NewTestGraph(edges)
    
    distance, path := Dijkstra(g, 0, 3)
    
    if distance != 7 {
        t.Fatalf("Wrong distance")
    }
    
    // TODO the test is not working at the moment and has to be finished
    
    for e := path.Front(); e != nil; e = e.Next() {
	    //e.Value
    }
}


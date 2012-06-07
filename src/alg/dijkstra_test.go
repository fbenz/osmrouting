package alg

import (
	"graph"
	"testing"
)

type TestNode struct {
	edges *[]graph.Edge
	lat float64
	lng float64
}

func (n *TestNode) Edges() []graph.Edge {
	return *n.edges
}

func (n *TestNode) LatLng() (float64, float64) {
	return n.lat, n.lng
}

type Edge interface {
	Length() float64
	StartPoint() graph.Node // e.g. via binary search on the node array
	EndPoint() graph.Node
	ReverseEdge() (graph.Edge, bool)
	Steps() []graph.Step
	// Label() string
}

type TestEdge struct {
	length float64
	startPoint graph.Node
	endPoint graph.Node
	reverseEdgeExists bool
	reverseEdge graph.Edge
	steps []graph.Step
}

func (e *TestEdge) Length() float64   		{ return e.length }
func (e *TestEdge) StartPoint() graph.Node  	{ return e.startPoint }
func (e *TestEdge) EndPoint() graph.Node    	{ return e.endPoint }
func (e *TestEdge) ReverseEdge() (graph.Edge, bool) {
	if e.reverseEdgeExists {
		return e.reverseEdge, true
	}
	return e, false
}
func (e *TestEdge) Steps() []graph.Step 		{ return e.steps }

func checkVertex(t *testing.T, pos int, expected, actual graph.Node) {
	if expected != actual {
		t.Fatalf("Wrong vertex at position %v: expected %v but was %v", pos, expected, actual)
	}
}

func checkEdge(t *testing.T, pos int, expected, actual graph.Edge) {
	if expected != actual {
		t.Fatalf("Wrong edge at position %v: expected %v but was %v", pos, expected, actual)
	}
}

func createSteps(x, y float64) []graph.Step {
	return []graph.Step{{x, y}}
}

func checkWays(t *testing.T, expected, actual graph.Way) {
	if expected.Length != actual.Length || expected.Node != actual.Node {
		t.Fatalf("Wrong length or node in way: expected % but was %v", expected, actual)
	}
	if len(expected.Steps) != len(actual.Steps) {
		t.Fatalf("Wrong number of steps in way: expected % but was %v", expected, actual)
	}
	for i := range expected.Steps {
		if expected.Steps[i].Lat != actual.Steps[i].Lat || expected.Steps[i].Lng != actual.Steps[i].Lng {
			t.Fatalf("Wrong step at position %d in way: expected % but was %v", i, expected, actual)
		}
	}
}

func TestDijkstraSimple(t *testing.T) {
	node1 := &TestNode{edges: nil, lat: 0, lng: 0}
	node2 := &TestNode{edges: nil, lat: 0, lng: 0}
	node3 := &TestNode{edges: nil, lat: 0, lng: 0}
	node4 := &TestNode{edges: nil, lat: 0, lng: 0}
	
	// two paths from 1 to 4
	// first: 1 - 2 - 4, length 7
	// second: 1 - 3 - 4, length 8
	edge1_2 := &TestEdge{length: 3, startPoint: node1, endPoint: node2, reverseEdgeExists: false, steps: createSteps(1, 2)}
	edge1_3 := &TestEdge{length: 2, startPoint: node1, endPoint: node3, reverseEdgeExists: false, steps: createSteps(1, 3)}
	edge2_4 := &TestEdge{length: 4, startPoint: node2, endPoint: node4, reverseEdgeExists: false, steps: createSteps(2, 4)}
	edge3_4 := &TestEdge{length: 6, startPoint: node3, endPoint: node4, reverseEdgeExists: false, steps: createSteps(3, 4)}
	
	startWay := graph.Way{Length: 100, Node: node1, Steps: createSteps(0, 1)}
	endWay := graph.Way{Length: 1000, Node: node4, Steps: createSteps(4, 100)}
	
	node1Edges := []graph.Edge{edge1_2, edge1_3}
	node2Edges := []graph.Edge{edge2_4}
	node3Edges := []graph.Edge{edge3_4}
	node4Edges := []graph.Edge{}
	node1.edges = &node1Edges
	node2.edges = &node2Edges
	node3.edges = &node3Edges
	node4.edges = &node4Edges
	
	distance, vertices, edges, startW, endW := Dijkstra([]graph.Way{startWay}, []graph.Way{endWay})
	
	// actual distance 7, but +100 for start and +1000 for end way
	if distance != 1107 {
		t.Fatalf("Wrong distance: expected 1107 but was %v", distance)
	}
	
	if vertices.Len() != 3 {
		t.Fatalf("Wrong number of vertices: expected 3 but was %v", vertices.Len())
	}

	vertexSlice := make([]graph.Node, 3)
	for e, i := vertices.Front(), 0; e != nil; e, i = e.Next(), i+1 {
		vertexSlice[i] = e.Value.(graph.Node)
	}
	checkVertex(t, 0, node1, vertexSlice[0])
	checkVertex(t, 1, node2, vertexSlice[1])
	checkVertex(t, 2, node4, vertexSlice[2])
	
	if edges.Len() != 2 {
		t.Fatalf("Wrong number of edges: expected 2 but was %v", edges.Len())
	}
	
	edgeSlice := make([]graph.Edge, 2)
	for e, i := edges.Front(), 0; e != nil; e, i = e.Next(), i+1 {
		edgeSlice[i] = e.Value.(graph.Edge)
	}
	checkEdge(t, 0, edge1_2, edgeSlice[0])
	checkEdge(t, 1, edge2_4, edgeSlice[1])
	
	checkWays(t, startWay, startW)
	checkWays(t, endWay, endW)
}

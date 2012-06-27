package alg

import (
	"ellipsoid"
	"graph"
	"testing"
)

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
	geo := ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter,
		ellipsoid.Longitude_is_symmetric, ellipsoid.Bearing_is_symmetric)
	vertices := []uint32{0, 2, 3, 3}
	edges := []uint32{1, 2, 3, 3}
	revEdges := []uint32{0, 1, 2, 3} // no reverse edges -> self pointer
	distances := []float64{3.0, 2.0, 4.0, 6.0}
	positions := make([]float64, len(vertices) * 2) // just 0s
	steps := []uint32{0, 1, 2, 3} // one step per edge
	stepPositions := make([]float64, len(positions) * 2) // just 0s

	g := graph.NewGraphFile(geo, vertices, edges, revEdges, distances, positions, steps, stepPositions)
	
	startWay := graph.Way{Length: 100, Node: 0, Steps: createSteps(0, 1)}
	endWay := graph.Way{Length: 1000, Node: 3, Steps: createSteps(4, 100)}
	
	dist, retVertices, retEdges, startW, endW := Dijkstra(g, []graph.Way{startWay}, []graph.Way{endWay})
	
	// actual distance 7, but +100 for start and +1000 for end way
	if dist != 1107 {
		t.Fatalf("Wrong distance: expected 1107 but was %v", distance)
	}
	
	if len(retVertices) != 3 {
		t.Fatalf("Wrong number of vertices: expected 3 but was %v", len(vertices))
	}

	checkVertex(t, 0, 0, retVertices[0])
	checkVertex(t, 1, 1, retVertices[1])
	checkVertex(t, 2, 3, retVertices[2])
	
	if len(retEdges) != 2 {
		t.Fatalf("Wrong number of edges: expected 2 but was %v", len(edges))
	}
	
	checkEdge(t, 0, 0, retEdges[0])
	checkEdge(t, 1, 2, retEdges[1])
	
	checkWays(t, startWay, startW)
	checkWays(t, endWay, endW)
}

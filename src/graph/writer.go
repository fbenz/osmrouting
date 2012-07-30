package graph

import (
	"alg"
	//"fmt"
	"log"
	"mm"
	"path"
)

// It's easy to misuse the general API, so we do some sanity checks here.
// Returns the vertexMap plus the number of nodes in the subgraph.
func validateNodeIndices(g *GraphFile, indices []int) ([]int, int) {
	if len(indices) != g.VertexCount() {
		log.Fatalf("Wrong number of node indices (is: %v, should be: %v).",
			len(indices), g.VertexCount())
	}
	
	// Determine the maximal node index.
	maxIndex := -1
	for _, i := range indices {
		if i > maxIndex {
			maxIndex = i
		}
	}
	
	// This is technically not an error, but points to a bug somewhere else.
	if maxIndex == -1 {
		log.Fatalf("Attempting to store an empty subgraph.")
	}
	
	// Ensure that the indices are consecutive.
	present := make([]bool, maxIndex + 1)
	for _, i := range indices {
		if i != -1 {
			if present[i] {
				log.Fatalf("Vertex %v is mapped twice.", i)
			}
			present[i] = true
		}
	}
	for i, t := range present {
		if !t {
			log.Fatalf("Vertex %v is not mapped.", i)
		}
	}
	
	vertices := make([]int, maxIndex+1)
	for old, i := range indices {
		if i != -1 {
			vertices[i] = old
		}
	}
	return vertices, maxIndex + 1
}

// Map the edge indices in the original graph to a new set of indices.
// Returns the mapping, containing -1 if the edge is not in the subgraph,
// and the number of edges in the subgraph.
func mapEdges(g *GraphFile, indices, partition, vertexMap []int) ([]int, int) {
	edgeCount := 0
	edgeMap := make([]int, g.EdgeCount())
	for i := range edgeMap {
		edgeMap[i] = -1
	}
	
	for _, u := range vertexMap {
		for e := g.FirstOut[u]; e < g.FirstOut[u+1]; e++ {
			v := g.EdgeOpposite(Edge(e), Vertex(u))
			if indices[v] == -1 || partition[u] == partition[v] {
				continue
			}
			edgeMap[e] = edgeCount
			edgeCount++
		}
	}
	
	return edgeMap, edgeCount
}

// Returns a mapping for the step indices and the size of the new step_positions file.
func mapSteps(g *GraphFile, edgeIndices []int) int {
	size := 0
	for e := 0; e < g.EdgeCount(); e++ {
		if edgeIndices[e] == -1 {
			continue
		}
		size += int(g.Steps[e+1] - g.Steps[e])
	}
	return size
}

func createGraphFile(base string, vertexCount, edgeCount, stepSize int) (*GraphFile, error) {
	g := &GraphFile{}
	vertexBits := (vertexCount + 7) / 8
	edgeBits := (edgeCount + 7) / 8
	files := []struct{name string; size int; p interface{}} {
		{"vertices.ftf",       vertexCount+1, &g.FirstOut},
		{"vertices-in.ftf",    vertexCount,   &g.FirstIn},
		{"positions.ftf",      2*vertexCount, &g.Coordinates},
		{"vaccess-car.ftf",    vertexBits,    &g.Access[Car]},
		{"vaccess-bike.ftf",   vertexBits,    &g.Access[Bike]},
		{"vaccess-foot.ftf",   vertexBits,    &g.Access[Foot]},
		{"access-car.ftf",     edgeBits,      &g.AccessEdge[Car]},
		{"access-bike.ftf",    edgeBits,      &g.AccessEdge[Bike]},
		{"access-foot.ftf",    edgeBits,      &g.AccessEdge[Foot]},
		{"oneway.ftf",         edgeBits,      &g.Oneway},
		{"edges-next.ftf",     edgeCount,     &g.NextIn},
		{"edges.ftf",          edgeCount,     &g.Edges},
		{"distances.ftf",      edgeCount,     &g.Distances},
		{"steps.ftf",          edgeCount+1,   &g.Steps},
		{"step_positions.ftf", stepSize,      &g.StepPositions},
		{"ferries.ftf",        edgeBits,      &g.Ferries},
		{"maxspeeds.ftf",      edgeCount,     &g.MaxSpeeds},
	}
	
	for _, file := range files {
		name := path.Join(base, file.name)
		err := mm.Create(name, file.size, file.p)
		if err != nil {
			return nil, err
		}
	}
	
	return g, nil
}

func writeVertexFirstOut(input, output *GraphFile, vertexMap, edgeIndices []int) {
	// Compute the out degrees.
	for i, u := range vertexMap {
		degree := 0
		for e := input.FirstOut[u]; e < input.FirstOut[u+1]; e++ {
			if edgeIndices[e] != -1 {
				degree++
			}
		}
		output.FirstOut[i] = uint32(degree)
	}
	
	// Store the partial sums, plus sentinel.
	current := 0
	for i := 0; i <= len(vertexMap); i++ {
		degree := output.FirstOut[i]
		output.FirstOut[i] = uint32(current)
		current += int(degree)
	}
}

func writeVertexAttributes(input, output *GraphFile, vertexIndices, edgeIndices []int) {
	for u := 0; u < input.VertexCount(); u++ {
		a := vertexIndices[u]
		if a == -1 {
			continue
		}
		
		// The first in edge is similar, but there is an additional special case:
		if input.FirstIn[u] == Sentinel {
			output.FirstIn[a] = Sentinel
		} else {
			e := input.FirstIn[u]
			for {
				if edgeIndices[e] != -1 {
					output.FirstIn[a] = uint32(edgeIndices[e])
					break
				}
				if e == input.NextIn[e] {
					output.FirstIn[a] = Sentinel
					break
				}
				e = input.NextIn[e]
			}
		}
		
		// Finally, the coordinates and access flags are easy:
		output.Coordinates[2 * a] = input.Coordinates[2 * u]
		output.Coordinates[2 * a + 1] = input.Coordinates[2 * u + 1]
		for t := 0; t < int(TransportMax); t++ {
			if alg.GetBit(input.Access[t], uint(u)) {
				alg.SetBit(output.Access[t], uint(a))
			}
		}
	}
	
	output.FirstOut[len(output.FirstOut)-1] = uint32(len(output.Edges))
}

func writeEdgeAttributes(input, output *GraphFile, edgeIndices []int) {
	for e := 0; e < input.EdgeCount(); e++ {
		f := edgeIndices[e]
		if f == -1 {
			continue
		}
		
		// Next edge... have to traverse the list looking for an edge in the subgraph.
		next := input.NextIn[e]
		if next == Sentinel {
			// shouldn't happen, write a cycle instead.
			output.NextIn[f] = uint32(f)
		} else {
			for {
				if edgeIndices[next] != -1 {
					output.NextIn[f] = uint32(edgeIndices[next])
					break
				}
				if next == input.NextIn[next] {
					output.NextIn[f] = uint32(f)
					break
				}
				next = input.NextIn[next]
			}
		}
		
		// Edge flags
		if alg.GetBit(input.Oneway, uint(e)) {
			alg.SetBit(output.Oneway, uint(f))
		}
		for t := 0; t < int(TransportMax); t++ {
			if alg.GetBit(input.AccessEdge[t], uint(e)) {
				alg.SetBit(output.AccessEdge[t], uint(f))
			}
		}
		if alg.GetBit(input.Ferries, uint(e)) {
			alg.SetBit(output.Ferries, uint(f))
		}
		
		// Distances
		output.Distances[f] = input.Distances[e]
		output.MaxSpeeds[f] = input.MaxSpeeds[f]
	}
}

func writeEdges(input, output *GraphFile, vertexIndices, edgeIndices []int) {
	for u := 0; u < input.VertexCount(); u++ {
		if vertexIndices[u] == -1 {
			continue
		}
		for e := input.FirstOut[u]; e < input.FirstOut[u+1]; e++ {
			f := edgeIndices[e]
			if f == -1 {
				continue
			}
			
			v := input.EdgeOpposite(Edge(e), Vertex(u))
			if vertexIndices[v] == -1 {
				log.Fatalf("Missing edge endpoint.")
			}
			output.Edges[f] = uint32(vertexIndices[u] ^ vertexIndices[v])
		}
	}
}

func writeEdgeSteps(input, output *GraphFile, edgeIndices []int) {
	// Compute the step sizes...
	for e := 0; e < input.EdgeCount(); e++ {
		if edgeIndices[e] == -1 {
			continue
		}
		output.Steps[edgeIndices[e]] = input.Steps[e+1] - input.Steps[e]
	}
	
	// and store the partial sums.
	stepSize := uint32(0)
	for i := 0; i < len(output.Steps); i++ {
		size := output.Steps[i]
		output.Steps[i] = stepSize
		stepSize += size
	}
	
	// copy the encoded steps (the encoded remains valid, since we do
	// not change the starting vertex).
	for e := 0; e < input.EdgeCount(); e++ {
		if edgeIndices[e] == -1 {
			continue
		}
		first := output.Steps[edgeIndices[e]]
		outStep := output.StepPositions[first:]
		inStep  := input.StepPositions[input.Steps[e]:input.Steps[e+1]]
		copy(outStep, inStep)
	}
}

// Output a subgraph of g to the directory path. A vertex v of g
// becomes the vertex with index indices[v] in the subgraph if
// indices[v] != -1. An edge {u, v} exists in the subgraph if
// u and v are in the subgraph and furthermore partition[u] != partition[v].
func (g *GraphFile) WriteSubgraph(base string, indices, partition []int) error {
	// Extend the mapping to edges and steps and compute the size of the subgraph.
	vertexMap, vertexCount := validateNodeIndices(g, indices)
	edgeIndices, edgeCount := mapEdges(g, indices, partition, vertexMap)
	stepCount := mapSteps(g, edgeIndices)
	
	// Create the new graph file.
	out, err := createGraphFile(base, vertexCount, edgeCount, stepCount)
	if err != nil {
		return err
	}
	
	writeVertexFirstOut(g, out, vertexMap, edgeIndices)
	writeVertexAttributes(g, out, indices, edgeIndices)
	writeEdgeAttributes(g, out, edgeIndices)
	writeEdgeSteps(g, out, edgeIndices)
	writeEdges(g, out, indices, edgeIndices)
	
	return CloseGraphFile(out)
}

// Convenience function which outputs an induced subgraph of g specified
// as a bitvector.
func (g *GraphFile) WriteInducedSubgraph(base string, vertices []byte) error {
	vertexIndices := make([]int, g.VertexCount())
	vertexCount := 0
	for i := 0; i < g.VertexCount(); i++ {
		if alg.GetBit(vertices, uint(i)) {
			vertexIndices[i] = vertexCount
			vertexCount++
		} else {
			vertexIndices[i] = -1
		}
	}
	return g.WriteSubgraph(base, vertexIndices, vertexIndices)
}

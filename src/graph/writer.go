package graph

import (
	"alg"
	"log"
	"mm"
	"path"
)

// It's easy to misuse the general API, so we do some sanity checks here.
// Returns the number of nodes in the subgraph.
func validateNodeIndices(g *GraphFile, indices []int) int {
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
			present[i] = true
		}
	}
	for i, t := range present {
		if !t {
			log.Fatalf("Node %v is not mapped.", i)
		}
	}
	
	return maxIndex + 1
}

// Map the edge indices in the original graph to a new set of indices.
// Returns the mapping, containing -1 if the edge is not in the subgraph,
// and the number of edges in the subgraph.
func mapEdges(g *GraphFile, indices, partition []int) ([]int, int) {
	edgeCount := 0
	edgeMap := make([]int, g.EdgeCount())
	for i := range edgeMap {
		edgeMap[i] = -1
	}
	
	for u := 0; u < g.VertexCount(); u++ {
		if indices[u] == -1 {
			continue
		}
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
func mapSteps(g *GraphFile, edgeIndices []int) ([]int, int) {
	stepSize := 0
	stepIndices := make([]int, g.EdgeCount())
	for e := 0; e < g.EdgeCount(); e++ {
		if edgeIndices[e] == -1 {
			continue
		}
		stepIndices[e] = stepSize
		stepSize += int(g.Steps[e+1] - g.Steps[e]) 
	}
	return stepIndices, stepSize
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
		{"distances.ftf",      edgeCount,     &g.Weights[Distance]},
		{"steps.ftf",          edgeCount+1,   &g.Steps},
		{"step_positions.ftf", stepSize,      &g.StepPositions},
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

func writeVertexAttributes(input, output *GraphFile, vertexIndices, edgeIndices []int) {
	for u := 0; u < input.VertexCount(); u++ {
		a := vertexIndices[u]
		if a == -1 {
			continue
		}
		
		// To find the first out edge of a we need to find the first
		// mapped edge out of u. It's possible that there is no edge
		// from u. Usually, this is not a problem, but we need to handle
		// the case where we run past the end of the edge array.
		for e := input.FirstOut[u]; int(e) <= input.EdgeCount(); e++ {
			if int(e) == input.EdgeCount() {
				output.FirstOut[a] = uint32(output.EdgeCount())
				break
			} else if edgeIndices[e] != -1 {
				output.FirstOut[a] = uint32(edgeIndices[e])
				break
			}
		}
		
		// The first in edge is similar, but there is an additional special case:
		if input.FirstIn[u] == 0xffffffff {
			output.FirstIn[a] = 0xffffffff
		} else {
			e := input.FirstIn[u]
			for {
				if edgeIndices[e] != -1 {
					output.FirstIn[a] = uint32(edgeIndices[e])
					break
				}
				if e == input.NextIn[e] {
					output.FirstIn[a] = 0xffffffff
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

func writeEdgeAttributes(input, output *GraphFile, edgeIndices, stepIndices []int) {
	for e := 0; e < input.EdgeCount(); e++ {
		f := edgeIndices[e]
		if f == -1 {
			continue
		}
		
		// Next edge... have to traverse the list looking for an edge in the subgraph.
		next := input.NextIn[e]
		if next == 0xffffffff {
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
		
		// Data attributes
		output.Weights[Distance][f] = input.Weights[Distance][e]
		step := stepIndices[e]
		if step == -1 {
			log.Fatalf("Unmapped step index.")
		}
		output.Steps[f] = uint32(step)
		
		// Step positions
		outStep := output.StepPositions[step:]
		inStep  := input.StepPositions[input.Steps[e]:input.Steps[e+1]]
		copy(outStep, inStep)
	}
	
	output.Steps[len(output.Steps)-1] = uint32(len(output.StepPositions))
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

// Output a subgraph of g to the directory path. A vertex v of g
// becomes the vertex with index indices[v] in the subgraph if
// indices[v] != -1. An edge {u, v} exists in the subgraph if
// u and v are in the subgraph and furthermore partition[u] != partition[v].
func (g *GraphFile) WriteSubgraph(base string, indices, partition []int) error {
	// Extend the mapping to edges and steps and compute the size of the subgraph.
	vertexCount := validateNodeIndices(g, indices)
	edgeIndices, edgeCount := mapEdges(g, indices, partition)
	stepIndices, stepCount := mapSteps(g, edgeIndices)
	
	// Create the new graph file.
	out, err := createGraphFile(base, vertexCount, edgeCount, stepCount)
	if err != nil {
		return err
	}
	
	writeVertexAttributes(g, out, indices, edgeIndices)
	writeEdgeAttributes(g, out, edgeIndices, stepIndices)
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

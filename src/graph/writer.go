package graph

// Output a subgraph of g to the directory path. A vertex v of g
// becomes the vertex with index indices[v] in the subgraph if
// indices[v] != -1. An edge {u, v} exists in the subgraph if
// u and v are in the subgraph and furthermore partition[u] != partition[v].
func (g *GraphFile) WriteSubgraph(path string, indices []int, partition []int) error {
	return nil
}

// Convenience function which outputs an induced subgraph of g specified
// as a bitvector.
func (g *GraphFile) WriteInducedSubgraph(path string, vertices []byte) error {
	return nil
}

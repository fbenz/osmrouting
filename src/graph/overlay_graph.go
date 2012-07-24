package graph

// TODO metric matrices
type OverlayGraphFile struct {
	GraphFile *GraphFile
	Cluster   []uint32 // cluster id -> vertex indices
}

// TODO interface implementation for the overlay graph

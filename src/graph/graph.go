package graph

type Node interface {
	Edges() []Edge
	LatLng() (float64, float64)
}

type Edge interface {
	Length() float64
	StartPoint() Node // e.g. via binary search on the node array
	EndPoint() Node
	ReverseEdge() (Edge, bool)
	Steps() []Step
	// Label() string
}

// "partial edge" is returned by the k-d tree
type Way interface {
	Length() float64
	Node() Node // StartPoint or EndPoint
	Steps() []Step
}

type Graph interface {
	NodeCount() int
	EdgeCount() int
	Node(uint) Node
	Edge(uint) Edge
	Positions() Positions
}

// Implementation sketch (wrapper around graph):
// The graph is loaded before and is given to this.
// Init: Both position files (vertexes and inner steps in an edge) are alread loaded
//   for the graph (Nodes.LatLng(), Edge.Steps() - Step.Lat/Lng).
//   Thus despite storing a pointer the graph, nothing to do here. 
// For every method where an index is given, there is a branch
//   if index < Graph.NodeCount()
//      work with Graph.Node(index)
//   else
//      work with the underlying Step array (index - Graph.NodeCount())
//      not possible efficiently with the current interface
//
// Positions is used for both creating the k-d tree in the preprocessing phase
// and for doing the nearest neighbor lookup during runtime.
type Positions interface {
	Len() int
	Lat(int) float64
	Lng(int) float64
	Step(int) Step
	Ways(int, bool) []Way // index, forward (i.e. looking at the edge in forward or in backward order)
}

type Step struct {
	Lat float64
	Lng float64
}

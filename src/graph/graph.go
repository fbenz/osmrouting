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

type Positions interface {
	Len() int
	Lat(int) float64
	Lng(int) float64
	// bit pattern
	// 0 (1 bit) + vetex index
	// 1 (1 bit) + step offset (11 bit) + edge/step index
	Encoding(int) int64
}

type Step struct {
	Lat float64
	Lng float64
}

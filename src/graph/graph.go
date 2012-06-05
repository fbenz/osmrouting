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

type Step struct {
	Lat float64
	Lng float64
}

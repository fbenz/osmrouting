package graph

// Edges are abstract now
type Edge interface{
	Weight() int // Get the weight of an edge currently int
	Startpoint() uint
	Endpoint() uint
}

// Graph interface, defining functions the graph structure should support
type Graph interface{	
	Outgoing(uint) []Edge
}


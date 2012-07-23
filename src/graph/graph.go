
package graph

import "geo"

type Vertex int
type Edge   int

type Transport int
const (
	Car Transport = iota
	Foot
	Bike
	TransportMax
)

type Metric int
const (
	Distance Metric = iota
	MetricMax
)

// Way is a "partial edge" that is returned by the k-d tree
type Way struct {
	Length  float64
	Node    Vertex // StartPoint or EndPoint
	Steps   []geo.Coordinate
	Target  geo.Coordinate
	Forward bool
}

type Graph interface {
	VertexCount() int
	EdgeCount()   int

	VertexAccessible(Vertex, Transport) bool
	VertexCoordinate(Vertex) geo.Coordinate
	VertexEdgeIterator(Vertex, bool, Transport) EdgeIterator

	EdgeOpposite(Edge, Vertex) Vertex
	EdgeSteps(Edge, Vertex)  []geo.Coordinate
	EdgeWeight(Edge, Transport, Metric) float64
}

type EdgeIterator interface {
	Next() (Edge, bool)
}

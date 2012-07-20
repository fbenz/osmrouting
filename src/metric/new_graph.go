// TODO remove
// this is just a dummy file to be able to work with the new interface

package main

import (
	"geo"
)

type Vertex int
type Edge int

type Metric int

const (
	MetricCar = iota
	MetricFoot
	MetricBike
	MetricMax
)

type Graph interface {
	VertexCount() int
	EdgeCount() int

	VertexCoordinate(Vertex) geo.Coordinate
	VertexEdgeIterator(Vertex, bool, Metric) EdgeIterator
	VertexReachable(Vertex, Metric) bool

	EdgeOpposite(Edge, Vertex) Vertex
	EdgeSteps(Edge, Vertex) []geo.Coordinate
	EdgeWeight(Edge, Metric) float64
}

type OverlayGraph interface {
	Graph

	PartitionSize(int) int
}

type EdgeIterator interface {
	Next() (Edge, bool)
}

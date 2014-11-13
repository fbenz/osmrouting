/*
 * Copyright 2014 Florian Benz, Steven Schäfer, Bernhard Schommer
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package graph

import "geo"

type Vertex int
type Edge int

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
	Time
	MetricMax
)

// Way is a "partial edge" that is returned by the k-d tree
type Way struct {
	Length  float64
	Vertex  Vertex // start or end point
	Steps   []geo.Coordinate
	Target  geo.Coordinate
	Forward bool
}

// A half edge + weight, for use by Dijkstra.
type Dart struct {
	Vertex Vertex
	Weight float32
}

type Graph interface {
	VertexCount() int
	EdgeCount() int

	VertexAccessible(Vertex, Transport) bool
	VertexCoordinate(Vertex) geo.Coordinate
	VertexEdges(Vertex, bool, Transport, []Edge) []Edge

	VertexNeighbors(Vertex, bool, Transport, Metric, []Dart) []Dart

	EdgeOpposite(Edge, Vertex) Vertex
	EdgeSteps(Edge, Vertex, []geo.Coordinate) []geo.Coordinate
	EdgeWeight(Edge, Transport, Metric) float64

	// direct access to edge attributes
	EdgeFerry(Edge) bool
	EdgeMaxSpeed(Edge) int
	EdgeOneway(Edge, Transport) bool
}

type OverlayGraph interface {
	Graph

	ClusterCount() int
	ClusterSize(int) int                // cluster id -> number of vertices
	VertexCluster(Vertex) (int, Vertex) // vertex id -> cluster id, cluster vertex id
	ClusterVertex(int, Vertex) Vertex   // cluster id, vertex id -> vertex id
}

func (t Transport) String() string {
	switch t {
	case Car:
		return "TransportCar"
	case Bike:
		return "TransportBike"
	case Foot:
		return "TransportFoot"
	case TransportMax:
		return "TransportMax"
	}
	return "Invalid Transport Enum"
}

func (m Metric) String() string {
	switch m {
	case Distance:
		return "MetricDistance"
	case MetricMax:
		return "MetricMax"
	}
	return "Invalid Metric Enum"
}

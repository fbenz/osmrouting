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

package main

import (
	"parser/pbf"
	"flag"
	"math"
	"os"
	"fmt"
)

var (
	// command line flags
	FlagInput string
	FlagLat   float64
	FlagLon   float64
)

func init() {
	flag.StringVar(&FlagInput, "i", "graph.osm.pbf", "the .osm.pbf file to parse")
	flag.Float64Var(&FlagLat, "lat", 0.0, "latitude of target point")
	flag.Float64Var(&FlagLon, "lon", 0.0, "longitude of target point")
}

func traverseGraph(file *os.File, visitor pbf.Visitor) error {
	_, err := file.Seek(0, 0)
	if err != nil {
		return err
	}

	pbf.VisitGraph(file, visitor)
	return nil
}

type StabbingQuery struct {
	Lat float64
	Lon float64
	Node *pbf.Node
	Ways []pbf.Way
	Nodes map[int64] pbf.Node
}

func distance(lat1, lon1, lat2, lon2 float64) float64 {
	dlat := math.Abs(lat1 - lat2)
	dlon := math.Abs(lon1 - lon2)
	return dlat + dlon
}

func (q *StabbingQuery) VisitNode(node pbf.Node) {
	// Memorize all nodes, since we might need them for the output.
	q.Nodes[node.Id] = node
	// Try to find the node closest to the query location.
	if q.Node == nil {
		q.Node = &node
		q.Ways = []pbf.Way {}
	} else {
		minDist := distance(q.Lat, q.Lon, q.Node.Lat, q.Node.Lon)
		curDist := distance(q.Lat, q.Lon, node.Lat, node.Lon)
		if curDist < minDist {
			q.Node = &node
			q.Ways = []pbf.Way {}
		}
	}
}

func (q *StabbingQuery) VisitWay(way pbf.Way) {
	if q.Node != nil {
		for _, ref := range way.Nodes {
			if ref == q.Node.Id {
				q.Ways = append(q.Ways, way)
				return
			}
		}
	}
}

func main() {
	flag.Parse()
	
	// Open the input file
	file, err := os.Open(FlagInput)
	if err != nil {
		println("Unable to open file:", err.Error())
		os.Exit(1)
	}
	
	fmt.Printf("Query for node at position (%.7f, %.7f)\n", FlagLat, FlagLon)
	
	// Perform the query
	query := &StabbingQuery{
		Lat: FlagLat,
		Lon: FlagLon,
		Node: nil,
		Ways: nil,
		Nodes: map[int64] pbf.Node {},
	}
	err = traverseGraph(file, query)
	if err != nil {
		println("Error parsing file:", err.Error())
		os.Exit(2)
	}
	
	if query.Node == nil {
		println("Did not find any node.")
		os.Exit(3)
	}
	
	// Output the results
	id  := query.Node.Id
	lat := query.Node.Lat
	lon := query.Node.Lon
	fmt.Printf(" - Closest Match: (%.7f, %.7f)\n", lat, lon)
	fmt.Printf(" - OSM node id: %d\n", id)
	fmt.Printf(" - Contained in %d Ways:\n", len(query.Ways))
	for i, way := range query.Ways {
		fmt.Printf("  - Way %d\n", i)
		for j, nodeId := range way.Nodes {
			node := query.Nodes[nodeId]
			id  := node.Id
			lat := node.Lat
			lon := node.Lon
			fmt.Printf("    [%d] id: %d, pos: (%.7f, %.7f)\n", j, id, lat, lon)
		}
		fmt.Printf("   Attributes:\n")
		for key, value := range way.Attributes {
			fmt.Printf("    [%s] %s\n", key, value)
		}
	}
}

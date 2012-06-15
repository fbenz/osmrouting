package main

import (
	"parser/pbf"
	"flag"
	"os"
	"fmt"
)

var (
	// command line flags
	FlagInput string
)

func init() {
	flag.StringVar(&FlagInput, "i", "graph.osm.pbf", "the .osm.pbf file to parse")
}

// Histogram handling
type Histogram struct {
	Name string
	Data map[string] int
	Exceptions int
}

func NewHistogram(name string) *Histogram {
	return &Histogram{
		Name: name,
		Data: map[string] int {},
		Exceptions: 0,
	}
}

func (h *Histogram) AddFail() {
	h.Exceptions++
}

func (h *Histogram) Add(value string) {
	if _, ok := h.Data[value]; ok {
		h.Data[value]++
	} else {
		h.Data[value] = 1
	}
}

func (h *Histogram) Print() {
	fmt.Printf("\n")
	fmt.Printf("Histogram for %s:\n", h.Name)
	total := 0
	for _, frequency := range h.Data {
		total += frequency
	}
	fmt.Printf(" - Total: %d\n", total)
	fmt.Printf(" - Exceptions: %d\n", h.Exceptions)
	fmt.Printf("=========================\n")
	for key, frequency := range h.Data {
		fmt.Printf(" %16s: %d\n", key, frequency)
	}
}

func traverseGraph(file *os.File, visitor pbf.Visitor) error {
	_, err := file.Seek(0, 0)
	if err != nil {
		return err
	}

	pbf.VisitGraph(file, visitor)
	return nil
}

type StatisticsQuery struct {
	Oneways          *Histogram
	Highways         *Histogram
	Junctions        *Histogram
	JunctionOneways  *Histogram
	JunctionHighways *Histogram
	RedundantJunctions int
	TotalJunctions     int
	// Counters
	NodeCount uint64
	WayCount  uint64
}

func ParseBool(b string) bool {
	switch b {
	case "true", "1", "yes", "-1":
		return true
	case "false", "0", "no":
		return false
	}
	fmt.Printf("Unrecognized boolean in ParseBool: %s\n", b)
	return false
}

func (q *StatisticsQuery) VisitNode(node pbf.Node) {
	q.NodeCount++
}

func (q *StatisticsQuery) VisitWay(way pbf.Way) {
	q.WayCount++
	
	// Gather statistics about way#oneway
	if value, ok := way.Attributes["oneway"]; ok {
		q.Oneways.Add(value)
		if way.Attributes["highway"] == "junction" {
			q.RedundantJunctions++
			q.TotalJunctions++
		}
	} else if way.Attributes["highway"] == "junction" {
		q.TotalJunctions++
	}
	// record all the highway tags we see
	if value, ok := way.Attributes["highway"]; ok {
		q.Highways.Add(value)
	}
	// and the same for junctions
	if value, ok := way.Attributes["junction"]; ok {
		q.Junctions.Add(value)
		// Oneways attributes seem to be mostly wrong
		if b, ok := way.Attributes["oneway"]; ok {
			q.JunctionOneways.Add(b)
		} else {
			q.JunctionOneways.AddFail()
		}
		// Junctions should be highways
		if value, ok := way.Attributes["highway"]; ok {
			q.JunctionHighways.Add(value)
		} else {
			q.JunctionHighways.AddFail()
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
	
	fmt.Printf("Parsing %s\n", FlagInput)
	
	query := &StatisticsQuery{
		Oneways: NewHistogram("Oneways"),
		Highways: NewHistogram("Highways"),
		Junctions: NewHistogram("Junctions"),
		JunctionOneways: NewHistogram("Junction-Oneways"),
		JunctionHighways: NewHistogram("Junction-Highways"),
		RedundantJunctions: 0,
		TotalJunctions: 0,
		NodeCount: 0,
		WayCount: 0,
	}
	err = traverseGraph(file, query)
	if err != nil {
		println("Error parsing file:", err.Error())
		os.Exit(2)
	}
	
	fmt.Printf("Parsed %d Nodes, %d Ways\n", query.NodeCount, query.WayCount)
	
	query.Oneways.Print()
	query.Highways.Print()
	query.Junctions.Print()
	query.JunctionOneways.Print()
	query.JunctionHighways.Print()
	
	fmt.Printf("\n")
	fmt.Printf("Number of highway junctions: %d\n", query.TotalJunctions)
	fmt.Printf("Number of missing oneway annotations: %d\n", query.TotalJunctions - query.RedundantJunctions)
}

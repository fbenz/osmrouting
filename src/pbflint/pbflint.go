package main

import (
	"alg/histogram"
	"encoding/json"
	"flag"
	"os"
	"osm"
	"fmt"
	"sort"
)

var (
	// command line flags
	FlagInput string
)

func init() {
	flag.StringVar(&FlagInput, "i", "", "the .osm.pbf file to parse")
}

type StatisticsQuery struct {
	Oneways          *alg.Histogram
	Highways         *alg.Histogram
	Junctions        *alg.Histogram
	JunctionOneways  *alg.Histogram
	JunctionHighways *alg.Histogram
	// Counters
	NodeCount uint64
	WayCount  uint64
	RelCount  uint64
}

func (q *StatisticsQuery) VisitNode(node osm.Node) {
	q.NodeCount++
}

func (q *StatisticsQuery) VisitWay(way osm.Way) {
	q.WayCount++
	
	// Gather statistics about way#oneway
	if value, ok := way.Attributes["oneway"]; ok {
		q.Oneways.Add(value)
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

func (q *StatisticsQuery) VisitRelation(rel osm.Relation) {
	q.RelCount++
}

type AccessQuery struct {
	Highways map[string] osm.Way
	Access   map[string] osm.Way
	// Counters
	NodeCount      uint64
	WayCount       uint64
	RelationCount  uint64
}

func (q *AccessQuery) VisitNode(_ osm.Node) {
	q.NodeCount++
}

func (q *AccessQuery) VisitRelation(_ osm.Relation) {
	q.RelationCount++
}

func accessMark(way osm.Way) bool {
	for _, t := range osm.AccessTable {
		if _, ok := way.Attributes[t.Key]; ok {
			return true
		}
	}
	return osm.DefaultAccessMask(way) != 0
}

func (q *AccessQuery) VisitWay(w osm.Way) {
	q.WayCount++
	// Save examples for all highway types
	if hw, ok := w.Attributes["highway"]; ok {
		if _, ok = q.Highways[hw]; !ok {
			q.Highways[hw] = w
			return
		}
	}
	// Also save examples for all known access types
	for _, t := range osm.AccessTable {
		if _, ok := w.Attributes[t.Key]; ok {
			if _, ok = q.Access[t.Key]; !ok {
				q.Access[t.Key] = w
				return
			}
		}
	}
}

type WayRestriction struct {
	Way    osm.Way
	Access map[string] bool
}

func encodeAccess(w osm.Way) map[string] bool {
	mask := osm.AccessMask(w)
	r := map[string] bool {}
	
	if mask & osm.AccessMotorcar != 0 {
		r["motorcar"] = true
	} else {
		r["motorcar"] = false
	}
	
	if mask & osm.AccessBicycle != 0 {
		r["bicycle"] = true
	} else {
		r["bicycle"] = false
	}
	
	if mask & osm.AccessFoot != 0 {
		r["foot"] = true
	} else {
		r["foot"] = false
	}
	
	return r
}

func main() {
	flag.Parse()
	if FlagInput == "" {
		flag.Usage()
		os.Exit(1)
	}
	
	// Open the input file
	file, err := os.Open(FlagInput)
	if err != nil {
		println("Unable to open file:", err.Error())
		os.Exit(1)
	}
	
	fmt.Printf("Parsing %s\n", FlagInput)
	
	/*
	query := &StatisticsQuery{
		Oneways: alg.NewHistogram("Oneways"),
		Highways: alg.NewHistogram("Highways"),
		Junctions: alg.NewHistogram("Junctions"),
		JunctionOneways: alg.NewHistogram("Junction-Oneways"),
		JunctionHighways: alg.NewHistogram("Junction-Highways"),
		NodeCount: 0,
		WayCount: 0,
		RelCount: 0,
	}
	
	err = osm.ParseFile(file, query)
	if err != nil {
		println("Error parsing file:", err.Error())
		os.Exit(2)
	}
	
	fmt.Printf("Parsed %d Nodes, %d Ways, %d Relations\n", query.NodeCount, query.WayCount, query.RelCount)
	
	query.Oneways.Print()
	query.Highways.Print()
	query.Junctions.Print()
	query.JunctionOneways.Print()
	query.JunctionHighways.Print()
	*/
	
	query := &AccessQuery{
		Highways: map[string] osm.Way {},
		Access: map[string] osm.Way {},
	}
	
	err = osm.ParseFile(file, query)
	if err != nil {
		println("Error parsing file:", err.Error())
		os.Exit(2)
	}
	
	fmt.Printf(" %d Nodes\n", query.NodeCount)
	fmt.Printf(" %d Ways\n", query.WayCount)
	fmt.Printf(" %d Relations\n", query.RelationCount)
	
	ways := make([]osm.Way, 0)
	for _, v := range query.Highways {
		ways = append(ways, v)
	}
	for _, v := range query.Access {
		ways = append(ways, v)
	}
	
	out, err := os.Create("out.txt")
	for _, way := range ways {
		restriction := WayRestriction{
			Way: way,
			Access: encodeAccess(way),
		}
		b, err := json.MarshalIndent(restriction, "", "  ")
		if err != nil {
			println("Error marshalling result:", err.Error())
			os.Exit(3)
		}
		out.Write(b)
	}
	out.Close()
}

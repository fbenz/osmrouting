package pbf

import "io"

type RoutingFilter struct {
	client Visitor
}

func (f RoutingFilter) VisitNode(node Node) {
	f.client.VisitNode(node)
}

func reverse(nodes []int64) {
	for i, j := 0, len(nodes)-1; i < j; i, j = i+1, j-1 {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
}

func normalizeOneway(way Way) {
	// First normalize the allowed booleans and take care of the
	// nasty -1 case...
	switch way.Attributes["oneway"] {
	case "yes", "true", "1":
		way.Attributes["oneway"] = "true"
		return
	case "-1":
		reverse(way.Nodes)
		way.Attributes["oneway"] = "true"
		return
	}

	// Secondly, there are a few special cases which imply 'oneway'
	if way.Attributes["junction"] == "roundabout" {
		way.Attributes["oneway"] = "true"
		return
	} else if way.Attributes["highway"] == "motorway" {
		way.Attributes["oneway"] = "true"
		return
	} else if way.Attributes["highway"] == "motorway_link" {
		way.Attributes["oneway"] = "true"
		return
	}

	// Finally... there are some cases which are just wrong.
	/*
		switch {
		case way.Attributes["oneway"] != "no",
			 way.Attributes["oneway"] != "0",
			 way.Attributes["oneway"] != "false":
			way.Attributes["oneway"] = "true"
		}
	*/
}

func (f RoutingFilter) VisitWay(way Way) {
	/*hw*/_, ok1 := way.Attributes["highway"]
	_,  ok2 := way.Attributes["junction"]
	if !ok1 && !ok2 {
		return
	}
	
	/*
	if ok1 {
		// There are some highway types which are only
		// accessible to pedestrians
		switch hw {
		case "pedestrian", "path", "bridleway", "cycleway", "footway":
			return
		}
	}
	*/
	
	normalizeOneway(way)
	f.client.VisitWay(way)
}

func VisitRoutes(stream io.Reader, client Visitor) error {
	visitor := RoutingFilter{client}
	return VisitGraph(stream, visitor)
}

package osm

func reverse(nodes []int64) {
	for i, j := 0, len(nodes)-1; i < j; i, j = i+1, j-1 {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
}

// There are a number of exceptions when it comes to oneway tags.
// After calling this function oneway is either true or false and
// the nodes are stored in the correct order.
// It returns false if we could not parse the oneway tag. In this
// case it is probably a good idea to ignore the street, since it
// might or might not be a one-way road and the direction might be
// wrong.
func NormalizeOneway(way Way) bool {
	// First normalize the allowed booleans and take care of the
	// nasty -1 (reversed) case.
	switch way.Attributes["oneway"] {
	case "yes", "true", "1":
		way.Attributes["oneway"] = "true"
		return true
	case "-1":
		reverse(way.Nodes)
		way.Attributes["oneway"] = "true"
		return true
	case "no", "false", "0":
		way.Attributes["oneway"] = "false"
		return true
	}

	// Secondly, there are a few special cases which imply 'oneway'
	if way.Attributes["junction"] == "roundabout" {
		way.Attributes["oneway"] = "true"
		return true
	}

	switch way.Attributes["highway"] {
	case "motorway", "motorway_link", "trunk":
		way.Attributes["oneway"] = "true"
		return true
	}

	// Finally... there are some cases which are just wrong.
	if _, ok := way.Attributes["oneway"]; ok {
		return false
	}
	return true
}

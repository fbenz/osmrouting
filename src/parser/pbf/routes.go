/*
 * Filter to ignore irrelevant data, like house outlines. Unfortunately, we
 * cannot easily filter out nodes, but parser 2.0 will improve this situation.
 * For now, we can filter roots that are accessible to pedestrians, cars or
 * bikes. The rules implemented here are by the way specific for Germany.
 * It's going to be a major pain to implement country specific rules in the future.
 */

package pbf

//import "fmt"

// The different access types.
type AccessType int
const (
	AccessMotorcar AccessType = 1 << 0
	AccessBicycle 			  = 1 << 1
	AccessFoot    			  = 1 << 2
)

// The osm access hierarchy.
type AccessData struct {
	Key  string
	Mask AccessType
}
var AccessTable = [...]AccessData {
	{ "access",        AccessMotorcar | AccessBicycle | AccessFoot },
	{ "foot",          AccessFoot },
	{ "vehicle",       AccessMotorcar | AccessBicycle },
	{ "bicycle",       AccessBicycle },
	{ "motor_vehicle", AccessMotorcar },
	{ "motorcar",      AccessMotorcar },
}

type RoutingFilter struct {
	client Visitor
	access AccessType
}

func (f RoutingFilter) VisitNode(node Node) {
	f.client.VisitNode(node)
}

func reverse(nodes []int64) {
	for i, j := 0, len(nodes)-1; i < j; i, j = i+1, j-1 {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
}

// Handle all the myriad exceptions and rules for the oneway tag.
// After calling this function oneway is either true or false and
// the nodes are stored in the correct order.
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
	case "no", "false", "0":
		way.Attributes["oneway"] = "false"
		return
	}

	// Secondly, there are a few special cases which imply 'oneway'
	if way.Attributes["junction"] == "roundabout" {
		way.Attributes["oneway"] = "true"
		return
	}
	
	switch way.Attributes["highway"] {
	case "motorway", "motorway_link", "trunk":
		way.Attributes["oneway"] = "true"
		return
	}

	// Finally... there are some cases which are just wrong.
	// TODO: We should probably just ignore these cases and act as if this
	// was not a road at all. After all, if we are not sure whether a road
	// is oneway only, and moreover in which direction it goes we cannot
	// use it safely.
	if _, ok := way.Attributes["oneway"]; ok {
		way.Attributes["oneway"] = "true"
	}
}

// Parse a boolean attribute.
func parseBoolean(way Way, key string) bool {
	// If the key is not present at all we interpret this as false.
	value, ok := way.Attributes[key]
	if !ok {
		return false
	}
	
	// Otherwise there are some values which we interpret as true,
	// everything else is false. Notice that this is often wrong,
	// but the right way to parse this is situation specific.
	switch value {
	case "yes", "true", "1", "designated":
		return true
	}
	// TODO: add more special cases?
	return false
}

// Compute the access mask based on the default access restrictions for Germany.
func defaultAccessMask(way Way) AccessType {
	// There is a special exceptional tag for motorroads which are not.
	if parseBoolean(way, "motorroad") {
		return AccessMotorcar
	}
	
	// Highway defaults
	switch way.Attributes["highway"] {
	// These roads are generally accessible
	case "trunk", "primary", "secondary",
		 "tertiary", "unclassified", "residential",
		 "living_street", "road":
		return AccessMotorcar | AccessBicycle | AccessFoot
	// The rest excludes some access types
	case "motorway":
		return AccessMotorcar
	case "path", "track":
		return AccessBicycle | AccessFoot
	case "footway", "pedestrian", "stairs", "service":
		return AccessFoot
	case "cycleway":
		return AccessBicycle
	}
	
	// In this case we don't know... just ignore this way.
	return 0
}

// Compute the access mask for a given way.
func accessMask(way Way) AccessType {
	var mask AccessType = 0
	individualTag := false
	
	// The designated access tags are hirachical.
	// This means that more specific tags override higher levels.
	for _, data := range AccessTable {
		if _, ok := way.Attributes[data.Key]; ok {
			individualTag = true
			if parseBoolean(way, data.Key) {
				mask |= data.Mask
			} else {
				mask &= ^data.Mask
			}
		}
	}
	
	// If this way was not individually tagged, return the default evaluation.
	if !individualTag {
		return defaultAccessMask(way)
	}
	return mask
}

func (f RoutingFilter) VisitWay(way Way) {
	// If this way is not a road to begin with, ignore it.
	if _, ok := way.Attributes["highway"]; !ok {
		// According to the wiki this seems necessary, but in practice the highway tag
		// is always present. (at least for Germany)
		if _, ok := way.Attributes["junction"]; !ok {
			return
		}
	}
	
	
	// We ignore ways with less than 2 nodes, since those will not become edges
	// in the street graph. Maybe this should throw an error?
	if len(way.Nodes) < 2 {
		return
	}
	
	// Ok, so it's a road. If we can access it we might as well have a look.
	access := accessMask(way)
	if access & f.access == 0 {
		//fmt.Printf("access: %v, should: %v\n", access, f.access)
		return
	}
	normalizeOneway(way)
	f.client.VisitWay(way)
}

// Uhm... encapsulation?
func NewRoutingFilter(client Visitor, access AccessType) RoutingFilter {
	return RoutingFilter{
		client: client,
		access: access,
	}
}

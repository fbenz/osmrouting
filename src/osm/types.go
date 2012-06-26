package osm

/*
 * Simpler types for working with open streetmap data in go.
 *
 * Every element has a 64 bit id which is persistent and unique among
 * elements of the same type. Tags are documented on the osm wiki at:
 *   http://wiki.openstreetmap.org/wiki/Map_Features
 *
 * In order to add a new tag, first check for the actual usage on
 * TagWatch: http://tagwatch.stoecker.eu/
 */

import (
	"geo"
)

type Type int

const (
	TypeNode Type = iota
	TypeEdge
	TypeRelation
)

// A node is one of the core elements in the OpenStreetMap data model.
// It consists of a single geospatial point using a latitude and longitude.
// Nodes can be used to define standalone point features or be used to define
// the path of a way. 
type Node struct {
	Id         int64
	Position   geo.Coordinate
	Attributes map[string]string
}

// A way is an ordered list of nodes which normally also has at least one
// tag or is included within a Relation.
type Way struct {
	Id         int64
	Nodes      []int64
	Attributes map[string]string
}

// A relation is one of the core data elements that consists of one or more tags
// and also an ordered list of one or more nodes and/or ways as members which is
// used to define logical or geographic relationships between other elements.
// A member of a relation can optionally have a role which describe the part that
// a particular feature plays within a relation. 
type Relation struct {
	Id         int64
	Members    []RelationMember
	Attributes map[string]string
}

// The meaning of the Id attribute depends on the Type. Role is completely dependant
// on the value of the "type" attribute of the containing relation.
type RelationMember struct {
	Type Type
	Id   int64
	Role string
}

type Visitor interface {
	VisitNode(Node)
	VisitWay(Way)
	VisitRelation(Relation)
}

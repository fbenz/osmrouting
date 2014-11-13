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

package osm

import (
	"code.google.com/p/goprotobuf/proto"
	"errors"
	"geo"
	"io"
	"os"
	"osm/pbf"
)

// Supported features
var Features = [...]string{
	"OsmSchema-V0.6",
	"DenseNodes",
}

// Parse the HeaderBlock. According to spec every file has to start with
// a header. We are required to parse it in order to check the required
// features.
func parseHeader(stream io.Reader) error {
	block, err := ReadBlock(stream)
	if err != nil {
		return err
	}

	if block.Kind != OSMHeader {
		return errors.New("File does not start with a HeaderBlock")
	}

	header := &pbf.HeaderBlock{}
	if err := proto.Unmarshal(block.Data, header); err != nil {
		return err
	}

	for _, feature := range header.RequiredFeatures {
		supported := false
		for _, capability := range Features {
			if capability == feature {
				supported = true
				break
			}
		}
		if !supported {
			return errors.New("Unsupported feature: " + feature)
		}
	}

	return nil
}

// Parse a primitive block, which is a container for an arbitrary sequence
// of data elements.
func parsePrimitiveBlock(block *Block) (*pbf.PrimitiveBlock, error) {
	if block.Kind != OSMData {
		return nil, errors.New("Duplicate HeaderBlock")
	}

	primitive := &pbf.PrimitiveBlock{}
	if err := proto.Unmarshal(block.Data, primitive); err != nil {
		return nil, err
	}
	return primitive, nil
}

// Internally, the encoding of coordinates depends on the context.
// This function maps between raw lat/long values and Coordinates.
func parseLocation(rawlat, rawlng int64, block *pbf.PrimitiveBlock) geo.Coordinate {
	granularity := int64(block.GetGranularity())
	latOffset := block.GetLatOffset()
	lngOffset := block.GetLonOffset()
	lat := float64(latOffset + granularity*rawlat) / 1000000000.0
	lng := float64(lngOffset + granularity*rawlng) / 1000000000.0
	return geo.Coordinate{lat, lng}
}

// Attributes are represented as two parallel arrays of indices into the
// block's string table.
func parseAttributes(keys, vals []uint32, block *pbf.PrimitiveBlock) map[string]string {
	attributes := map[string]string{}
	for i, keyIndex := range keys {
		valIndex := vals[i]
		key := string(block.Stringtable.S[keyIndex])
		val := string(block.Stringtable.S[valIndex])
		attributes[key] = val
	}
	return attributes
}

// Parse a node in the standard format.
func visitNode(node *pbf.Node, block *pbf.PrimitiveBlock, client Visitor) {
	rawlat := node.GetLat()
	rawlon := node.GetLon()
	n := Node{
		Id:         node.GetId(),
		Position:   parseLocation(rawlat, rawlon, block),
		Attributes: parseAttributes(node.Keys, node.Vals, block),
	}
	client.VisitNode(n)
}

// Parse an array of nodes in the dense format.
// This is basically an array of nodes, but with all attribute data tightly packed
// into a single array.
func visitDenseNodes(group *pbf.PrimitiveGroup, block *pbf.PrimitiveBlock, client Visitor) {
	var prevNodeId int64 = 0
	var prevLat int64 = 0
	var prevLon int64 = 0
	keyValIndex := 0

	for idx, deltaNodeId := range group.Dense.Id {
		id := prevNodeId + deltaNodeId
		rawlon := prevLon + group.Dense.Lon[idx]
		rawlat := prevLat + group.Dense.Lat[idx]
		pos := parseLocation(rawlat, rawlon, block)

		prevNodeId = id
		prevLon = rawlon
		prevLat = rawlat

		// This is undocumented behaviour: If the length of the KeyVals array
		// is less than the number of nodes, the remaining nodes do not have
		// key/value pairs associated with them.
		attributes := map[string]string{}
		if keyValIndex < len(group.Dense.KeysVals) {
			for group.Dense.KeysVals[keyValIndex] != 0 {
				key := string(block.Stringtable.S[group.Dense.KeysVals[keyValIndex]])
				val := string(block.Stringtable.S[group.Dense.KeysVals[keyValIndex+1]])
				attributes[key] = val
				keyValIndex += 2
			}
			keyValIndex++
		}

		client.VisitNode(Node{id, pos, attributes})
	}
}

func visitWay(way *pbf.Way, block *pbf.PrimitiveBlock, client Visitor) {
	w := Way{
		Id:         way.GetId(),
		Nodes:      make([]int64, len(way.Refs)),
		Attributes: parseAttributes(way.Keys, way.Vals, block),
	}

	var prevId int64 = 0
	for i, ref := range way.Refs {
		w.Nodes[i] = prevId + ref
		prevId += ref
	}

	client.VisitWay(w)
}

func visitRelation(relation *pbf.Relation, block *pbf.PrimitiveBlock, client Visitor) {
	r := Relation{
		Id:         relation.GetId(),
		Members:    make([]RelationMember, len(relation.Memids)),
		Attributes: parseAttributes(relation.Keys, relation.Vals, block),
	}

	var prevId int64 = 0
	for i, deltaId := range relation.Memids {
		r.Members[i] = RelationMember{
			Id:   prevId + deltaId,
			Type: Type(relation.Types[i]),
			Role: string(block.Stringtable.S[relation.RolesSid[i]]),
		}
		prevId += deltaId
	}

	client.VisitRelation(r)
}

func visitGroup(group *pbf.PrimitiveGroup, block *pbf.PrimitiveBlock, client Visitor) {
	for _, node := range group.Nodes {
		visitNode(node, block, client)
	}

	if group.Dense != nil {
		visitDenseNodes(group, block, client)
	}

	for _, way := range group.Ways {
		visitWay(way, block, client)
	}

	for _, relation := range group.Relations {
		visitRelation(relation, block, client)
	}
}

func readBlocks(stream io.Reader, cs chan *Block) {
	for {
		block, err := ReadBlock(stream)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err.Error())
		}
		cs <- block
	}
	cs <- nil
}

func readPrimitiveBlocks(stream io.Reader, cs chan *pbf.PrimitiveBlock) {
	bs := make(chan *Block, 2)
	go readBlocks(stream, bs)
	for block := range bs {
		if block == nil {
			break
		}
		pb, err := parsePrimitiveBlock(block)
		if err != nil {
			panic(err.Error())
		}
		cs <- pb
	}
	cs <- nil
}

func Parse(stream io.Reader, client Visitor) error {
	err := parseHeader(stream)
	if err != nil {
		return err
	}

	cs := make(chan *pbf.PrimitiveBlock, 2)
	go readPrimitiveBlocks(stream, cs)
	for block := range cs {
		if block == nil {
			break
		}
		for _, group := range block.Primitivegroup {
			visitGroup(group, block, client)
		}
	}
	return nil
}

func ParseFile(file *os.File, client Visitor) error {
	_, err := file.Seek(0, 0)
	if err != nil {
		return err
	}

	return Parse(file, client)
}

package pbf

import (
	"code.google.com/p/goprotobuf/proto"
	"io"
	"parser/OSMPBF"
)

type Node struct {
	Id         int64
	Lat        float64
	Lon        float64
	Attributes map[string]string
}

type Way struct {
	Id         int64
	Nodes      []int64
	Attributes map[string]string
}

type Visitor interface {
	VisitNode(Node)
	VisitWay(Way)
}

func readPrimitiveBlock(stream io.Reader) (*OSMPBF.PrimitiveBlock, error) {
	// Locate the next Data block
	var block *Block = nil
	var err error = nil
	for {
		block, err = ReadBlock(stream)
		if err != nil {
			return nil, err
		}
		//log.Printf("Block{ kind: %d, size: %d }\n", block.Kind, len(block.Data))
		if block.Kind == OSMData {
			break
		}
	}

	primitive := &OSMPBF.PrimitiveBlock{}
	if err := proto.Unmarshal(block.Data, primitive); err != nil {
		return nil, err
	}
	return primitive, nil
}

func decodeLocation(rawlat int64, rawlon int64, block *OSMPBF.PrimitiveBlock) (float64, float64) {
	lonOffset := proto.GetInt64(block.LonOffset)
	latOffset := proto.GetInt64(block.LatOffset)
	granularity := int64(proto.GetInt32(block.Granularity))

	lon := .000000001 * float64(lonOffset+(granularity*rawlon))
	lat := .000000001 * float64(latOffset+(granularity*rawlat))
	return lat, lon
}

func visitNode(node *OSMPBF.Node, block *OSMPBF.PrimitiveBlock, client Visitor) {
	id := proto.GetInt64(node.Id)

	rawlat := proto.GetInt64(node.Lat)
	rawlon := proto.GetInt64(node.Lon)
	lat, lon := decodeLocation(rawlat, rawlon, block)

	attributes := map[string]string{}
	for i, keyIndex := range node.Keys {
		valIndex := node.Vals[i]
		key := string(block.Stringtable.S[keyIndex])
		val := string(block.Stringtable.S[valIndex])
		attributes[key] = val
	}

	client.VisitNode(Node{id, lat, lon, attributes})
}

func visitDenseNodes(group *OSMPBF.PrimitiveGroup, block *OSMPBF.PrimitiveBlock, client Visitor) {
	var prevNodeId int64 = 0
	var prevLat int64 = 0
	var prevLon int64 = 0
	keyValIndex := 0

	for idx, deltaNodeId := range group.Dense.Id {
		id := prevNodeId + deltaNodeId
		rawlon := prevLon + group.Dense.Lon[idx]
		rawlat := prevLat + group.Dense.Lat[idx]
		lat, lon := decodeLocation(rawlat, rawlon, block)

		prevNodeId = id
		prevLon = rawlon
		prevLat = rawlat

		// This is undocumented behaviour: If the length of the KeyVals array
		// is less than the number of nodes, the remaining nodes do not have
		// key/value pairs associated with them.
		attributes := map[string]string{}
		if len(group.Dense.KeysVals) != 0 {
			length := 0
			for group.Dense.KeysVals[keyValIndex+2*length] != 0 {
				length++
			}

			for i := 0; i < length; i++ {
				key := string(block.Stringtable.S[group.Dense.KeysVals[keyValIndex+(i*2)]])
				val := string(block.Stringtable.S[group.Dense.KeysVals[keyValIndex+(i*2)+1]])
				attributes[key] = val
			}

			keyValIndex += 2 * length
		}

		client.VisitNode(Node{id, lat, lon, attributes})

		keyValIndex++
	}
}

func visitWay(way *OSMPBF.Way, block *OSMPBF.PrimitiveBlock, client Visitor) {
	id := proto.GetInt64(way.Id)

	attributes := map[string]string{}
	for i, keyIndex := range way.Keys {
		valIndex := way.Vals[i]
		key := string(block.Stringtable.S[keyIndex])
		val := string(block.Stringtable.S[valIndex])
		attributes[key] = val
	}

	var prevId int64 = 0
	refs := make([]int64, len(way.Refs))
	for i, ref := range way.Refs {
		r := prevId + ref
		refs[i] = r
		prevId = r
	}

	client.VisitWay(Way{id, refs, attributes})
}

func visitGroup(group *OSMPBF.PrimitiveGroup, block *OSMPBF.PrimitiveBlock, client Visitor) {
	for _, node := range group.Nodes {
		visitNode(node, block, client)
	}

	if group.Dense != nil {
		visitDenseNodes(group, block, client)
	}

	for _, way := range group.Ways {
		visitWay(way, block, client)
	}
}

func VisitGraph(stream io.Reader, client Visitor) error {
	for {
		block, err := readPrimitiveBlock(stream)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		for _, group := range block.Primitivegroup {
			visitGroup(group, block, client)
		}
	}
	return nil
}

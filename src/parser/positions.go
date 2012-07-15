
/*
 * To recover all edge steps we need to keep a massive table with (almost) all
 * coordinates in the pbf file. To reduce the overhead, we use another veb tree
 * instead of a simple map.
 */

package main

import "geo"

type Positions interface {
	Get(id int64) geo.Coordinate
	Set(id int64, p geo.Coordinate)
}

type FlatVector []uint64

// Global region allocator (since we would overflow the go heap otherwise)
var allocator *Region

func init() {
	allocator = NewRegion()
}

func EncodePoint(p geo.Coordinate) uint64 {
	lat, lng := p.Encode()
	return (uint64(lat) << 32) | uint64(lng)
}

func DecodePoint(p uint64) geo.Coordinate {
	lat := int32(p >> 32)
	lng := int32(p & 0xffffffff)
	return geo.DecodeCoordinate(lat, lng)
}

func (v FlatVector) Get(key int64) geo.Coordinate {
	return DecodePoint(v[key])
}

func (v FlatVector) Set(key int64, p geo.Coordinate) {
	v[key] = EncodePoint(p)
}

type VebTree struct {
	bits uint
	data map[int64] Positions
}

func (t *VebTree) Get(key int64) geo.Coordinate {
	msb := (key & (^0 << t.bits)) >> t.bits
	if subtable, ok := t.data[msb]; ok {
		lsb := key & ((1 << t.bits) - 1)
		return subtable.Get(lsb)
	}
	return DecodePoint(0)
}

func (t *VebTree) Set(key int64, p geo.Coordinate) {
	msb := (key & (^0 << t.bits)) >> t.bits
	lsb := key & ((1 << t.bits) - 1)
	subtable, ok := t.data[msb]
	if !ok {
		subtable = NewPositions(t.bits)
		t.data[msb] = subtable
	}
	subtable.Set(lsb, p)
}

func nextPowerOf2(v uint64) uint64 {
	if v == 0 {
		return 1
	} else if v & (v - 1) == 0 {
		return v
	}
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v |= v >> 32
	return v + 1
}

func NewPositions(bits uint) Positions {
	if bits <= 8 {
		flatv := allocator.AllocateUint64(1 << bits)
		//flatv := make([]uint64, 1 << bits, 1 << bits)
		//for i, _ := range flatv {
		//	flatv[i] = 0
		//}
		return FlatVector(flatv)
	}
	
	// In order to use a VebTree the word size needs to be
	// a power of two. Also, we pass around pointers to VebTrees,
	// instead of copying them everytime.
	bits = uint(nextPowerOf2(uint64(bits)))
	return &VebTree{
		bits: bits / 2,
		data: map[int64] Positions {},
	}
}

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


/*
 * To recover all edge steps we need to keep a massive table with (almost) all
 * coordinates in the pbf file. To reduce the overhead, we use another veb tree
 * instead of a simple map.
 */

package main

import (
	"geo"
	"mm"
)

type Positions interface {
	Get(id int64) geo.Coordinate
	Set(id int64, p geo.Coordinate)
}

type FlatVector []int32

// Global region allocator (since we would overflow the go heap otherwise)
var allocator *mm.Region

func init() {
	allocator = mm.NewRegion(0)
}

func (v FlatVector) Get(key int64) geo.Coordinate {
	lat, lng := v[2*key], v[2*key+1]
	return geo.DecodeCoordinate(lat, lng)
}

func (v FlatVector) Set(key int64, p geo.Coordinate) {
	v[2*key], v[2*key+1] = p.Encode()
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
	return geo.Coordinate{}
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
		var flatv []int32
		allocator.Allocate(1 << (bits+1), &flatv)
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

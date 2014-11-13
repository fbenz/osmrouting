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
 * Package BitVector
 *
 * Here's a problem for you: The germany.osm.pbf file contains 101628726 nodes.
 * That's more than 2^26 (actually ~3 * 2^25) nodes.
 * The usual way to implement a set in go is as:
 *   map[int64] bool
 * and for a small number of items, say less than 2^23 this is very much acceptable.
 * Beyond that you run into trouble on a certain memory starved machine. Specifically,
 * my machine chokes (2GB of RAM total) for n somewhere between 2^24 and 2^25 elements.
 *
 * This is problematic, since the parser needs to find all node intersections
 * and that means keeping at least one bit per node. Well, this file is the
 * compromise to make this possible, with an acceptable performance hit.
 *
 * We implement van Emde Boas trees for 64 bit integers, shifting to flat
 * bitvectors for 16 bits or fewer. In my tests, the size of the resuling
 * sets quickly reaches about 300 MB if you use random 32 bit ints - that's
 * obvious since we just have to hit every second 16 kB block once to get to this
 * level. Beyond that it stays pretty much constant and will never be larger
 * than about 600MB - which is quite acceptable, since a plain bit vector
 * would require 512MB for the whole set of 32 bit integers.
 * The whole set of 64 bit integers will not fit into memory, no matter which
 * future you come from, so I wouldn't worry too much.
 *
 * In principle space consumption is linear, but as mentioned the overhead is
 * ridiculous for small sets. This is mostly since our block size is 16 bits,
 * rather than the more common 8 bits. (16 bits correspond to 8 kB of storage,
 * 8 bits are just 32 words)
 *
 * This is *necessary* the reason is that the meta data overhead of a map in
 * go is very high. I think in my measurements it was about 50 bytes per element.
 * If you set the block size to 8 bits, the program will explode (or really, use
 * more than a GB of storage) relatively quickly. Although it will scale much
 * smoother with few elements. But then again, if you have less than 2^24 elements,
 * don't use this. It's not going to be faster, and it is going to need more
 * memory unless all of your values are pretty much contiguous.
 */

package alg

import (
	"fmt"
	"unsafe"
)

/*
 * Notes on Van-Emde Boas trees:
 * - We use a hash table to store the pointers to the next level and only
 *   create objects for non-empty subtrees. This means that the space consumption
 *   is linear. The overhead is still large for small sets though, but that's
 *   not really surprising.
 * - For k = 8, store a bitvector explicitly. This is because at this level
 *   we need 2^8 / 8 = 32 bytes to store the bitvector. Assuming 32-bit pointers,
 *   we would store 16 * 4 = 64 bytes for the additional indirection. This just
 *   just isn't going to help and will hurt space consumption for even a relatively
 *   modest load factor.
 * - Go does not have union types. We use interfaces instead, which results in an
 *   additional pointer for every reference. Since a veb tree stores only order
 *   sqrt(n) indirections this should be bearable.
 */

type BitVector interface {
	Get(key int64) bool
	Set(key int64, value bool)
}

type FlatBitVector []byte

func (v FlatBitVector) Get(key int64) bool {
	i := key / 8
	if i < 0 || i >= int64(len(v)) {
		return false
	}
	return (v[i] & (1 << uint(key % 8))) != 0
}

func (v FlatBitVector) Set(key int64, value bool) {
	i := key / 8
	if i < 0 || i >= int64(len(v)) {
		panic(fmt.Sprintf("index out of range: %d, len: %d", i, len(v)))
	}
	var bit byte = 1 << uint(key % 8)
	if value {
		v[i] |= bit
	} else {
		v[i] &= ^bit
	}
}

type VebTree struct {
	bits uint
	data map[int64] BitVector
}

func (t *VebTree) Get(key int64) bool {
	msb := (key & (^0 << t.bits)) >> t.bits
	if subtable, ok := t.data[msb]; ok {
		lsb := key & ((1 << t.bits) - 1)
		return subtable.Get(lsb)
	}
	return false
}

func (t *VebTree) Set(key int64, value bool) {
	msb := (key & (^0 << t.bits)) >> t.bits
	lsb := key & ((1 << t.bits) - 1)
	subtable, ok := t.data[msb]
	if !ok {
		subtable = NewBitVector(t.bits)
		t.data[msb] = subtable
	}
	subtable.Set(lsb, value)
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

func NewBitVector(bits uint) BitVector {
	if bits <= 16 {
		flatBv := make([]byte, (1 << bits) / 8, (1 << bits) / 8)
		for i, _ := range flatBv {
			flatBv[i] = 0
		}
		return FlatBitVector(flatBv)
	}
	
	// In order to use a VebTree the word size needs to be
	// a power of two. Also, we pass around pointers to VebTrees,
	// instead of copying them everytime.
	bits = uint(nextPowerOf2(uint64(bits)))
	return &VebTree{
		bits: bits / 2,
		data: map[int64] BitVector {},
	}
}

func BitVectorDebugSize(b BitVector) int {
	switch b.(type) {
	case FlatBitVector:
		return cap(b.(FlatBitVector))
	case *VebTree:
		tree  := b.(*VebTree)
		total := int(unsafe.Sizeof(*tree))
		count := 0
		for _, subtable := range tree.data {
			total += BitVectorDebugSize(subtable)
			count++
		}
		count = int(nextPowerOf2(uint64(count)))
		return 4 * count + total
	}
	return 0
}

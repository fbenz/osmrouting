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


package mm

type Region struct {
	BlockSize  int
	Block      []byte
	Chunks     [][]byte
	HugeBlocks [][]byte
}

const DefaultBlockSize = (1 << 20) // 1 MB blocks

func Allocate(size int, p interface{}) error {
	rsize := size * reflect_elem_size(p)
	m, err := sys_mmap_anon(rsize)
	if err != nil {
		return err
	}
	ProfileAllocate(rsize)
	reflect_set(p, m)
	return nil
}

func Free(p interface{}) error {
	b := reflect_get(p)
	ProfileFree(len(b))
	if err := sys_close(b); err != nil {
		return err
	}
	reflect_set(p, nil)
	return nil
}

func new_block(r *Region) {
	bk, err := sys_mmap_anon(r.BlockSize)
	if err != nil {
		panic(err.Error())
	}
	ProfileAllocate(r.BlockSize)
	r.Block  = bk
	r.Chunks = append(r.Chunks, bk)
}

func NewRegion(blockSize int) *Region {
	if blockSize == 0 {
		blockSize = DefaultBlockSize
	}
	r := new(Region)
	r.BlockSize = blockSize
	new_block(r)
	return r
}

func (r *Region) Allocate(size int, p interface{}) error {
	rsize := size * reflect_elem_size(p)
	
	if rsize > r.BlockSize {
		bk, err := sys_mmap_anon(rsize)
		if err != nil {
			return err
		}
		reflect_set(p, bk)
		r.HugeBlocks = append(r.HugeBlocks, bk)
		return nil
	}
	
	if len(r.Block) < rsize {
		new_block(r)
	}
	bk := r.Block[:rsize]
	r.Block = r.Block[rsize:]
	reflect_set(p, bk)
	return nil
}

func (r *Region) Free() error {
	r.Block = nil
	for _, bk := range r.Chunks {
		ProfileFree(len(bk))
		err := sys_close(bk)
		if err != nil {
			return err
		}
	}
	r.Chunks = nil
	for _, bk := range r.HugeBlocks {
		ProfileFree(len(bk))
		err := sys_close(bk)
		if err != nil {
			return err
		}
	}
	r.HugeBlocks = nil
	return nil
}

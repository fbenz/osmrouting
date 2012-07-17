
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

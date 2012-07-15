// Avoid go's memory manager (which has a hard limit of 16GB).
// We use mmap for file output, which is actually more convenient, but does
// not offer the memory safety guarantees of regular go. Furthermore we can
// use mmap with MAP_ANONYMOUS to get a simple malloc implementation for large
// blocks.

package main

import (
	"fmt"
	"os"
	"reflect"
	"syscall"
	"unsafe"
)

type Region struct {
	Block   []byte
	Chunks  [][]byte
}

const BlockSize = (1 << 20) // 1 MB blocks

// mmap calls

func MapFile(name string, size int) ([]byte, error) {
	file, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	
	// Ensure that the file is large enough
	_, err = file.WriteAt([]byte{0}, int64(size - 1))
	if err != nil {
		file.Close()
		return nil, err
	}
	
	fd := int(file.Fd())
	m, err := syscall.Mmap(fd, 0, size, syscall.PROT_READ | syscall.PROT_WRITE, syscall.MAP_SHARED)
	
	// Mmap creates a new reference to the file descriptor, so we can close it immediately.
	file.Close()
	return m, err
}

func MapAnonymous(size int) ([]byte, error) {
	return syscall.Mmap(-1, 0, size, syscall.PROT_READ | syscall.PROT_WRITE,
		syscall.MAP_PRIVATE | syscall.MAP_ANON)
}

func Unmap(mapping []byte) error {
	return syscall.Munmap(mapping)
}

func Sync(mapping []byte) error {
	h := *(*reflect.SliceHeader)(unsafe.Pointer(&mapping))
	flags := syscall.MS_SYNC
	_, _, err := syscall.Syscall(syscall.SYS_MSYNC, uintptr(h.Data), uintptr(h.Len), uintptr(flags))
	if err != 0 {
		return err
	}
	return nil
}

func UnmapFile(mapping []byte) error {
	err := Sync(mapping)
	if err != nil {
		return err
	}
	return Unmap(mapping)
}

// memory manager

func newBlock(r *Region) {
	bk, err := MapAnonymous(BlockSize)
	if err != nil {
		panic(err.Error())
	}
	r.Block  = bk
	r.Chunks = append(r.Chunks, bk)
}

func NewRegion() *Region {
	r := new(Region)
	newBlock(r)
	return r
}

func FreeRegion(r *Region) {
	for _, bk := range r.Chunks {
		err := Unmap(bk)
		if err != nil {
			panic(err.Error())
		}
	}
	r.Block  = nil
	r.Chunks = nil
}

func (r *Region) Allocate(size int) []byte {
	if size > BlockSize {
		panic(fmt.Sprintf("Cannot allocate region of size %v.\n", size))
	}
	if len(r.Block) < size {
		newBlock(r)
	}
	c := r.Block[0:size]
	r.Block = r.Block[size:len(r.Block)-1]
	return c
}

// typed interface (this sucks)

func MapFileUint32(name string, size int) ([]uint32, error) {
	m, err := MapFile(name, size * 4)
	if err != nil {
		return nil, err
	}

	dh := (*reflect.SliceHeader)(unsafe.Pointer(&m))
	dh.Len /= 4
	dh.Cap /= 4
	return *(*[]uint32)(unsafe.Pointer(&m)), nil
}

func MapFileInt32(name string, size int) ([]int32, error) {
	m, err := MapFile(name, size * 4)
	if err != nil {
		return nil, err
	}

	dh := (*reflect.SliceHeader)(unsafe.Pointer(&m))
	dh.Len /= 4
	dh.Cap /= 4
	return *(*[]int32)(unsafe.Pointer(&m)), nil
}

func MapFileUint16(name string, size int) ([]uint16, error) {
	m, err := MapFile(name, size * 2)
	if err != nil {
		return nil, err
	}

	dh := (*reflect.SliceHeader)(unsafe.Pointer(&m))
	dh.Len /= 2
	dh.Cap /= 2
	return *(*[]uint16)(unsafe.Pointer(&m)), nil
}

func UnmapFileInt32(m []int32) error {
	dh := (*reflect.SliceHeader)(unsafe.Pointer(&m))
	dh.Len *= 4
	dh.Cap *= 4
	return UnmapFile(*(*[]byte)(unsafe.Pointer(&m)))
}

func UnmapFileUint32(m []uint32) error {
	dh := (*reflect.SliceHeader)(unsafe.Pointer(&m))
	dh.Len *= 4
	dh.Cap *= 4
	return UnmapFile(*(*[]byte)(unsafe.Pointer(&m)))
}

func UnmapFileUint16(m []uint16) error {
	dh := (*reflect.SliceHeader)(unsafe.Pointer(&m))
	dh.Len *= 2
	dh.Cap *= 2
	return UnmapFile(*(*[]byte)(unsafe.Pointer(&m)))
}

func (r *Region) AllocateUint64(size int) []uint64 {
	b := r.Allocate(size * 8)
	dh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	dh.Len /= 8
	dh.Cap /= 8
	return *(*[]uint64)(unsafe.Pointer(&b))
}

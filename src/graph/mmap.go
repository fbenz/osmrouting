package graph

import (
	"os"
	"path"
	"reflect"
	"syscall"
	"unsafe"
)

func mapFile(base, name string) ([]byte, error) {
	file, err := os.Open(path.Join(base, name))
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	// Thanks to Windows compatibility file.Fd is not declared int...
	fdfu := file.Fd()
	fd := *(*int)(unsafe.Pointer(&fdfu))
	// This is bad. Slices have int size and capacity fields, which
	// means that we might truncate here. We can work around this issue
	// by using unsafe.Pointer internally and only convert to slices for
	// individual edge/step lists... But for now our files are small
	// and this works:
	size := int(info.Size())
	return syscall.Mmap(fd, 0, size, syscall.PROT_READ, syscall.MAP_PRIVATE)
}

func MmapFileUint32(base, name string) ([]uint32, error) {
	m, err := mapFile(base, name)
	if err != nil {
		return nil, err
	}

	dh := (*reflect.SliceHeader)(unsafe.Pointer(&m))
	dh.Len /= 4
	dh.Cap /= 4
	return *(*[]uint32)(unsafe.Pointer(&m)), nil
}

func MmapFileFloat64(base, name string) ([]float64, error) {
	m, err := mapFile(base, name)
	if err != nil {
		return nil, err
	}

	dh := (*reflect.SliceHeader)(unsafe.Pointer(&m))
	dh.Len /= 8
	dh.Cap /= 8
	return *(*[]float64)(unsafe.Pointer(&m)), nil
}

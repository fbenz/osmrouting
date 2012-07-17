
package mm

import (
	"syscall"
)

func sys_mmap_open(fd, size int) ([]byte, error) {
	return syscall.Mmap(fd, 0, size, syscall.PROT_READ, syscall.MAP_PRIVATE)
}

func sys_mmap_anon(size int) ([]byte, error) {
	prot := syscall.PROT_READ | syscall.PROT_WRITE
	flag := syscall.MAP_PRIVATE | syscall.MAP_ANON
	return syscall.Mmap(-1, 0, size, prot, flag)
}

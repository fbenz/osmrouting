
package mm

import (
	"syscall"
)

func sys_mmap_open(fd, size int) ([]byte, error) {
	flag := syscall.MAP_POPULATE | syscall.MAP_PRIVATE
	return syscall.Mmap(fd, 0, size, syscall.PROT_READ, flag)
}

func sys_mmap_anon(size int) ([]byte, error) {
	prot := syscall.PROT_READ | syscall.PROT_WRITE
	flag := syscall.MAP_PRIVATE | syscall.MAP_ANONYMOUS
	return syscall.Mmap(-1, 0, size, prot, flag)
}

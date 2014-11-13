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

import (
	"fmt"
	"os"
	"reflect"
	"syscall"
	"unsafe"
)

// If int is 64 bits we have no problems, as we can use mmap on files whose
// size in bytes fits in an int. This annoying restriction stems from the
// internal representation of slices in Go. We could work around it of
// course, but it's probably better to just wait for an official fix.
const maxInt = int(^uint(0) >> 1)

func sys_open(path string) ([]byte, error) {
	// Open the file and call stat to determine it's size.
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	
	// At this point it might be possible that the size overflows an int,
	// in which case we cannot map the whole file into memory. This is not
	// really a recoverable error and so we panic.
	if info.Size() > int64(maxInt) {
		panic(fmt.Sprintf("%s does not fit into memory.", path))
	}
	
	fd   := int(file.Fd())
	size := int(info.Size())
	return sys_mmap_open(fd, size)
}

func sys_create(path string, size int) ([]byte, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	// Ensure that the file is large enough
	_, err = file.WriteAt([]byte{0}, int64(size - 1))
	if err != nil {
		return nil, err
	}
	
	fd := int(file.Fd())
	return syscall.Mmap(fd, 0, size, syscall.PROT_READ | syscall.PROT_WRITE, syscall.MAP_SHARED)
}

func sys_sync(m []byte) error {
	h := *(*reflect.SliceHeader)(unsafe.Pointer(&m))
	flags := syscall.MS_SYNC
	_, _, err := syscall.Syscall(syscall.SYS_MSYNC, uintptr(h.Data), uintptr(h.Len), uintptr(flags))
	if err != 0 {
		return err
	}
	return nil
}

func sys_close(m []byte) error {
	return syscall.Munmap(m)
}

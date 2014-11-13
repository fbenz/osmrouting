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

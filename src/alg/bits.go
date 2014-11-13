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

package alg

var (
	popc []byte
)

func init() {
	popc = make([]byte, 256)
	for i := range popc {
		popc[i] = popc[i / 2] + byte(i % 2)
	}
}

func GetBit(ary []byte, i uint) bool {
	return ary[i / 8] & (1 << (i % 8)) != 0
}

func SetBit(ary []byte, i uint) {
	ary[i / 8] |= 1 << (i % 8)
}

func Intersection(a, b []byte) []byte {
	l := len(a)
	if len(b) < l {
		l = len(b)
	}
	result := make([]byte, l)
	for i, _ := range result {
		result[i] = a[i] & b[i]
	}
	return result
}

func Union(a, b []byte) []byte {
	l := len(a)
	if len(b) < l {
		l = len(b)
	}
	result := make([]byte, l)
	for i, _ := range result {
		result[i] = a[i] | b[i]
	}
	return result
}

func Popcount(ary []byte) int {
	size := 0
	for _, b := range ary {
		size += int(popc[b])
	}
	return size
}

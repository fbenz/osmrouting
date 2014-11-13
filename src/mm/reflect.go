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
	"reflect"
	"unsafe"
)

func reflect_value(p interface{}) reflect.Value {
	s := reflect.ValueOf(p)
	if s.Kind() != reflect.Ptr || s.Elem().Kind() != reflect.Slice {
		panic("Argument should be a pointer to a slice.")
	}
	return s.Elem()
}

func reflect_elem_size(p interface{}) int {
	s := reflect_value(p)
	return int(s.Type().Elem().Size())
}

func reflect_slice(p interface{}) (*reflect.SliceHeader, int) {
	s := reflect_value(p)
	head := (*reflect.SliceHeader)(unsafe.Pointer(s.UnsafeAddr()))
	size := int(s.Type().Elem().Size())
	return head, size
}

func reflect_get(p interface{}) []byte {
	s, size := reflect_slice(p)
	
	var r []byte
	h := (*reflect.SliceHeader)(unsafe.Pointer(&r))
	h.Data = s.Data
	h.Len  = s.Len * size
	h.Cap  = s.Cap * size
	return r
}

func reflect_set(p interface{}, b []byte) {
	s, size := reflect_slice(p)
	s.Data = (*reflect.SliceHeader)(unsafe.Pointer(&b)).Data
	s.Len  = len(b) / size
	s.Cap  = cap(b) / size
}

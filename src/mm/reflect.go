
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

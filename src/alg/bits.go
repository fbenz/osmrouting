
package alg

var (
	popc []byte
)

func init() {
	popc = make([]byte, 256)
	for i := range popc {
		c := 0
		for j := i; j != 0; j = j & (j - 1) {
			c++
		}
		popc[i] = byte(c)
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

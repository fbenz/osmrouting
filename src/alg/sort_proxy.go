
package alg

import (
	"sort"
)

type SortProxy struct {
	Client      sort.Interface
	Size        int
	Permutation []int
}

func (v *SortProxy) Len() int {
	return v.Size
}

func (v *SortProxy) Less(i, j int) bool {
	return v.Client.Less(v.Permutation[i], v.Permutation[j])
}

func (v *SortProxy) Swap(i, j int) {
	v.Permutation[i], v.Permutation[j] = v.Permutation[j], v.Permutation[i]
}

func SortPermutation(c sort.Interface) []int {
	l := c.Len()
	proxy := &SortProxy{
		Client:      c,
		Size:        l,
		Permutation: make([]int, l),
	}
	for i := 0; i < l; i++ {
		proxy.Permutation[i] = i
	}
	sort.Sort(proxy)
	if !sort.IsSorted(proxy) {
		panic("Failed to sort sequence.")
	}
	return proxy.Permutation
}

func ApplyPermutation(c sort.Interface, p []int) {
	for i := 0; i < len(p); i++ {
		for i != p[i] {
			c.Swap(p[i], p[p[i]])
			p[i], p[p[i]] = p[p[i]], p[i]
		}
	}
}

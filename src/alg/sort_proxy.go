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

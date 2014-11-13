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
	"math/rand"
	"testing"
)

type Permutation []int

const MinPermutationSize = 10
const MaxPermutationSize = 100
const NumTests = 1000

func (p Permutation) Len() int {
	return len(p)
}

func (p Permutation) Less(i, j int) bool {
	return p[i] < p[j]
}

func (p Permutation) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func TestSortPermutation(t *testing.T) {
	// QuickCheck cannot distinguish between Permutation and []int ?
	for k := 0; k < NumTests; k++ {
		n := MinPermutationSize + rand.Intn(MaxPermutationSize - MinPermutationSize)
		p := Permutation(rand.Perm(n))
		q := SortPermutation(p)
		// q should be the inversion of p and thus q[p[i]] = i
		for i := 0; i < len(p); i++ {
			if q[p[i]] != i {
				t.Errorf("q is not an inversion of p\n")
				t.Errorf("  p: %v\n", p)
				t.Errorf("  q: %v\n", q)
			}
		}
		// apply permutation corresponds to composition of permutations
		// so we should have q . p = id
		p2 := make([]int, len(p))
		q2 := make([]int, len(q))
		copy(p2, p)
		copy(q2, q)
		ApplyPermutation(p, q)
		for i := 0; i < len(p); i++ {
			if p[i] != i {
				t.Errorf("q . p = %v\n", p)
				t.Errorf("     p: %v\n", p2)
				t.Errorf("     q: %v\n", q2)
				t.Errorf("    iq: %v\n", q)
			}
		}
	}
}

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
	"runtime"
	"testing"
	"testing/quick"
)

// Test that get/set implements a map from non-negative
// 64 bit ints to bool.
func TestGetSet(t *testing.T) {
	model := map[int64] bool {}
	bits  := NewBitVector(64)
	
	getSet := func(value int64) bool {
		if value < 0 {
			return true
		}
		
		// First test that the current value is correct.
		if model[value] != bits.Get(value) {
			return false
		}
		// Now add it to both sets and repeat the test as a sanity check.
		model[value] = true
		bits.Set(value, true)
		return bits.Get(value) == true
	}
	
	if err := quick.Check(getSet, nil); err != nil {
		t.Error(err)
	}
}

const (
	MaxIndex = (1 << 31) - 100
	// MaxTests = 1 << 24
	MaxTests = 1 << 22
)

// Benchmark a normal map set
func BenchmarkMapRandom(b *testing.B) {
	b.StopTimer()
	m := map[int64] bool {}
	b.StartTimer()
	
	k := rand.Intn(MaxIndex)
	m[int64(k)] = true
	for i := 1; i < b.N; i++ {
		prev := m[int64(k)]
		k = rand.Intn(MaxIndex)
		m[int64(k)] = prev
	}
}

func BenchmarkBitVectorRandom(b *testing.B) {
	b.StopTimer()
	bv := NewBitVector(64)
	b.StartTimer()
	
	k := rand.Intn(MaxIndex)
	bv.Set(int64(k), true)
	for i := 1; i < b.N; i++ {
		prev := bv.Get(int64(k))
		k = rand.Intn(MaxIndex)
		bv.Set(int64(k), prev)
	}
}

func BenchmarkMapSequential(b *testing.B) {
	b.StopTimer()
	m := map[int64] bool {}
	b.StartTimer()
	
	m[int64(0)] = true
	for i := 1; i < b.N; i++ {
		prev := m[int64(i - 1)]
		m[int64(i)] = prev
	}
}

func BenchmarkBitVectorSequential(b *testing.B) {
	b.StopTimer()
	bv := NewBitVector(64)
	b.StartTimer()
	
	bv.Set(int64(0), true)
	for i := 1; i < b.N; i++ {
		prev := bv.Get(int64(i - 1))
		bv.Set(int64(i), prev)
	}
}

func MapMemoryUsage(n int, t *testing.T) {
	var p0 runtime.MemStats
	runtime.ReadMemStats(&p0)
	
	m := map[int64]bool {}
	for i := 0; i < MaxTests; i++ {
		k := int64(rand.Intn(MaxIndex))
		m[k] = true
	}
	
	var p1 runtime.MemStats
	runtime.ReadMemStats(&p1)
	t.Logf("Map memory usage: %.2f MB\n", float64(p1.Alloc - p0.Alloc) / 1048576.0)
	t.Logf("Total memory allocated: %.2f MB\n", float64(p1.Sys - p0.Sys) / 1048576.0)
	capacity := nextPowerOf2(uint64(float64(n) * 1.1))
	t.Logf("Idealized: %.2f MB\n", float64(16 * capacity) / 1048576.0)
}

func BitVectorMemoryUsageRandom(n int, t *testing.T) {
	var p0 runtime.MemStats
	runtime.ReadMemStats(&p0)
	
	bv := NewBitVector(64)
	for i := 0; i < MaxTests; i++ {
		k := int64(rand.Intn(MaxIndex))
		bv.Set(k, true)
	}
	
	var p1 runtime.MemStats
	runtime.ReadMemStats(&p1)
	t.Logf("BitVector memory usage: %.2f MB\n", float64(p1.Alloc - p0.Alloc) / 1048576.0)
	t.Logf("Total memory allocated: %.2f MB\n", float64(p1.Sys - p0.Sys) / 1048576.0)
	t.Logf("Idealized: %.2f MB\n", float64(BitVectorDebugSize(bv)) / 1048576.0)
}

func BitVectorMemoryUsageSequential(n int, t *testing.T) {
	var p0 runtime.MemStats
	runtime.ReadMemStats(&p0)
	
	bv := NewBitVector(64)
	for i := 0; i < MaxTests; i++ {
		bv.Set(int64(i), true)
	}
	
	var p1 runtime.MemStats
	runtime.ReadMemStats(&p1)
	t.Logf("BitVector memory usage: %.2f MB\n", float64(p1.Alloc - p0.Alloc) / 1048576.0)
	t.Logf("Total memory allocated: %.2f MB\n", float64(p1.Sys - p0.Sys) / 1048576.0)
	t.Logf("Idealized: %.2f MB\n", float64(BitVectorDebugSize(bv)) / 1048576.0)
}

// You really don't want to run this test. It's quite memory intensive.
/*
func TestMemoryUsage(t *testing.T) {
	if !testing.Short() {
		runtime.GC()
		MapMemoryUsage(MaxTests, t)
		runtime.GC()
		BitVectorMemoryUsageSequential(MaxTests, t)
		runtime.GC()
		BitVectorMemoryUsageRandom(MaxTests, t)
	}
}
*/

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
	"io"
	"runtime"
)

const (
	// Update intervals
	ProfileAllocInterval = 500 << 20
	ProfileInuseInterval = 100 << 20
)

type ProfileRecord struct {
	Stack        []uintptr
	AllocObjects int
	InuseObjects int
	AllocBytes   int
	InuseBytes   int
}

var (
	// Output
	ProfilingEnabled = false
	ProfileRecords []ProfileRecord = nil

	// Heap State
	AllocObjects = 0
	InuseObjects = 0
	AllocBytes   = 0
	InuseBytes   = 0
	LastAlloc    = 0
	LastInuse    = 0
)

func add_sample(delta int) bool {
	InuseBytes += delta
	if delta > 0 {
		AllocObjects++
		InuseObjects++
		AllocBytes += delta
	} else {
		InuseObjects--
	}
	
	dump := false
	if AllocBytes > LastAlloc + ProfileAllocInterval {
		dump = true
	} else if InuseBytes > LastInuse + ProfileInuseInterval {
		dump = true
	} else if InuseBytes < LastInuse - ProfileInuseInterval {
		dump = true
	}
	
	if dump {
		LastAlloc = AllocBytes
		LastInuse = InuseBytes
		return true
	}
	return false
}

func profile_sample(delta int) {
	if !ProfilingEnabled || !add_sample(delta) {
		return
	}
	
	stk := make([]uintptr, 32)
	n := runtime.Callers(4, stk[:])
	
	record := ProfileRecord {
		Stack:        stk[:n],
		AllocObjects: AllocObjects,
		InuseObjects: InuseObjects,
		AllocBytes:   AllocBytes,
		InuseBytes:   InuseBytes,
	}
	
	// There is a divide by 0 in pprof if inuse/alloc = 0.
	if InuseObjects == 0 {
		record.InuseObjects++
	}
	if AllocObjects == 0 {
		record.AllocObjects++
	}
	
	ProfileRecords = append(ProfileRecords, record)
}

func ProfileAllocate(bytes int) {
	profile_sample(bytes)
}

func ProfileFree(bytes int) {
	profile_sample(-bytes)
}

func EnableProfiling(b bool) {
	ProfilingEnabled = b
}

func WriteProfile(w io.Writer) {
	if !ProfilingEnabled {
		return
	}
	
	totalInuseObjects := 0
	totalInuseBytes   := 0
	totalAllocObjects := 0
	totalAllocBytes   := 0
	
	for _, r := range ProfileRecords {
		totalInuseObjects += r.InuseObjects
		totalInuseBytes   += r.InuseBytes
		totalAllocObjects += r.AllocObjects
		totalAllocBytes   += r.AllocBytes
	}
	
	fmt.Fprintf(w, "heap profile: %d: %d [%d: %d] @ heapprofile/%d\n",
		totalInuseObjects, totalInuseBytes,
		totalAllocObjects, totalAllocBytes,
		2*ProfileInuseInterval)
	
	for _, r := range ProfileRecords {
		fmt.Fprintf(w, "%d: %d [%d: %d] @",
			r.InuseObjects, r.InuseBytes,
			r.AllocObjects, r.AllocBytes)
		for _, pc := range r.Stack {
			fmt.Fprintf(w, " %#x", pc)
		}
		fmt.Fprintf(w, "\n")
	}
}

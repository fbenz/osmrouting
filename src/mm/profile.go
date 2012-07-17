
package mm

import (
	"fmt"
	"io"
	"runtime"
)

const (
	// Update intervals
	ProfileAllocInterval = 5 << 20
	ProfileFreeInterval  = 5 << 20
	ProfileInuseInterval = 5 << 20
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
	AllocObjects     = 0
	InuseObjects     = 0
	AllocCount       = 0
	FreeCount        = 0
	LastAllocCount   = 0
	LastFreeCount    = 0
	LastInuseCount   = 0
)

func add_sample(delta int) bool {
	if delta > 0 {
		AllocObjects++
		InuseObjects++
		AllocCount += delta
	} else {
		InuseObjects--
		FreeCount  -= delta
	}
	InuseCount := AllocCount - FreeCount
	
	dump := false
	if AllocCount > LastAllocCount + ProfileAllocInterval {
		dump = true
	} else if FreeCount > LastFreeCount + ProfileFreeInterval {
		dump = true
	} else if InuseCount > LastInuseCount + ProfileInuseInterval {
		dump = true
	} else if InuseCount < LastInuseCount - ProfileInuseInterval {
		dump = true
	}
	
	if dump {
		LastAllocCount = AllocCount
		LastFreeCount  = FreeCount
		LastInuseCount = InuseCount
		return true
	}
	return false
}

func profile_sample(delta int) {
	if !ProfilingEnabled || !add_sample(delta) {
		return
	}
	
	stk := make([]uintptr, 32)
	n := runtime.Callers(2, stk[:])
	
	record := ProfileRecord {
		Stack:        stk[:n],
		AllocObjects: AllocObjects,
		InuseObjects: InuseObjects,
		AllocBytes:   AllocCount,
		InuseBytes:   AllocCount - FreeCount,
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
	
	fmt.Fprintf(w, "heap profile: %d: %d [%d: %d] @ heap/%d\n",
		InuseObjects, AllocCount - FreeCount,
		AllocObjects, AllocCount,
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

// TODO:
// - Add missing features:
//   * We need to parse relations, since these are used to encode
//     access restrictions between different roads.
//     Look at the university main entrance for a nice example.
//   * Obviously, we need max_speed information. However, this
//     is simply ridiculously convoluted.
//     max_speed is implicit for many roads and depends both
//     on the country and on whether or not the road lies
//     in a residential area. This means that we will have to
//     parse the corresponding relations and then do a few point
//     in polygon tests for any road without max_speed...

package main

import (
	"flag"
	"os"
	"osm"
	"strings"
	"runtime"
	"runtime/pprof"
)

var (
	// command line flags
	InputFile  string
	AccessType string
	CpuProfile string
	MemProfile string
)

func init() {
	flag.StringVar(&InputFile,  "i", "", "input pbf file")
	flag.StringVar(&AccessType, "f", "car", "access type (car, bike, foot or combinations, e.g. car,bike)")
	flag.StringVar(&CpuProfile, "cpuprofile", "", "write cpu profile to file")
	flag.StringVar(&MemProfile, "memprofile", "", "write memory profile to file")
	
	// The parser only uses 3 threads:
	// - one for disk reads + decompression
	// - another one for decoding the protocol buffers
	// - and the actual worker thread
	runtime.GOMAXPROCS(3)
}

func setup() (*os.File, osm.AccessType) {
	file, err := os.Open(InputFile)
	if err != nil {
		println("Unable to open input file:", err.Error())
		os.Exit(1)
	}

	var access osm.AccessType = 0
	for _, f := range strings.Split(AccessType, ",") {
		switch f {
		case "car":
			access |= osm.AccessMotorcar
		case "bike":
			access |= osm.AccessBicycle
		case "foot":
			access |= osm.AccessFoot
		default:
			println("Unrecognized access type:", access)
			os.Exit(1)
		}
	}
	
	return file, access
}

func main() {
	flag.Parse()
	
	if InputFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	if CpuProfile != "" {
		f, err := os.Create(CpuProfile)
		if err != nil {
			println("Unable to open cpuprofile:", err.Error())
			os.Exit(1)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	file, access := setup()
	
	println("Pass 1: Find the street graph.")
	graph, err := NewStreetGraph(file, access)
	if err != nil {
		println(err.Error())
		os.Exit(2)
	}

	println("Pass 2: Compute node attributes.")
	vertices, err := ComputeNodeAttributes(graph)
	if err != nil {
		println(err.Error())
		os.Exit(3)
	}

	println("Pass 3: Compute edge attributes.")
	err = ComputeEdgeAttributes(graph, vertices)
	if err != nil {
		println(err.Error())
		os.Exit(4)
	}
	
	// Write a memory profile for the most recent GC run.
	if MemProfile != "" {
		file, err := os.Create(MemProfile)
		if err != nil {
			println("Unable to open memprofile:", err.Error())
			os.Exit(5)
		}
		pprof.WriteHeapProfile(file)
		file.Close()
	}
}

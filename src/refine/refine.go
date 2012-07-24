
package main

import (
	"flag"
	"fmt"
	"graph"
	"math/rand"
	"mm"
	"os"
	"runtime/pprof"
)

var (
	// command line flags
	InputFile  string
	OutputFile string
	CpuProfile string
	MemProfile string
	RandomSeed int64
)

func init() {
	flag.StringVar(&InputFile,  "i", "", "input graph directory")
	flag.StringVar(&OutputFile, "o", "", "output graph directory")
	flag.StringVar(&CpuProfile, "cpuprofile", "", "write cpu profile to file")
	flag.StringVar(&MemProfile, "memprofile", "", "write memory profile to file")
	flag.Int64Var(&RandomSeed, "seed", 1, "random seed")
}

func main() {
	flag.Parse()
	
	if InputFile == "" {
		flag.Usage()
		os.Exit(1)
	}
	if InputFile == OutputFile {
		println("Input and output must not be equal.")
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
	
	if MemProfile != "" {
		f, err := os.Create(fmt.Sprintf("%s.mm.pprof", MemProfile))
		if err != nil {
			println("Unable to open memprofile:", err.Error())
			os.Exit(1)
		}
		mm.EnableProfiling(true)
		defer mm.WriteProfile(f)
	}

	rand.Seed(RandomSeed)

	println("Open input file.")
	g, _ := graph.OpenGraphFile(InputFile, true)
	println("Pass 1/2: Find the accessible subgraph.")
	subgraph := AccessibleRegion(g)
	println("Pass 2/2: Output the subgraph.")
	if OutputFile != "" {
		g.WriteInducedSubgraph(OutputFile, subgraph)
	}

	// Write a memory profile for the most recent GC run.
	if MemProfile != "" {
		file, err := os.Create(fmt.Sprintf("%s.go.pprof", MemProfile))
		if err != nil {
			println("Unable to open memprofile:", err.Error())
			os.Exit(5)
		}
		pprof.Lookup("heap").WriteTo(file, 0)
		file.Close()
	}
}

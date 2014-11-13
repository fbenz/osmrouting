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


package main

import (
	"flag"
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
		f, err := os.Create(CpuProfile + ".pprof")
		if err != nil {
			println("Unable to open cpuprofile:", err.Error())
			os.Exit(1)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	
	if MemProfile != "" {
		f, err := os.Create(MemProfile + ".mm.pprof")
		if err != nil {
			println("Unable to open memprofile:", err.Error())
			os.Exit(1)
		}
		mm.EnableProfiling(true)
		defer mm.WriteProfile(f)
	}

	rand.Seed(RandomSeed)

	println("Open input file. " + InputFile)
	g, _ := graph.OpenGraphFile(InputFile, true)
	println("Pass 1/2: Find the accessible subgraph.")
	subgraph := AccessibleRegion(g)
	println("Pass 2/2: Output the subgraph.")
	if OutputFile != "" {
		g.WriteInducedSubgraph(OutputFile, subgraph)
	}

	// Write a memory profile for the most recent GC run.
	if MemProfile != "" {
		file, err := os.Create(MemProfile + ".go.pprof")
		if err != nil {
			println("Unable to open memprofile:", err.Error())
			os.Exit(5)
		}
		pprof.Lookup("heap").WriteTo(file, 0)
		file.Close()
	}
}

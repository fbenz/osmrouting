package main

import (
	"fmt"
	"flag"
	"graph"
	"math/rand"
	"mm"
	"os"
	"route"
	"runtime/pprof"
	"time"
)

const (
	MaxSources = 5
	MaxTargets = 5
	MaxInitialWeight = 100
)

var (
	// command line flags
	InputFile    string
	CpuProfile   string
	MemProfile   string
	InputOverlay bool
	RandomSeed   int64
	NumRuns      int
)

func init() {
	flag.StringVar(&InputFile,  "i", "", "input graph directory")
	flag.StringVar(&CpuProfile, "cpuprofile", "", "write cpu profile to file")
	flag.StringVar(&MemProfile, "memprofile", "", "write memory profile to file")
	flag.BoolVar(&InputOverlay, "overlay", false, "open input as overlay graph")
	flag.Int64Var(&RandomSeed,  "seed", 1, "random seed")
	flag.IntVar(&NumRuns,       "runs", 1000, "number of iterations")
}

func OpenGraph(base string, overlay bool) graph.Graph {
	if !overlay {
		// This is a regular graph
		println("Open GraphFile")
		g, err := graph.OpenGraphFile(base, false /* ignoreErrors */)
		if err != nil {
			println(err.Error())
			os.Exit(1)
		}
		return g
	}
	
	// An overlay graph
	println("Open Overlay")
	g, err := graph.OpenOverlay(base, true /* loadMatrices */, false /* ignoreErrors */)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	return g
}

func BenchmarkBidirectional(g graph.Graph) {
	router := &route.BidiRouter {
		Transport: graph.Car,
		Metric:    graph.Distance,
	}
	
	duration := time.Duration(0)
	minDuration := time.Duration(time.Hour)
	maxDuration := time.Duration(0)
	for i := 0; i < NumRuns; i++ {
		numSources := rand.Intn(MaxSources)
		numTargets := rand.Intn(MaxTargets)
		t1 := time.Now()
		router.Reset(g)
		for j := 0; j < numSources; j++ {
			for {
				k := rand.Intn(g.VertexCount())
				if g.VertexAccessible(graph.Vertex(k), graph.Car) {
					router.AddSource(graph.Vertex(k), rand.Float32() * MaxInitialWeight)
					break
				}
			}
		}
		for j := 0; j < numTargets; j++ {
			for {
				k := rand.Intn(g.VertexCount())
				if g.VertexAccessible(graph.Vertex(k), graph.Car) {
					router.AddTarget(graph.Vertex(k), rand.Float32() * MaxInitialWeight)
					break
				}
			}
		}
		router.Run()
		diff := time.Since(t1)
		if diff > maxDuration {
			maxDuration = diff
		}
		if diff < minDuration {
			minDuration = diff
		}
		duration += diff
	}
	
	millis := float64(duration) / float64(time.Millisecond)
	maxMillis := float64(maxDuration) / float64(time.Millisecond)
	minMillis := float64(minDuration) / float64(time.Millisecond)
	fmt.Printf("Average Duration: %.2f ms\n", millis / float64(NumRuns))
	fmt.Printf("Maximum Duration: %.2f ms\n", maxMillis)
	fmt.Printf("Minimum Duration: %.2f ms\n", minMillis)
}

func main() {
	flag.Parse()
	
	if InputFile == "" {
		flag.Usage()
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

	g := OpenGraph(InputFile, InputOverlay)
	BenchmarkBidirectional(g)

	// Write a memory profile for the most recent GC run.
	if MemProfile != "" {
		file, err := os.Create(MemProfile + ".go.pprof")
		if err != nil {
			println("Unable to open memprofile:", err.Error())
			os.Exit(1)
		}
		pprof.Lookup("heap").WriteTo(file, 0)
		file.Close()
	}
}

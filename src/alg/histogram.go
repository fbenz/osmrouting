package alg

// Histogram handling and pretty printing.
// This originated in pbflint, but it turns out to be generally useful
// for debugging and gathering quick statistics.

import (
	"fmt"
	"sort"
)

type Histogram struct {
	Name string
	Data map[string] int
}

// Interface for sorting

type Sample struct {
	Frequency int
	Name      string
}

type Samples []Sample

func (s Samples) Len() int {
	return len(s)
}

func (s Samples) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Samples) Less(i, j int) bool {
	// Sort in descending order.
	return s[j].Frequency < s[i].Frequency
}

func NewHistogram(name string) *Histogram {
	return &Histogram{
		Name: name,
		Data: map[string] int {},
	}
}

func (h *Histogram) Add(sample string) {
	if _, ok := h.Data[sample]; ok {
		h.Data[sample]++
	} else {
		h.Data[sample] = 1
	}
}

func (h *Histogram) Samples() Samples {
	var s Samples = make([]Sample, 0)
	for key, frequency := range h.Data {
		s = append(s, Sample{frequency, key})
	}
	sort.Sort(s)
	return s
}

func (h *Histogram) Print() {
	fmt.Printf("Histogram for %s:\n", h.Name)
	total := 0
	for _, frequency := range h.Data {
		total += frequency
	}
	fmt.Printf(" - Sample Count: %d\n", total)
	fmt.Printf("=========================\n")
	samples := h.Samples()
	maxLen  := 1
	for _, sample := range samples {
		if len(sample.Name) > maxLen {
			maxLen = len(sample.Name)
		}
	}
	format := fmt.Sprintf(" %%%ds: %%d\n", maxLen)
	for _, sample := range samples {
		key := sample.Name
		frequency := sample.Frequency
		fmt.Printf(format, key, frequency)
	}
}

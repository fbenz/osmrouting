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

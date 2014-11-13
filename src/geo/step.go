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

package geo

import (
	"ellipsoid"
	"encoding/binary"
)

var (
	e ellipsoid.Ellipsoid
)

func init() {
	e = ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter,
		ellipsoid.Longitude_is_symmetric, ellipsoid.Bearing_is_symmetric)
}

// Steps are sequences of relative coordinates. The main point why you
// would want to use these instead of []Coordinate is that in practice
// we can compress them very well.

func EncodeStep(start Coordinate, step []Coordinate) []byte {
	// Minor annoyance: PutVarint will panic if the buffer is too small.
	// We have to allocate a large buffer here, end up copying the result
	// to a smaller buffer later on.
	buf := make([]byte, 2*binary.MaxVarintLen32*len(step))
	bufc := buf
	prevLat, prevLng := start.Encode()
	size := 0

	for _, curr := range step {
		currLat, currLng := curr.Encode()
		dlat := currLat - prevLat
		dlng := currLng - prevLng
		n := binary.PutVarint(bufc, int64(dlat))
		bufc = bufc[n:]
		m := binary.PutVarint(bufc, int64(dlng))
		bufc = bufc[m:]
		size += n + m
		prevLat, prevLng = currLat, currLng
	}

	// This happens in the parser, because we don't allocate steps on the go heap.
	//result := make([]byte, size)
	//copy(result, buf)
	//return result
	return buf[:size]
}

func DecodeStep(start Coordinate, step []byte, buf []Coordinate) []Coordinate {
	buf = buf[:0]
	prevLat, prevLng := start.Encode()
	b := step

	for len(b) > 0 {
		dlat, n := binary.Varint(b)
		b = b[n:]
		dlng, m := binary.Varint(b)
		b = b[m:]
		lat := prevLat + int32(dlat)
		lng := prevLng + int32(dlng)
		buf = append(buf, DecodeCoordinate(lat, lng))
		prevLat, prevLng = lat, lng
	}

	return buf
}

func StepLength(steps []Coordinate) float64 {
	if len(steps) < 2 {
		return 0
	}

	prev := steps[0]
	total := 0.0
	for _, step := range steps[1:] {
		distance, _ := e.To(prev.Lat, prev.Lng, step.Lat, step.Lng)
		total += distance
		prev = step
	}
	return total
}

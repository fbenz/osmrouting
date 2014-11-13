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
	"testing"
	"testing/quick"
)

func TestStepEncodeDecode(t *testing.T) {
	// The Generate method is defined in coordinate_test.go
	embedProject := func(start Coordinate, step []Coordinate) bool {
		// Encode it and Decode it again
		encoding := EncodeStep(start, step)
		decoded  := DecodeStep(start, encoding)
		for i, _ := range step {
			if !step[i].Equal(decoded[i]) {
				return false
			}
		}
		return true
	}
	
	if err := quick.Check(embedProject, nil); err != nil {
		t.Error(err)
	}
}

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

// The structure of the JSON object for the route response exactly as in the API definition

package route

const (
	StatusOk      = "OK"
	StatusNoRoute = "No route found"
)

type Result struct {
	BoundingBox BoundingBox `json:"boundingBox"`
	Routes      []Route     `json:"routes"`
}

type BoundingBox struct {
	Northwest Point `json:"nw"`
	Southeast Point `json:"se"`
}

type Route struct {
	Distance      Distance `json:"distance"`
	Duration      Duration `json:"duration"`
	StartLocation Point    `json:"start_location"`
	EndLocation   Point    `json:"end_location"`
	Legs          []Leg    `json:"legs"`
}

type Leg struct {
	Status        string   `json:"status"`
	Distance      Distance `json:"distance"`
	Duration      Duration `json:"duration"`
	StartLocation Point    `json:"start_location"`
	EndLocation   Point    `json:"end_location"`
	Steps         []Step   `json:"steps"`
}

type Step struct {
	Distance      Distance `json:"distance"`
	Duration      Duration `json:"duration"`
	StartLocation Point    `json:"start_location"`
	EndLocation   Point    `json:"end_location"`
	Polyline      Polyline `json:"polyline"`
	Instruction   string   `json:"instruction"`
}

type Distance struct {
	Text  string `json:"text"`
	Value int    `json:"value"`
}

type Duration struct {
	Text  string `json:"text"`
	Value int    `json:"value"`
}

type Point []float64

type Polyline []Point

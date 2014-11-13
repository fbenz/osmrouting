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

// The features supported by this implementation

package main

type Features struct {
	TravelMode TravelMode `json:"travelmode"`
	Metric     Metric     `json:"metric"`
	Avoid      Avoid      `json:"avoid"`
}

type TravelMode struct {
	Driving   bool `json:"driving"`
	Walking   bool `json:"walking"`
	Bicycling bool `json:"bicycling"`
}

type Metric struct {
	Distance bool `json:"distance"`
	Time     bool `json:"time"`
}

type Avoid struct {
	Ferries bool `json:"ferries"`
}

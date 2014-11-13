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

package osm

// All speeds are in km/h.

// Default maximum speeds *for Germany*. This is most certainly not correct
// for other countries. On the other hand, to be correct you would need to
// implement point in polygon tests and parse the country borders... and so
// on. It's a real mess and right now we ignore the problematic parts.
func DefaultMaxSpeed(way Way) float64 {
	// There is no maximum speed on motorroads, but 130 km/h is 'recommended'.
	if ParseBool(way.Attributes["motorroad"]) {
		return 130
	}
	
	// Designated cycling routes impose "moderate speeds" on drivers.
	if way.Attributes["bicycle"] == "designated" {
		return 30
	}
	
	// Ferries obviously have intrinsically different speeds, but the tags
	// are usually just plain missing. We go with an average speed of 12 km/h
	// here, because that happened to be correct for a sample size of 1 ferry.
	if way.Attributes["route"] == "ferry" {
		return 12
	}

	// Highway defaults
	switch way.Attributes["highway"] {
	case "motorway":
		return 130
	case "motorway_link":
		return 80
	// For living_street and pedestrian roads we know that the maximum
	// speed should be walking pace. In all other cases 
	case "living_street", "pedestrian",
		 "footway", "steps", "path", "track",
		 "service", "road":
		return 10 // <- "walking pace"
	case "cycleway":
		return 30 // <- not a clue.
	case "residential":
		return 50
	// These are all different depending on whether we are inside an urban
	// area. We don't have that information unless it is explicitly tagged,
	// so we default to the lower value.
	case "trunk", "primary", "primary_link",
	     "secondary", "secondary_link", "tertiary",
		 "unclassified", "tertiary_link":
		return 50
	}

	// No clue... be safe and designate it as walking speed.
	return 10
}

// Parse or make up a maximum speed for the given way.
func MaxSpeed(way Way) float64 {
	// If this way is not a road to begin with, ignore it.
	if _, ok := way.Attributes["highway"]; !ok {
		if _, ok := way.Attributes["junction"]; !ok {
			if way.Attributes["route"] != "ferry" {
				return 0
			}
		}
	}

	// Some roads are not actually built yet.
	// Normally, these are tagged as highway=construction|proposed, but it is
	// also permissible to tag it as construction=yes.
	if ParseBool(way.Attributes["construction"]) {
		return 0
	}

	// Try to parse the maxspeed tag, and if that fails, fall back to the
	// default values.
	if way.Attributes["maxspeed"] == "signals" {
		// useless.
		return DefaultMaxSpeed(way)
	} else if way.Attributes["maxspeed"] == "none" {
		// clamp to 130 km/h
		return 130
	}
	speed, err := ParseSpeed(way.Attributes["maxspeed"])
	if err != nil {
		return DefaultMaxSpeed(way)
	}
	return speed
}

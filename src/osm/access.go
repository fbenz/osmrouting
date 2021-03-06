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

// The different access types.
type AccessType int

const (
	AccessMotorcar AccessType = 1 << 0
	AccessBicycle             = 1 << 1
	AccessFoot                = 1 << 2
)

// The access hierarchy.
type AccessData struct {
	Key  string
	Mask AccessType
}

var AccessTable = [...]AccessData{
	{"access", AccessMotorcar | AccessBicycle | AccessFoot},
	{"foot", AccessFoot},
	{"vehicle", AccessMotorcar | AccessBicycle},
	{"bicycle", AccessBicycle},
	{"motor_vehicle", AccessMotorcar},
	{"motorcar", AccessMotorcar},
}

// Compute the access mask based on the default access restrictions for Germany.
// This function should probably take an additional argument to specify the country
// we're in.
func DefaultAccessMask(way Way) AccessType {
	// There is a special exceptional tag for motorroads.
	if ParseBool(way.Attributes["motorroad"]) {
		return AccessMotorcar
	}

	// Highway defaults
	switch way.Attributes["highway"] {
	// These roads are generally accessible
	case "trunk", "primary", "secondary",
		"tertiary", "unclassified", "residential",
		"living_street", "road", "trunk_link",
		"primary_link", "secondary_link", "tertiary_link":
		return AccessMotorcar | AccessBicycle | AccessFoot
	// The rest excludes some access types
	case "motorway", "motorway_link":
		return AccessMotorcar
	case "path", "track":
		return AccessBicycle | AccessFoot
	case "footway", "pedestrian", "steps":
		return AccessFoot
	case "cycleway":
		return AccessBicycle
	// These roads are either tagged explicitly or are too often
	// wrong to be of any use... we do the unsafe thing and assume
	// that any highway tagged as "service" is generally accessible.
	case "service":
		return AccessMotorcar | AccessBicycle | AccessFoot
	}

	// Ferry routes do not imply any special access permissions, but the
	// access tags are often missing. We assume that ferry => acces foot,bike,
	// otherwise you will never be able to travel to Sicily from mainland europe.
	if way.Attributes["route"] == "ferry" {
		return AccessBicycle | AccessFoot
	}

	return 0
}

// Compute the access mask for a given way.
func AccessMask(way Way) AccessType {
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

	mask := DefaultAccessMask(way)

	// The designated access tags are hirachical.
	// This means that more specific tags override the previous ones.
	for _, data := range AccessTable {
		if _, ok := way.Attributes[data.Key]; ok {
			if ParseBool(way.Attributes[data.Key]) {
				mask |= data.Mask
			} else {
				mask &= ^data.Mask
			}
		}
	}

	return mask
}

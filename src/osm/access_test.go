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

import (
	"encoding/json"
	"testing"
)

// Some more or less random ways from saarland.osm.pbf. These have been
// classified by hand.
var fixtures = [...]struct {
	string
	AccessType
}{
	{`{"Id":5403598,"Nodes":[22257161,22257122,22257108,22257012,22257026],
	"Attributes":{"TODO":"yes","highway":"secondary_link","oneway":"yes"}}`,
		AccessMotorcar | AccessBicycle | AccessFoot},
	{`{"Id":24522164,"Nodes":[267039373,1199429438,267039374,267039376,1278475993,
	267039386,1387572088,266653483],"Attributes":{"bicycle":"yes","foot":"yes",
	"highway":"road","motorcar":"no","motorcycle":"no","tracktype":"grade1"}}`,
		AccessBicycle | AccessFoot},
	{`{"Id":135156253,"Nodes":[1484573099,1484573103,1484573106,1484573105],
	"Attributes":{"highway":"proposed","name":"Im Erdbeerfeld","proposed":"residential"}}`,
		0},
	{`{"Id":24257834,"Nodes":[261076061,261076062,261076063,264162777],
	"Attributes":{"construction":"residential","highway":"construction",
	"lcn":"yes","name":"Moseluferpromenade"}}`,
		0},
	{`{"Id":5017034,"Nodes":[25694997,88836620],"Attributes":{"created_by":"Potlatch 0.9c",
	"cycleway":"track","highway":"cycleway"}}`,
		AccessBicycle},
	{`{"Id":24209826,"Nodes":[262139872,1631172244,262139084,1631172344,262139086],
	"Attributes":{"highway":"tertiary_link","maxspeed":"50","oneway":"yes"}}`,
		AccessMotorcar | AccessBicycle | AccessFoot},
	{`{"Id":14332085,"Nodes":[279157258,138160840],"Attributes":
	{"bridge":"yes","highway":"trunk","lanes":"1","layer":"1","motorroad":"yes",
	"oneway":"yes","ref":"L 252","surface":"asphalt"}}`,
		AccessMotorcar},
	{`{"Id":4165360,"Nodes":[23843924,21997049],"Attributes":
	{"created_by":"JOSM","highway":"footway","name":"Virchowstraße"}}`,
		AccessFoot},
	{`{"Id":6073541,"Nodes":[33989562,33989563,33989564,33989566,33989568,
	33989570,33989571],"Attributes":{"highway":"primary_link"}}`,
		AccessMotorcar | AccessBicycle | AccessFoot},
	{`{"Id":11970764,"Nodes":[107853043,107853107,107853111,107853123,1268523245,
	1268523149,1268523197,335931846],"Attributes":{"highway":"living_street",
	"name":"Parkstraße"}}`,
		AccessMotorcar | AccessBicycle | AccessFoot},
	{`{"Id":4065354,"Nodes":[500954,10705316,10705289,10705237,10705299,10705250,
	1444315922,1444315929,1444315932,10705225,10705220,10705266],
	"Attributes":{"highway":"service"}}`,
		AccessFoot}, // controversial
	{`{"Id":4065613,"Nodes":[502869,502872,502873,502874,502286],"Attributes":
	{"highway":"motorway_link","name":"Waldmohr","oneway":"yes"}}`,
		AccessMotorcar},
	{`{"Id":5016730,"Nodes":[25694165,25694170,25694171,25694172,1759747299,25694173,
	1759704557,25694176,1759704571,25694178,1759704583,25694179],"Attributes":
	{"highway":"track","tracktype":"grade1"}}`,
		AccessBicycle | AccessFoot},
	{`{"Id":4065530,"Nodes":[500712,500715,500717,500720,500722,500723,500725],
	"Attributes":{"cycleway":"track","highway":"primary","name":"Bexbacher Straße",
	"oneway":"yes","ref":"B 423"}}`,
		AccessMotorcar | AccessBicycle | AccessFoot},
	{`{"Id":28181776,"Nodes":[309546895,309546932,309546933,309546935,309546936,
	309546938,309546939,309546940,309546941,309546942,309546944,309546895],"Attributes":
	{"created_by":"Potlatch 0.10f","foot":"yes","highway":"bridleway"}}`,
		AccessFoot},
	{`{"Id":14332075,"Nodes":[1416042564,1416042568,1416042570,1416042590,1416042594,
	1416042602,138160724],"Attributes":{"fixme":"fix bus lines","highway":"trunk_link",
	"lanes":"1","oneway":"yes"}}`,
		AccessMotorcar | AccessBicycle | AccessFoot},
	{`{"Id":6057940,"Nodes":[49804867,49804842,49804847,49804851,49804854],"Attributes":
	{"created_by":"JOSM","highway":"steps","name":"Am Schloßberg"}}`,
		AccessFoot},
	{`{"Id":4634343,"Nodes":[22497118,700356592,10705223,17029290,17029285,413718663],
	"Attributes":{"bicycle":"designated","foot":"designated","highway":"secondary",
	"name":"Saarbrücker Straße","ref":"L 119"}}`,
		AccessMotorcar | AccessBicycle | AccessFoot},
	{`{"Id":4065364,"Nodes":[278238898,34592439,1387507711],"Attributes":{"access":"yes",
	"bicycle":"no","foot":"no","highway":"motorway","horse":"no","lanes":"2","lit":"no",
	"maxspeed":"60","motor_vehicle":"yes","oneway":"yes","ref":"A 8",
	"source:lit":"http://www.autobahn-bilder.de","surface":"asphalt"}}`,
		AccessMotorcar},
}

func TestAccessMask(t *testing.T) {
	for _, fixture := range fixtures {
		var way Way
		err := json.Unmarshal([]byte(fixture.string), &way)
		if err != nil {
			t.Fatalf("Could not unmarshal fixture: %s", fixture.string)
		}
		mask := AccessMask(way)
		if mask != fixture.AccessType {
			t.Errorf("Wrong access type %d (expected: %d) for way: %v\n",
				mask, fixture.AccessType, way)
		}
	}
}

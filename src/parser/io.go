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

// Wrappers around the io calls which perform some rudimentary
// error reporting and then abort the program.
package main

import (
	"log"
	"mm"
)

func Create(name string, size int, p interface{}) {
	err := mm.Create(name, size, p)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func Allocate(size int, p interface{}) {
	err := mm.Allocate(size, p)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func Close(p interface{}) {
	err := mm.Close(p)
	if err != nil {
		log.Fatal(err.Error())
	}
}

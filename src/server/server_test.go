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

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// How to use a test for profiling:
// go test -test.cpuprofile cpu.out
// go test -test.memprofile mem.out

// How to use our built-in route request profiling:
// ./server --cpuprofile=server.prof

// Interactive access to the profile:
// go tool pprof server server.prof

// Generate call graph:
// go tool pprof --svg server server.prof > prof.svg

func TestRoutes(t *testing.T) {
	if err := setup(); err != nil {
		t.Fatalf("Setup failed: %v", err.Error())
		return
	}

	respRecorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "http://x.de/routes?waypoints=49.2572069321567,7.04588517266191|49.2574019507051,7.04324261219973&travelmode=walking", nil /* body io.Reader */)
	routes(respRecorder, request)
	// TODO finish test
}

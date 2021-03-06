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

// The HTTP server processing route and feature requests

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"geo"
	"graph"
	"html/template"
	"io"
	"kdtree"
	"log"
	"net/http"
	"os"
	"route"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"
)

const (
	MaxThreads = 32

	ParameterWaypoints  = "waypoints"
	ParameterTravelmode = "travelmode"
	ParameterMetric     = "metric"
	ParameterAvoid      = "avoid"

	SeparatorWaypoints = "|"
	SeparatorLatLng    = ","

	DefaultPort = 23401

	TravelmodeCar  = "driving"
	TravelmodeFoot = "walking"
	TravelmodeBike = "bicycling"

	MetricDistance = "distance"
	MetricTime     = "time"
)

var (
	featureResponse []byte

	// command line flags
	FlagDir        string
	FlagPort       int
	FlagLogging    bool
	FlagCpuProfile string
	FlagCaching    bool

	startupTime time.Time

	clusterGraph *graph.ClusterGraph
)

func init() {
	flag.StringVar(&FlagDir, "dir", "", "base directory of the graph")
	flag.IntVar(&FlagPort, "port", DefaultPort, "the port where the server is running")
	flag.BoolVar(&FlagLogging, "logging", false, "enables logging of requests")
	flag.StringVar(&FlagCpuProfile, "cpuprofile", "", "enables CPU profiling")
	flag.BoolVar(&FlagCaching, "caching", false, "enables caching of route requests")
}

func main() {
	runtime.GOMAXPROCS(MaxThreads)
	log.Println("Starting...")

	// call the command line parser
	flag.Parse()

	if err := setup(); err != nil {
		log.Fatal("Setup failed: ", err)
	}

	// map URLs to functions
	http.HandleFunc("/", root)
	http.HandleFunc("/routes", routes)
	http.HandleFunc("/features", features)
	http.HandleFunc("/awesome", test)
	http.HandleFunc("/status", status)
	http.HandleFunc("/forward", forward)
	http.HandleFunc("/stop6bbw753i08wn1ca", stop)

	// start the HTTP server
	log.Println("Serving...")
	startupTime = time.Now()
	err := http.ListenAndServe(":"+strconv.Itoa(FlagPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// setup does some initialization before the HTTP server starts.
func setup() error {
	// Load the cluster graphs and the overlay graph as well as the
	// precomputed matrices for the metrics.
	var err error
	clusterGraph, err = graph.OpenClusterGraph(FlagDir, true /* load matrices */)
	if err != nil {
		return err
	}

	// Load the k-d trees for the cluster and the overlay graph. In addition,
	// the bounding boxes for the clusters are loaded.
	err = kdtree.LoadKdTree(clusterGraph, FlagDir)
	if err != nil {
		return err
	}

	if FlagLogging {
		InitLogger()
	}
	if FlagCaching {
		InitCache()
	}

	// Create the feature response only once (no change at runtime).
	supportedTravelmodes := TravelMode{Driving: true, Walking: true, Bicycling: true}
	supportedMetrics := Metric{Distance: true, Time: true}
	supportedRestrictions := Avoid{Ferries: false} // not implemented yet.
	supportedFeatures := &Features{
		TravelMode: supportedTravelmodes,
		Metric:     supportedMetrics,
		Avoid:      supportedRestrictions,
	}
	if fp, err := json.Marshal(supportedFeatures); err != nil {
		return err
	} else {
		// only assign if the creation was successful
		featureResponse = fp
	}
	return nil
}

// root just tells that the server is alive.
func root(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Server is up and running")
}

// routes returns routes according to the given parameters.
func routes(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// profiling if enabled
	if FlagCpuProfile != "" {
		f, err := os.Create(FlagCpuProfile)
		if err != nil {
			log.Fatal("Creating profile: ", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// parse URL and extract parameters
	urlParameter := r.URL.Query()

	// handle waypoints parameter
	if urlParameter[ParameterWaypoints] == nil || len(urlParameter[ParameterWaypoints]) < 1 {
		http.Error(w, "no waypoints", http.StatusBadRequest)
		return
	}
	waypoints, err := getWaypoints(urlParameter[ParameterWaypoints][0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// travel mode, using strings as constant
	travelmode := TravelmodeCar // Per default driving
	if urlParameter[ParameterTravelmode] != nil {
		if urlParameter[ParameterTravelmode][0] == TravelmodeCar ||
			urlParameter[ParameterTravelmode][0] == TravelmodeFoot ||
			urlParameter[ParameterTravelmode][0] == TravelmodeBike {
			travelmode = urlParameter[ParameterTravelmode][0]
		} else {
			http.Error(w, "wrong travelmode", http.StatusBadRequest)
			return
		}
	}
	transport := getTransport(travelmode)

	// Metrics
	metric := graph.Time
	if urlParameter[ParameterMetric] != nil {
		switch urlParameter[ParameterMetric][0] {
		case MetricDistance:
			metric = graph.Distance
		case MetricTime:
			// nothing to do here
		default:
			http.Error(w, "wrong metric", http.StatusBadRequest)
			return
		}
	}

	// Restrictions
	avoidFerries := false
	if urlParameter[ParameterAvoid] != nil {
		if urlParameter[ParameterAvoid][0] == "ferries" {
			avoidFerries = true
		} else {
			http.Error(w, "wrong avoid", http.StatusBadRequest)
			return
		}
	}

	cachingKey := urlParameter[ParameterWaypoints][0] + travelmode
	if FlagCaching {
		if resp, ok := CacheGet(cachingKey); ok {
			w.Write(resp)
			return
		}
	}

	// Do the actual route computation.
	planner := &route.RoutePlanner{
		Graph:           clusterGraph,
		Waypoints:       waypoints,
		Transport:       transport,
		Metric:          metric,
		AvoidFerries:    avoidFerries,
		ConcurrentKd:    true,
		ConcurrentLegs:  true,
		ConcurrentPaths: true,
	}
	result := planner.Run()

	endTime := time.Now()
	defer LogRequest(r, startTime, endTime)

	jsonResult, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "unable to create a proper JSON object", http.StatusInternalServerError)
		return
	}
	if FlagCaching {
		CachePut(cachingKey, jsonResult)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Write(jsonResult)
}

// getWaypoints parses the given waypoints.
func getWaypoints(waypointString string) ([]geo.Coordinate, error) {
	waypointStrings := strings.Split(waypointString, SeparatorWaypoints)
	if len(waypointStrings) < 2 {
		return nil, errors.New("too few waypoints. at least 2 waypoints are required")
	}

	points := make([]geo.Coordinate, len(waypointStrings))
	for i, v := range waypointStrings {
		coordinateStrings := strings.Split(v, SeparatorLatLng)
		if len(coordinateStrings) != 2 {
			return nil, errors.New("wrong formatted coordinate in waypoint list: " + v)
		}
		lat, err := strconv.ParseFloat(coordinateStrings[0], 64 /* bitSize */)
		if err != nil {
			return nil, errors.New("wrong formatted number in waypoint list: " + coordinateStrings[0])
		}
		lng, err := strconv.ParseFloat(coordinateStrings[1], 64 /* bitSize */)
		if err != nil {
			return nil, errors.New("wrong formatted number in waypoint list: " + coordinateStrings[1])
		}
		points[i] = geo.Coordinate{lat, lng}
	}
	return points, nil
}

func getTransport(travelmode string) graph.Transport {
	switch travelmode {
	case TravelmodeCar:
		return graph.Car
	case TravelmodeFoot:
		return graph.Foot
	case TravelmodeBike:
		return graph.Bike
	}
	return graph.Car
}

// features handles feature requests.
func features(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Write(featureResponse)
	LogRequest(r, startTime, time.Now())
}

// stop allows terminating the server a request.
func stop(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	LogRequest(r, startTime, time.Now())
	// wait 5 seconds so that the logger has time to write the request to file
	time.Sleep(5 * time.Second)

	os.Exit(1)
}

// forward redirects the routing request to another port. This is used by our test page so
// that we can work around the same origin policy.
func forward(w http.ResponseWriter, r *http.Request) {
	// only extract the "port" parameter
	urlParameter := r.URL.Query()
	if urlParameter["port"] == nil || len(urlParameter["port"]) < 1 {
		http.Error(w, "no port", http.StatusBadRequest)
		return
	}
	port := urlParameter["port"][0]

	forwardParameter := ""
	for k, v := range urlParameter {
		if k != "port" {
			forwardParameter += fmt.Sprintf("&%s=%s", k, v[0])
		}
	}
	// remove first &
	forwardParameter = forwardParameter[1:]
	resp, err := http.Get("http://urania.mpi-inf.mpg.de:" + port + "/routes?" + forwardParameter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	body := make([]byte, 1024)
	length := 1
	var readErr error
	for length > 0 {
		length, readErr = resp.Body.Read(body)
		if length != 0 && readErr != nil {
			http.Error(w, "error while reading response from remote server: "+readErr.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(body[:length])
	}
}

// status returns an HTML page with some status information about the server
func status(w http.ResponseWriter, r *http.Request) {
	statusInfo := make(map[string]string)
	statusInfo["startupTime"] = startupTime.Format(time.RFC3339 /* "2006-01-02T15:04:05Z07:00" */)
	uptime := time.Now().Sub(startupTime)
	hours := int64(uptime.Hours())
	minutes := int64(uptime.Minutes()) % 60
	statusInfo["uptimeHours"] = strconv.FormatInt(hours, 10 /* base */)
	statusInfo["uptimeMinutes"] = strconv.FormatInt(minutes, 10 /* base */)
	statusInfo["cacheCurrent"] = strconv.FormatInt(int64(cache.Size/1024), 10 /* base */)
	statusInfo["cacheMax"] = strconv.FormatInt(int64(MaxCacheSize/1024), 10 /* base */)

	if err := statusTemplate.Execute(w, statusInfo); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var statusTemplate = template.Must(template.New("status").Parse(statusTemplateHTML))

const statusTemplateHTML = `
<!DOCTYPE html>
<html>
<head>
  <title>Team FortyTwo Server Status</title>
</head>
<body>
  <h1>Team FortyTwo Server Status</h1>
  <p>Started: {{ .startupTime }}</p>
  <p>Uptime: {{ .uptimeHours }} h {{ .uptimeMinutes }} min</p>
  <p>Cache: {{ .cacheCurrent }} of {{ .cacheMax }} kB</p>
</body>
</html>
`

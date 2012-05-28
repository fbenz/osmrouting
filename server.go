
package main

/* The HTTP server waiting for requests */

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	ParameterWaypoints = "waypoints"
	ParameterTravelmode = "travelmode"
	ParameterMetric = "metric"
	ParameterAvoid = "avoid"

	SeparatorWaypoints = "|"
	SeparatorLatLng = ","

	DefaultPort = 23401 // the default port number
)

var (
	featureResponse []byte

	// command line flags
	Port int // the flag for the port
	Logging bool
)

func init(){
	flag.IntVar(&Port, "port", DefaultPort, "the port where the server is running")
	flag.BoolVar(&Logging, "logging", false, "enables logging of requests")
}

func main() {
	// call the command line parser
	flag.Parse()

	setupErr := setup()
	if setupErr != nil {
		log.Fatal("Setup failed:", setupErr)
		return
	}
	
	// map URLs to functions
	http.HandleFunc("/", root)
	http.HandleFunc("/routes", routes)
	http.HandleFunc("/features", features)

	// start the HTTP server
	err := http.ListenAndServe(":"+strconv.Itoa(Port), nil)
	if err != nil {
        log.Fatal("ListenAndServe:", err)
    }
}

// Do some stuff when the server is started
func setup() error {
	InitLogger()

	// create the feature response only once (no change at runtime)
	features := &Features{}
	var err error
	featureResponse, err = json.Marshal(features)
	if err != nil {
		log.Fatal("Creating feature response:", err)
		return err
	}
	return nil
}

// Just tell that the server is alive
func root(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Server is up and running")
}

// Computes routes (at the moment only one route is returned statically)
func routes(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer LogRequest(r, startTime)

	// parse URL and extract parameters
	urlParameter := r.URL.Query()
	
	// handle waypoints parameter
	if urlParameter[ParameterWaypoints] == nil || len(urlParameter[ParameterWaypoints]) < 1 {
		http.Error(w, "no waypoints", http.StatusBadRequest)
		return
	}
	_, err := getWaypoints(urlParameter["waypoints"][0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// there is no need to handle the other parameters at the moment as
	// the implementation should not fail for unknown parameters/values

	// hard coded values for the route response
	var polyline1 = [][]float64{{49.25708,7.045980000000001},{49.257070000000006,7.045960000000001},{49.25652,7.044390000000001}}
	distance1 := Distance{"0.1 km", 131}
	duration1 := Duration{"2 min", 124}
	startLocation1 := Point{49.257080, 7.045980000000001}
	endLocation1 := Point{49.256520, 7.044390000000001}
	step1 := Step{distance1, duration1, startLocation1, endLocation1, polyline1, ""}
	
	var polyline2 = [][]float64{{49.25652,7.044390000000001},{49.25661,7.0444},{49.25668,7.044390000000001},{49.25674,7.044320000000001},{49.25677,7.044300000000001},{49.256800000000005,7.044270000000001},{49.256820000000005,7.044230000000001},{49.256840000000004,7.0442100000000005},{49.256870000000006,7.04419},{49.25688,7.04415},{49.256910000000005,7.0441400000000005},{49.25694000000001,7.044110000000001},{49.256980000000006,7.044060000000001},{49.25703000000001,7.0440000000000005},{49.25706,7.043950000000001},{49.257090000000005,7.043940000000001},{49.25704,7.043310000000001}}
	distance2 := Distance{"0.1 km", 122}
	duration2 := Duration{"2 min", 136}
	startLocation2 := Point{49.256520, 7.044390000000001}
	endLocation2 := Point{49.257040, 7.043310000000001}
	step2 := Step{distance2, duration2, startLocation2, endLocation2, polyline2, ""}
	
	distanceL := Distance{"0.3 km", 253}
	durationL := Duration{"4 min", 260}
	startLocationL := Point{49.257080, 7.045980000000001}
	endLocationL := Point{49.257040, 7.043310000000001}
	steps := []Step{step1, step2}
	leg := Leg{distanceL, durationL, startLocationL, endLocationL, steps}
	
	legs := []Leg{leg}
	route := Route{distanceL, durationL, startLocationL, endLocationL, legs}
	routes := []Route{route}
	
	northwest := Point{49.25709000000001, 7.043310000000001}
	southeast := Point{49.256520, 7.045980000000001}
	boundingBox := BoundingBox{northwest, southeast}
	result := &Result{boundingBox, routes}
	jsonResult, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "unable to create a proper JSON object", http.StatusInternalServerError)
		return
	}
	w.Write(jsonResult)
}

// Parses the given waypoints
func getWaypoints(waypointString string) ([]Point, error) {
	waypointStrings := strings.Split(waypointString, SeparatorWaypoints)
	if len(waypointStrings) < 2 {
		return nil, errors.New("too few waypoints. at least 2 waypoints are needed")
	}

	points := make([]Point, len(waypointStrings))
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
		points[i] = Point{lat, lng}
	}
	return points, nil
}

// Handles feature requests
func features(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer LogRequest(r, startTime)

	w.Write(featureResponse)
}


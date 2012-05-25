
package main

/* The HTTP server waiting for route requests */

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const (
	ParameterWaypoints = "waypoints"

	SeparatorWaypoints = "|"
	SeparatorLatLng = ","
)

var (
	featureResponse []byte
)

func main() {
	http.HandleFunc("/", root)
    http.HandleFunc("/routes", routes)
    http.HandleFunc("/features", features)
    
    features := &Features{}
	featureResponse, _ = json.Marshal(features)
	// TODO
	// jsonErr
	/*if jsonErr != nil {
        log.Fatal("Creating feature response:", jsonErr)
    }*/

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
        log.Fatal("ListenAndServe:", err)
    }
}

func root(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Server is up and running")
}

func routes(w http.ResponseWriter, r *http.Request) {
	// parse URL and extract parameters
	urlParameter := r.URL.Query()
	
	// handle waypoints
	if urlParameter[ParameterWaypoints] == nil || len(urlParameter[ParameterWaypoints]) < 1 {
		http.Error(w, "no waypoints", http.StatusBadRequest)
		return
	}
	waypoints, err := getWaypoints(urlParameter["waypoints"][0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// TODO handle the other parameters

	var polyline1 = [][]float64{{49.25708,7.045980000000001},{49.257070000000006,7.045960000000001},{49.25652,7.044390000000001},{49.256420000000006,7.0441400000000005},{49.256370000000004,7.04396},{49.25634,7.04384},{49.25632,7.043760000000001},{49.25632,7.04368},{49.25632,7.04361},{49.256310000000006,7.04351},{49.256310000000006,7.043430000000001},{49.256310000000006,7.043290000000001},{49.25632,7.043130000000001},{49.256330000000005,7.042980000000001},{49.25618,7.04302},{49.25573000000001,7.043010000000001}}
	distance1 := Distance{"0.3 km", 272}
	duration1 := Duration{"1 min", 43}
	startLocation1 := Point{49.257080, 7.045980000000001}
	endLocation1 := Point{49.256370, 7.042530}
	step1 := Step{distance1, duration1, startLocation1, endLocation1, polyline1, ""}
	
	var polyline2 = [][]float64{{49.256370000000004,7.04253},{49.256980000000006,7.0424500000000005},{49.25706,7.042440000000001},{49.257250000000006,7.04243}}
	distance2 := Distance{"0.1 km", 98}
	duration2 := Duration{"1 min", 20}
	startLocation2 := Point{49.256370,7.042530}
	endLocation2 := Point{49.25725000000001,7.042430}
	step2 := Step{distance2, duration2, startLocation2, endLocation2, polyline2, ""}
	
	distanceL := Distance{"0.4 km", 370}
	durationL := Duration{"1 min", 63}
	startLocationL := Point{49.257080,7.045980000000001}
	endLocationL := Point{49.25725000000001, 7.042430}
	steps := []Step{step1, step2}
	leg := Leg{distanceL, durationL, startLocationL, endLocationL, steps}
	
	legs := []Leg{leg}
	route := Route{distanceL, durationL, startLocationL, endLocationL, legs}
	routes := []Route{route}
	
	northwest := Point{49.25725000000001, 7.042430 + waypoints[0].Lat - waypoints[0].Lat}
	southeast := Point{49.256320, 7.045980000000001}
	boundingBox := BoundingBox{northwest, southeast}
	result := &Result{boundingBox, routes}
	jsonResult, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "unable to create a proper JSON object", http.StatusInternalServerError)
		return
	}
	w.Write(jsonResult)
}

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

func features(w http.ResponseWriter, r *http.Request) {
	w.Write(featureResponse)
}


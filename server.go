
package main

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

func main() {
	http.HandleFunc("/", root)
    http.HandleFunc("/routes", routes)

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

	
	// create result, TODO do it right ;)
	northwest := waypoints[0]
	southeast := waypoints[1]
	boundingBox := BoundingBox{northwest, southeast}
	result := &Result{boundingBox, nil}
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

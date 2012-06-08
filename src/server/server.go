// The HTTP server processing route and feature requests

package main

import (
	"alg"
	"encoding/json"
	"errors"
	"flag"
	"graph"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	ParameterWaypoints  = "waypoints"
	ParameterTravelmode = "travelmode"
	ParameterMetric     = "metric"
	ParameterAvoid      = "avoid"

	SeparatorWaypoints = "|"
	SeparatorLatLng    = ","

	DefaultPort = 23401 // the default port number
)

var (
	featureResponse []byte

	// command line flags
	FlagPort    int
	FlagLogging bool

	startupTime time.Time
	
	osmGraph graph.Graph
)

func init() {
	flag.IntVar(&FlagPort, "port", DefaultPort, "the port where the server is running")
	flag.BoolVar(&FlagLogging, "logging", false, "enables logging of requests")
}

func main() {
	runtime.GOMAXPROCS(8)
	log.Println("Starting...")

	// call the command line parser
	flag.Parse()

	if err := setup(); err != nil {
		log.Fatal("Setup failed:", err)
		return
	}

	// map URLs to functions
	http.HandleFunc("/", root)
	http.HandleFunc("/routes", routes)
	http.HandleFunc("/features", features)
	http.HandleFunc("/awesome", test)
	http.HandleFunc("/status", status)
	http.HandleFunc("/stop6bbw753i08wn1ca", stop)

	// start the HTTP server
	log.Println("Serving...")
	startupTime = time.Now()
	err := http.ListenAndServe(":"+strconv.Itoa(FlagPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

// setup does some initialization before the HTTP server starts.
func setup() error {
	// at the moment all files have to be in the same folder as the server executable
	if g, err := graph.Open("" /* base */); err != nil {
		log.Fatal("Loading graph:", err)
		return err
	} else {
		osmGraph = g
	}
	
	if err := alg.LoadKdTree(osmGraph.(graph.Positions)); err != nil {
		log.Fatal("Loading k-d tree:", err)
		return err
	}

	InitLogger()

	// create the feature response only once (no change at runtime)
	if fp, err := json.Marshal(&Features{}); err != nil {
		log.Fatal("Creating feature response:", err)
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

// routes returns routes according to the given parameters. (at the moment only one route is returned statically)
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
	waypoints, err := getWaypoints(urlParameter["waypoints"][0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// there is no need to handle the other parameters at the moment as
	// the implementation should not fail for unknown parameters/values
	legs := make([]Leg, len(waypoints) - 1)
	distance := 0.0
	duration := 0.0
	for i := 0; i < len(waypoints) - 1; i++ {
		/*startStep*/_, startWays := alg.NearestNeighbor(waypoints[i][0], waypoints[i][1], true /* forward */)
		/*endStep*/_, endWays := alg.NearestNeighbor(waypoints[i+1][0], waypoints[i+1][1], false /* forward */)
		
		/*
		log.Printf("Start: %v\n", startStep)
		log.Printf("End:   %v\n", endStep)
		log.Printf("Number of start points: %d\n", len(startWays))
		log.Printf("Number of end points: %d\n", len(endWays))
		*/
		
		dist, vertices, edges, start, end := alg.Dijkstra(startWays, endWays)
		legs[i] = PathToLeg(dist,vertices,edges,start,end)
		distance += float64(legs[i].Distance.Value)
		duration += float64(legs[i].Duration.Value)
	}
	
	route := Route{
		Distance: FormatDistance(distance),
		Duration: FormatDuration(duration),
		StartLocation: legs[0].StartLocation,
		EndLocation: legs[len(legs)-1].EndLocation,
		Legs: legs,
	}
	
	result := &Result{
		BoundingBox: ComputeBounds(route),
		Routes:      []Route{route},
	}
	
	jsonResult, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "unable to create a proper JSON object", http.StatusInternalServerError)
		return
	}
	w.Write(jsonResult)
}

// getWaypoints parses the given waypoints.
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

// features handles feature requests.
func features(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer LogRequest(r, startTime)

	w.Write(featureResponse)
}

// stop The server can be terminated with a request.
func stop(w http.ResponseWriter, r *http.Request) {
	os.Exit(1)
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
</body>
</html>
`

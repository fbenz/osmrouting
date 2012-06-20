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
	"kdtree"
	"log"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"
	
	//"fmt"
)

const (
	ParameterWaypoints  = "waypoints"
	ParameterTravelmode = "travelmode"
	ParameterMetric     = "metric"
	ParameterAvoid      = "avoid"

	SeparatorWaypoints = "|"
	SeparatorLatLng    = ","

	DefaultPort = 23401 // the default port number

	TravelmodeCar = "driving"
	TravelmodeFoot = "walking"
	TravelmodeBike = "bicycling"
)

type RoutingData struct {
	graph  graph.Graph
	kdtree *kdtree.KdTree
}

var (
	featureResponse []byte

	// command line flags
	FlagPort    	int
	FlagLogging 	bool
	FlagCpuProfile 	string

	startupTime time.Time
	
	osmData map[string] RoutingData
)

func init() {
	flag.IntVar(&FlagPort, "port", DefaultPort, "the port where the server is running")
	flag.BoolVar(&FlagLogging, "logging", false, "enables logging of requests")
	flag.StringVar(&FlagCpuProfile, "cpuprofile", "", "enables CPU profiling")
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
	http.HandleFunc("/forward", forward)
	http.HandleFunc("/stop6bbw753i08wn1ca", stop)

	// start the HTTP server
	log.Println("Serving...")
	startupTime = time.Now()
	err := http.ListenAndServe(":" + strconv.Itoa(FlagPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func loadFiles(base string) (*RoutingData, error) {
	g, err := graph.Open(base)
	if err != nil {
		log.Fatal("Loading graph: ", err)
		return nil, err
	}
	t, err := alg.LoadKdTree(base, g.Positions());
	if  err != nil {
		log.Fatal("Loading k-d tree: ", err)
		return nil, err
	}
	return &RoutingData{g, t}, nil
}

// setup does some initialization before the HTTP server starts.
func setup() error {
	osmData = map[string] RoutingData {}
	
	dat, err := loadFiles("car")
	if err != nil {
		return err
	}
	osmData["driving"] = *dat
	
	dat, err = loadFiles("bike")
	if err != nil {
		return err
	}
	osmData["bike"] = *dat // <- look this up.
	
	dat, err = loadFiles("foot")
	if err != nil {
		return err
	}
	osmData["walking"] = *dat

	InitLogger()

	// create the feature response only once (no change at runtime)
	supportedTravelmodes := TravelMode{Driving: true, Walking: true, Bicycling: true}
	supportedFeatures := &Features{TravelMode: supportedTravelmodes}
	if fp, err := json.Marshal(supportedFeatures); err != nil {
		log.Fatal("Creating feature response: ", err)
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
	waypoints, err := getWaypoints(urlParameter["waypoints"][0])
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
	
	// there is no need to handle the other parameters at the moment as
	// the implementation should not fail for unknown parameters/values
	data := osmData[travelmode]
	legs := make([]Leg, len(waypoints) - 1)
	distance := 0.0
	duration := 0.0
	for i := 0; i < len(waypoints) - 1; i++ {
		_, startWays := alg.NearestNeighbor(data.kdtree, waypoints[i][0],   waypoints[i][1],   true /* forward */)
		_, endWays   := alg.NearestNeighbor(data.kdtree, waypoints[i+1][0], waypoints[i+1][1], false /* forward */)
		
		//stD := time.Now()
		dist, vertices, edges, start, end := alg.Dijkstra(data.graph, startWays, endWays)
		//dTime := time.Now().Sub(stD)
		//fmt.Printf("dijkstra: %v\n", dTime.Nanoseconds()/1000)
		
		legs[i] = PathToLeg(data.graph, dist, vertices, edges, start, end)
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
	startTime := time.Now()
	LogRequest(r, startTime)
	// wait 5 seconds so that the logger has time to write the request to file
	time.Sleep(5 * time.Second)

	os.Exit(1)
}

// forward redirects the routing request to another port
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
			forwardParameter += "&" + k + "=" + v[0]
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
			http.Error(w, "error while reading response from remote server: " + readErr.Error(), http.StatusInternalServerError)
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

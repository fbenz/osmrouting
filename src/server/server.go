// The HTTP server processing route and feature requests

package main

import (
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
	"route"
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

	TravelmodeCar  = "driving"
	TravelmodeFoot = "walking"
	TravelmodeBike = "bicycling"
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
	runtime.GOMAXPROCS(8)
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
	clusterGraph, err = graph.OpenClusterGraph(FlagDir, true /* loadMatrices */)
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

	cachingKey := urlParameter[ParameterWaypoints][0] + travelmode
	if FlagCaching {
		if resp, ok := CacheGet(cachingKey); ok {
			w.Write(resp)
			return
		}
	}

	// there is no need to handle the other parameters at the moment as
	// the implementation should not fail for unknown parameters/values

	// Do the actual route computation.
	result := route.Routes(clusterGraph, waypoints, graph.Distance, transport)

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
func getWaypoints(waypointString string) ([]route.Point, error) {
	waypointStrings := strings.Split(waypointString, SeparatorWaypoints)
	if len(waypointStrings) < 2 {
		return nil, errors.New("too few waypoints. at least 2 waypoints are needed")
	}

	points := make([]route.Point, len(waypointStrings))
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
		points[i] = route.Point{lat, lng}
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
	defer LogRequest(r, startTime)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
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

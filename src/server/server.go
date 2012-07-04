// The HTTP server processing route and feature requests

package main

import (
	"alg"
	"encoding/json"
	"errors"
	"flag"
	"geo"
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
	FlagCaching		bool

	startupTime time.Time
	
	osmData map[string] RoutingData
)

func init() {
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
	osmData[TravelmodeCar] = *dat
	
	dat, err = loadFiles("bike")
	if err != nil {
		return err
	}
	osmData[TravelmodeBike] = *dat
	
	dat, err = loadFiles("foot")
	if err != nil {
		return err
	}
	osmData[TravelmodeFoot] = *dat

	if FlagLogging {
		InitLogger()
	}
	if FlagCaching {
		InitCache()
	}

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
	
	cachingKey := urlParameter[ParameterWaypoints][0] + travelmode
	if FlagCaching {
		if resp, ok := CacheGet(cachingKey); ok {
			w.Write(resp)
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
		_, startWays := alg.NearestNeighbor(data.kdtree, waypoints[i][0],   waypoints[i][1],   true  /* forward */)
		_, endWays   := alg.NearestNeighbor(data.kdtree, waypoints[i+1][0], waypoints[i+1][1], false /* forward */)
		allequal:=true
		oneequal:=false
		if len(startWays) != len(endWays){
			allequal = false
		}
		for _,startPoint:=range(startWays) {
			existequal:=false
			for _,endPoint:=range(endWays) {
				existequal = existequal || (startPoint.Node == endPoint.Node)
			}
			oneequal = oneequal || existequal
			allequal = allequal && existequal
		}
		//Start and Endpoint lie on the same edge
		if allequal {
			//Start node == End node
			if len(startWays) == 1 {
				polyline:= make([]Point,1)
				startpoint := Point{startWays[0].Target.Lat,startWays[0].Target.Lng}
				polyline[0]=startpoint
				instruction:="Stay at Point" // Mockup describtion
				duration:=Duration{"0.00 secs",0.0}
				distance:=Distance{"0.00 mm",0.0}
				step:=Step{distance,duration,startpoint,startpoint,polyline,instruction}
				steps:=make([]Step,1)
				steps[0]=step
				legs[i]=Leg{distance,duration,startpoint,startpoint,steps}
			} else { // Start and End node are on the same edge
				var correctStartWay,correctEndWay graph.Way
				S:
					for _,startPoint:=range(startWays) {
						for _,endPoint:=range(endWays) {
							if startPoint.Node == endPoint.Node && (startPoint.Length-endPoint.Length)>0 {
								correctStartWay=startPoint
								correctEndWay=endPoint
								break S
							}
						}
					}
				length:=correctStartWay.Length - correctEndWay.Length
				polyline:=make([]graph.Step,(len(correctStartWay.Steps) - len(correctEndWay.Steps)))
				// Find the steps from start to endpoint
				startsteps:=correctStartWay.Steps
				for i:=0;startsteps[i]!=correctEndWay.Steps[len(correctEndWay.Steps)-1];i++{
					polyline=append(polyline,startsteps[i])
				}
				step:=PartwayToStep(polyline,correctStartWay.Target,correctEndWay.Target,length)
				steps:=make([]Step,1)
				steps[0]=step
				legs[i]=Leg{step.Distance,step.Duration,step.StartLocation,step.EndLocation,steps}
				
				
			}
		} else if oneequal{
			if len(startWays)==1 { // If the end node is on the edge outgoing from s
				var correctEndWay graph.Way
				for _,i:=range(endWays) {
					if i.Node==startWays[0].Node {
						correctEndWay = i
						break
					}
				}
				n:=len(correctEndWay.Steps)
				polyline:=make([]graph.Step,n)
				for i,item := range(correctEndWay.Steps){
					polyline[n-i-1]=item
				}
				step:=PartwayToStep(polyline,startWays[0].Target,correctEndWay.Target,correctEndWay.Length)
				steps:=make([]Step,1)
				steps[0]=step
				legs[i]=Leg{step.Distance,step.Duration,step.StartLocation,step.EndLocation,steps}
			} else if len(endWays)==1 { // If the start node is on the edge outgoint from e
				var correctStartWay graph.Way
				for _,i:=range(startWays) {
					if i.Node==endWays[0].Node {
						correctStartWay =i
						break
					}
				}
				step:=PartwayToStep(correctStartWay.Steps,correctStartWay.Target,endWays[0].Target,correctStartWay.Length)
				steps:=make([]Step,1)
				steps[0]=step
				legs[i]=Leg{step.Distance,step.Duration,step.StartLocation,step.EndLocation,steps}
			} else { // we have s->u->e so they are on adjacent edges.
				var correctStartWay,correctEndWay graph.Way
				for _,i:=range(startWays) {
					for _,j:=range(endWays) {
						if i.Node==j.Node {
							correctStartWay=i
							correctEndWay=j
						}
					}
				}
				step1:=PartwayToStep(correctStartWay.Steps,correctStartWay.Target,NodeToStep(data.graph,correctStartWay.Node),correctStartWay.Length)
				step2:=PartwayToStep(correctEndWay.Steps,NodeToStep(data.graph,correctEndWay.Node),correctEndWay.Target,correctEndWay.Length)
				steps:=make([]Step,2)
				steps[0]=step1
				steps[1]=step2
				length:=correctStartWay.Length+correctEndWay.Length
				legs[i]=Leg{FormatDistance(length),MockupDuration(length),step1.StartLocation,step2.EndLocation,steps}
			}
		} else {
		// Use the Dijkatrs version using a large slice only for long roues where the map of the
		// other version can get quite large
			if getDistance(data.graph, startWays[0].Node, endWays[0].Node) > 100.0 * 1000.0 { // > 100km
				dist, vertices, edges, start, end := alg.DijkstraSlice(data.graph, startWays, endWays)
				legs[i] = PathToLeg(data.graph, dist, vertices, edges, start, end)
			} else {
				dist, vertices, edges, start, end := alg.Dijkstra(data.graph, startWays, endWays)
				legs[i] = PathToLeg(data.graph, dist, vertices, edges, start, end)
			}
			distance += float64(legs[i].Distance.Value)
			duration += float64(legs[i].Duration.Value)
		}
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
	if FlagCaching {
		CachePut(cachingKey, jsonResult)
	}
	w.Write(jsonResult)
}

// getDistance returns the distance between the two given nodes
func getDistance(g graph.Graph, n1 graph.Node, n2 graph.Node) float64 {
	lat1, lng1 := g.NodeLatLng(n1)
	lat2, lng2 := g.NodeLatLng(n2)
	return geo.Distance(geo.Coordinate{Lat: lat1, Lng: lng1}, geo.Coordinate{Lat: lat2, Lng: lng2})
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
	statusInfo["cacheCurrent"] = strconv.FormatInt(int64(cache.Size / 1024), 10 /* base */)
	statusInfo["cacheMax"] = strconv.FormatInt(int64(MaxCacheSize / 1024), 10 /* base */)

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

// Proof of concept for delta compressing int32 encoded steps
// The compression factor is around 0.63

// TODO We still have 0s in the data and thus are taking care of them. Solve the actual problem!
// TODO Not optimized for performance yet

package main

import (
	"code.google.com/p/snappy-go/snappy"
	"compress/flate"
	"alg"
	"graph"
	"log"
	"flag"
	"fmt"
	"bytes"
	"math"
	"encoding/binary"
	"time"
	"geo"
	"os"
)

const (
	TravelmodeCar = "driving"
	TravelmodeFoot = "walking"
	TravelmodeBike = "bicycling"
	
	OriginalStepSize = 2 * 4 // in bytes
	
	// From the geo package:
	// OsmEpsilon is the smallest difference between two osm coordinates.
	OsmEpsilon = 1e-7
	// The inverse of OsmEpsilon
	OsmPrecision = 1e7
)

type RoutingData struct {
	graph  graph.Graph
}

var (	
	osmData map[string] RoutingData
	FlagMode string
)

func init() {
	flag.StringVar(&FlagMode, "mode", "driving", "travel mode")
}

// adjusted version from the geo package
func Encode(lat, lng float64) (int32, int32) {
	latInt := int32(math.Floor(lat * OsmPrecision + 0.5))
	lngInt := int32(math.Floor(lng * OsmPrecision + 0.5))
	return latInt, lngInt
}

// adjusted version from the geo package
func Decode(latInt, lngInt int32) (float64, float64) {
	lat := float64(latInt) / OsmPrecision
	lng := float64(lngInt) / OsmPrecision
	return lat, lng
}

func main() {
	flag.Parse()

	if err := setup(); err != nil {
		log.Fatal("Setup failed:", err)
		return
	}
	fmt.Printf("encode and check...\n")
	//encodeAndCheck()
	//fmt.Printf("encode and check 2...\n")
	encodeAndCheck2()
}

func loadFiles(base string) (*RoutingData, error) {
	g, err := graph.Open(base)
	if err != nil {
		log.Fatal("Loading graph: ", err)
		return nil, err
	}
	return &RoutingData{g}, nil
}

// setup from the HTTP server
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
	osmData["bicycling"] = *dat
	
	dat, err = loadFiles("foot")
	if err != nil {
		return err
	}
	osmData["walking"] = *dat

	return nil
}

func encodeAndCheck() {
	g := osmData[FlagMode].graph
	
	encoded := make([][]byte, g.EdgeCount())
		
	bytesOrg := 0
	bytesNew := 0
	bytesSaved := 0
	
	time1 := time.Now()
	
	// Encode
	for j := 0; j < g.EdgeCount(); j++ {
		/*if j % 1000 == 0 {
			process := float64(j) / float64(g.EdgeCount())
			fmt.Printf("process %v\n", process)
		}*/

		// don't consider steps twice
		if revI, exists := g.EdgeReverse(graph.Edge(j)); exists && int(revI) < j {
			continue
		}

		initialBuf := make([]byte, 0, 8 /* minimum needed */)
		buf := bytes.NewBuffer(initialBuf)
		b := make([]byte, 4)
		steps := g.EdgeSteps(graph.Edge(j))
		if len(steps) == 0 {
			continue
		}
		
		continueBecauseZero := false
		lat, lng := Encode(steps[0].Lat, steps[0].Lng)
		for i, step := range steps {
			if step.Lat == 0 || step.Lng == 0 {
				continueBecauseZero = true
				break
			}
			if i == 0 {
				PutUint32(b, uint32(lat))
				buf.Write(b)
				PutUint32(b,  uint32(lng))
				buf.Write(b)
			} else {
				curLat, curLng := Encode(step.Lat, step.Lng)
				latDiff := lat - curLat
				lngDiff := lng - curLng
				
				buf.Write(diffToBytes(latDiff))
				buf.Write(diffToBytes(lngDiff))
				
				lat, lng = curLat, curLng
			}
		}
		if continueBecauseZero {
			continue
		}
		
		encoded[int(j)] = buf.Bytes()
				
		bytesOrg += OriginalStepSize * len(steps)
		bytesNew += buf.Len()
		bytesSaved += OriginalStepSize * len(steps) - buf.Len()
	}
	// end of encoding
	
	time2 := time.Now()
	
	fmt.Printf("bytes saved: %v\n", bytesSaved)
	fmt.Printf("%v / %v   %v\n", bytesNew, bytesOrg, float32(bytesNew) / float32(bytesOrg))
	
	var timeStepsSum int64 = 0
	var timeRefStepsSum int64 = 0
	edgeCount := 0
	
	maxError := 0.0
	sumError := 0.0
	stepCount := 0
	stepLengthSum := 0
	for j := 0; j < g.EdgeCount(); j++ {
		if encoded[j] == nil || len(encoded[j]) == 0 {
			continue
		}
		
		timeSteps1 := time.Now()
		steps := decodeSteps(encoded[j])
		timeSteps2 := time.Now()
		timeStepsSum += timeSteps2.Sub(timeSteps1).Nanoseconds()
		edgeCount++
		
		timeSteps3 := time.Now()
		refSteps := g.EdgeSteps(graph.Edge(j))
		timeSteps4 := time.Now()
		timeRefStepsSum += timeSteps4.Sub(timeSteps3).Nanoseconds()
		
		stepLengthSum += len(steps)
		
		if len(steps) != len(refSteps) {
			fmt.Printf("%d != %d\n", len(steps), len(refSteps))
			panic("length mismatch")
		}
		for i, _ := range steps {
			diffLat := math.Abs(steps[i].Lat - refSteps[i].Lat)
			diffLng := math.Abs(steps[i].Lng - refSteps[i].Lng)
			
			if diffLat > maxError { maxError = diffLat }
			if diffLng > maxError { maxError = diffLng }
			
			sumError += diffLat + diffLng
			stepCount += 2
		}
	}
	
	time3 := time.Now()
	
	fmt.Printf("max error: %v\n", maxError)
	fmt.Printf("avg error: %v\n", sumError / float64(stepCount))
	
	fmt.Printf("avg step length: %v\n", float64(stepLengthSum) / float64(edgeCount))
	
	fmt.Printf("time 1-2: %d\n", time2.Sub(time1).Nanoseconds() / (1000 * 1000))
	fmt.Printf("time 2-3: %d\n", time3.Sub(time2).Nanoseconds() / (1000 * 1000))
	
	fmt.Printf("    average time per decoding: %d\n", timeStepsSum / int64(edgeCount))
	fmt.Printf("vs. average time per access:   %d\n", timeRefStepsSum / int64(edgeCount))
}

func Bits(a int32) int {
	if a < 0 {
		a = -a
	}
	c := math.Log2(float64(a + 1))
	return int(math.Ceil(c))
}

func EncodeStepHistogram(start geo.Coordinate, step []geo.Coordinate, h *alg.Histogram) {
	prevLat, prevLng := start.Encode()
	for _, curr := range step {
		currLat, currLng := curr.Encode()
		dlat := currLat - prevLat
		dlng := currLng - prevLng
		h.Add(fmt.Sprintf("%v", Bits(dlat)))
		h.Add(fmt.Sprintf("%v", Bits(dlng)))
		prevLat, prevLng = currLat, currLng
	}
}

func EncodeSnappy(buffer []byte) []byte {
	sl, err := snappy.Encode(nil, buffer)
	if err != nil {
		panic(err.Error())
	}
	return sl
}

func EncodeFlate(buffer []byte) []byte {
	output := new(bytes.Buffer)
	w, err := flate.NewWriter(output, -1)
	if err != nil {
		panic(err.Error())
	}
	w.Write(buffer)
	w.Close()
	return output.Bytes()
}

func Entropy(h *alg.Histogram) int {
	samples := h.Samples()
	size := float64(len(samples))
	entropy := 0.0
	for _, sample := range samples {
		freq := float64(sample.Frequency)
		p := freq / size
		entropy += p * math.Log2(p)
	}
	entropy = -entropy
	
	fmt.Printf("Entropy: %.2f bits\n", entropy)
	encodedSize := int(math.Ceil(size * entropy / 8.0))
	fmt.Printf("Minimum size: %d bytes\n", encodedSize)
	return encodedSize
}

func encodeAndCheck2() {
	g := osmData[FlagMode].graph
	
	h := alg.NewHistogram("step indices")
		
	bytesOrg := 0
	bytesNew := 0
	stepCount := 0
	stepLengthSum := 0
	
	data := new(bytes.Buffer)
	
	//time1 := time.Now()
	
	// Encode
	for j := 0; j < g.EdgeCount(); j++ {
		e := graph.Edge(j)
		v := g.EdgeStartPoint(e)
		lat,lng := g.NodeLatLng(v)
		start := geo.Coordinate{lat, lng}

		// don't consider steps twice
		if revI, exists := g.EdgeReverse(e); exists && int(revI) < j {
			continue
		}
		
		gsteps := g.EdgeSteps(e)
		steps := make([]geo.Coordinate, len(gsteps))
		for i, s := range gsteps {
			steps[i] = geo.Coordinate{s.Lat, s.Lng}
		}
		l := geo.EncodeStep(start, steps)
		
		bytesOrg += OriginalStepSize * len(steps)
		bytesNew += len(l)
		stepCount++
		stepLengthSum += len(steps)
		
		data.Write(l)
		
		EncodeStepHistogram(start, steps, h)
	}
	// end of encoding
	
	out, err := os.Create("polylines.ftf")
	if err != nil {
		panic(err.Error())
	}
	out.Write(data.Bytes())
	out.Close()
	
	bytesSnap  := len(EncodeSnappy(data.Bytes()))
	bytesFlate := len(EncodeFlate(data.Bytes()))
	
	//time2 := time.Now()
	
	fmt.Printf("bytes saved (delta):  %v\n", bytesOrg - bytesNew)
	fmt.Printf("%v / %v   %v\n", bytesNew, bytesOrg, float32(bytesNew) / float32(bytesOrg))
	fmt.Printf("bytes saved (snappy): %v\n", bytesOrg - bytesSnap)
	fmt.Printf("%v / %v   %v\n", bytesSnap, bytesOrg, float32(bytesSnap) / float32(bytesOrg))
	fmt.Printf("bytes saved (flate):  %v\n", bytesOrg - bytesFlate)
	fmt.Printf("%v / %v   %v\n", bytesFlate, bytesOrg, float32(bytesFlate) / float32(bytesOrg))
	fmt.Printf("avg step length: %v\n", float64(stepLengthSum) / float64(stepCount))
	fmt.Printf("avg step size (org):   %v\n", float64(bytesOrg)  / float64(stepCount))
	fmt.Printf("avg step size (enc):   %v\n", float64(bytesNew)  / float64(stepCount))
	fmt.Printf("avg step size (snap):  %v\n", float64(bytesSnap) / float64(stepCount))
	fmt.Printf("avg step size (flate): %v\n", float64(bytesFlate) / float64(stepCount))
	//fmt.Printf("time: %d\n", time2.Sub(time1).Nanoseconds() / (1000 * 1000))
	//encodedSize := Entropy(h)
	//fmt.Printf("best ratio: %v\n", float64(encodedSize) / float64(bytesOrg))
	
	h.Print()
}

func decodeSteps(bytes []byte) []graph.Step {
	stepsCap := 1
	if len(bytes) > 8 {
		// approximate the needed capacity
		stepsCap = (len(bytes)-8) / (2*2) + 1
	}
	steps := make([]graph.Step, 1, stepsCap)

	latInt := int32(GetUint32(bytes[0:4]))
	lngInt := int32(GetUint32(bytes[4:8]))
	lat, lng := Decode(latInt, lngInt)
	steps[0] = graph.Step{lat, lng}
	
	for i := 8; i < len(bytes); {
		latDiff, bytesReadLat := diffFromBytes(bytes[i:])
		i += bytesReadLat
		lngDiff, bytesReadLng := diffFromBytes(bytes[i:])
		i += bytesReadLng
		latInt -= latDiff
		lngInt -= lngDiff
		lat, lng := Decode(latInt, lngInt)
		steps = append(steps, graph.Step{lat, lng})
	}
	return steps
}

func diffToBytes(diff int32) []byte {
	bytes := make([]byte, 10)
	bytesUsed := binary.PutVarint(bytes, int64(diff))
	return bytes[:bytesUsed]
}

func diffFromBytes(bytes []byte) (int32, int) {
	diff, bytesRead := binary.Varint(bytes)
	return int32(diff), bytesRead
}

// little endian
func PutUint32(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

// little endian
func GetUint32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

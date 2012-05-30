// The structure of the JSON object for the route response exactly as in the API definition

package main

type Result struct {
    BoundingBox BoundingBox `json:"boundingBox"`
    Routes []Route          `json:"routes"`
}

type BoundingBox struct {
    Nortwest Point  `json:"nw"`
    Southeast Point `json:"se"`
}

type Route struct {
    Distance Distance   `json:"distance"`
    Duration Duration   `json:"duration"`
    StartLocation Point `json:"start_location"`
    EndLocation Point   `json:"end_location"`
    Legs []Leg          `json:"legs"`
}

type Leg struct {
    Distance Distance   `json:"distance"`
    Duration Duration   `json:"duration"`
    StartLocation Point `json:"start_location"`
    EndLocation Point   `json:"end_location"`
    Steps []Step        `json:"steps"`
}

type Step struct {
    Distance Distance   `json:"distance"`
    Duration Duration   `json:"duration"`
    StartLocation Point `json:"start_location"`
    EndLocation Point   `json:"end_location"`
    Polyline Polyline   `json:"polyline"`
    Instruction string  `json:"instruction"`
}

type Point struct {
    Lat float64 `json:"lat"`
    Lng float64 `json:"lng"`
}

type Distance struct {
    Text string `json:"text"`
    Value int   `json:"value"`
}

type Duration struct {
    Text string `json:"text"`
    Value int   `json:"value"`
}

type Polyline [][]float64


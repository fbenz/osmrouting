
package main

/* Structure of the JSON object exactly as in the API definition */

type Result struct {
	BoundingBox BoundingBox `json:"boundingBox"`
	Routes []Route			`json:"routes"`
}

type BoundingBox struct {
	Nortwest Point	`json:"nw"`
	Southeast Point	`json:"se"`
}

type Route struct {
	Distance Distance	`json:"distance"`
	Duration Duration	`json:"duration"`
	StartLocation Point	`json:"start_location"`
	EndLocation Point	`json:"end_location"`
	Legs Leg			`json:"legs"`
}

type Distance struct {
	Text string `json:"text"`
	Value int	`json:"value"`
}

type Duration struct {
	Text string	`json:"text"`
	Value int	`json:"value"`
}

type Leg struct {
	Distance Distance	`json:"distance"`
	Duration Duration	`json:"duration"`
	StartLocation Point	`json:"start_location"`
	EndLocation Point	`json:"end_location"`
	Steps Step			`json:"steps"`
}

type Polyline [][]float64

type Step struct {
	Distance Distance	`json:"distance"`
	Duration Duration	`json:"duration"`
	StartLocation Point	`json:"start_location"`
	EndLocation Point	`json:"end_location"`
	Polyline Polyline	`json:"end_location"`
	Instruction string	`json:"instruction"`
}


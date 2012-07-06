// The structure of the JSON object for the route response exactly as in the API definition

package route

type Result struct {
	BoundingBox BoundingBox `json:"boundingBox"`
	Routes      []Route     `json:"routes"`
}

type BoundingBox struct {
	Northwest Point `json:"nw"`
	Southeast Point `json:"se"`
}

type Route struct {
	Distance      Distance `json:"distance"`
	Duration      Duration `json:"duration"`
	StartLocation Point    `json:"start_location"`
	EndLocation   Point    `json:"end_location"`
	Legs          []*Leg   `json:"legs"`
}

type Leg struct {
	Distance      Distance `json:"distance"`
	Duration      Duration `json:"duration"`
	StartLocation Point    `json:"start_location"`
	EndLocation   Point    `json:"end_location"`
	Steps         []Step   `json:"steps"`
}

type Step struct {
	Distance      Distance `json:"distance"`
	Duration      Duration `json:"duration"`
	StartLocation Point    `json:"start_location"`
	EndLocation   Point    `json:"end_location"`
	Polyline      Polyline `json:"polyline"`
	Instruction   string   `json:"instruction"`
}

type Distance struct {
	Text  string `json:"text"`
	Value int    `json:"value"`
}

type Duration struct {
	Text  string `json:"text"`
	Value int    `json:"value"`
}

type Point []float64

type Polyline []Point

// The features supported by this implementation

package main

type Features struct {
	TravelMode TravelMode `json:"travelmode"`
	Metric     Metric     `json:"metric"`
	Avoid 	   Avoid	  `json:"avoid"`
}

type TravelMode struct {
	Driving bool	`json:"driving"`
	Walking bool	`json:"walking"`
	Bicycling bool	`json:"bicycling"`
}

type Metric struct {
	Distance bool	`json:"distance"`
	Time bool		`json:"time"`
}

type Avoid struct {
	Ferries bool	`json:"ferries"`
}

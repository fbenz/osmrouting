
package main

/* The features supported by this implementation */

type Features struct {
	TravelMode TravelMode 	`json:"travelmode"`
	Metric Metric			`json:"metric"`
	Avoid Avoid				`json:"avoid"`
}

type TravelMode struct {
}

type Metric struct {
}

type Avoid struct {
}



package main

type Point struct {
	Lat float64	`json:"lat"`
	Lng float64	`json:"lng"`
}

type RouteInfo struct {
	Waypoints []Point
}

func NewPoint(lat, lng float64) *Point {
	return &Point{lat, lng}
}

func NewRouteInfo(waypoints []Point) *RouteInfo {
	return &RouteInfo{waypoints}
}

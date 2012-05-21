
package main

/* All information provided with a route query */

type RouteInfo struct {
	Waypoints []Point
}

func NewPoint(lat, lng float64) *Point {
	return &Point{lat, lng}
}

func NewRouteInfo(waypoints []Point) *RouteInfo {
	return &RouteInfo{waypoints}
}

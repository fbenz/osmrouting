// All information provided with a route query

package main

type RouteInfo struct {
    Waypoints []Point
}

func NewRouteInfo(waypoints []Point) *RouteInfo {
    return &RouteInfo{waypoints}
}


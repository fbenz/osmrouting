package geo

import "math"

type BBox struct {
	Min Coordinate
	Max Coordinate
}

func NewBBox(a, b Coordinate) BBox {
	minLat := math.Min(a.Lat, b.Lat)
	minLng := math.Min(a.Lng, b.Lng)
	maxLat := math.Max(a.Lat, b.Lat)
	maxLng := math.Max(a.Lng, b.Lng)
	return BBox{Coordinate{minLat, minLng}, Coordinate{maxLat, maxLng}}
}

func NewBBoxPoint(a Coordinate) BBox {
	return BBox{a, a}
}

func EmptyBBox() BBox {
	return BBox{Coordinate{1,1},Coordinate{-1,-1}}
}

func (b BBox) Encode() [4]int32 {
	r0, r1 := b.Min.Encode()
	r2, r3 := b.Max.Encode()
	return [4]int32{r0,r1,r2,r3}
}

func DecodeBBox(e []int32) BBox {
	min := DecodeCoordinate(e[0], e[1])
	max := DecodeCoordinate(e[2], e[3])
	return BBox{min, max}
}

func (b BBox) Northwest() Coordinate {
	return Coordinate{b.Max.Lat, b.Min.Lng}
}

func (b BBox) Southeast() Coordinate {
	return Coordinate{b.Min.Lat, b.Max.Lng}
}

func (b BBox) Union(a BBox) BBox {
	minLat := math.Min(a.Min.Lat, b.Min.Lat)
	minLng := math.Min(a.Min.Lng, b.Min.Lng)
	maxLat := math.Max(a.Max.Lat, b.Max.Lat)
	maxLng := math.Max(a.Max.Lng, b.Max.Lng)
	return BBox{Coordinate{minLat, minLng}, Coordinate{maxLat, maxLng}}
}

func (b BBox) Center() Coordinate {
	return Coordinate{
		Lat: (b.Min.Lat + b.Max.Lat) / 2.0,
		Lng: (b.Min.Lng + b.Max.Lng) / 2.0,
	}
}

func (b BBox) Contains(p Coordinate) bool {
	return b.Min.Lat <= p.Lat && p.Lat <= b.Max.Lat &&
		   b.Min.Lng <= p.Lng && p.Lng <= b.Max.Lng
}

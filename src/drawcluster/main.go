/*
 * Copyright 2014 Florian Benz, Steven Schäfer, Bernhard Schommer
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"

	"geo"
	"graph"
)

type Projection struct {
	Fn         func(geo.Coordinate) (float64, float64)
	MinX, MinY float64
	MaxX, MaxY float64
}

var (
	// command line flags
	InputFile    string
	OutputFile   string
	OutputWidth  uint
	OutputHeight uint
)

func init() {
	flag.StringVar(&InputFile, "i", "", "input graph directory")
	flag.StringVar(&OutputFile, "o", "", "output image file")
	flag.UintVar(&OutputWidth, "w", 1024, "width of the output image")
}

func MercatorProjection(c geo.Coordinate) (x, y float64) {
	shift := 2 * math.Pi * 6378137 / 2.0 // <- google maps compatible
	x = c.Lng * shift / 180.0
	y = math.Log(math.Tan((90+c.Lat)*math.Pi/360.0)) / (math.Pi / 180.0)
	y = y * shift / 180.0
	return
}

func ComputeProjection(partitions []*graph.GraphFile) *Projection {
	minx, miny := math.MaxFloat64, math.MaxFloat64
	maxx, maxy := -math.MaxFloat64, -math.MaxFloat64
	for _, g := range partitions {
		for i := 0; i < g.VertexCount(); i++ {
			v := graph.Vertex(i)
			x, y := MercatorProjection(g.VertexCoordinate(v))
			if x < minx {
				minx = x
			}
			if x > maxx {
				maxx = x
			}
			if y < miny {
				miny = y
			}
			if y > maxy {
				maxy = y
			}
		}
	}
	return &Projection{
		Fn:   MercatorProjection,
		MinX: minx,
		MinY: miny,
		MaxX: maxx,
		MaxY: maxy,
	}
}

func (p *Projection) Project(c geo.Coordinate) (x, y float64) {
	rx, ry := p.Fn(c)
	x = float64(OutputWidth) * (rx - p.MinX) / (p.MaxX - p.MinX)
	y = float64(OutputHeight) * (1.0 - (ry-p.MinY)/(p.MaxY-p.MinY))
	return
}

func (p *Projection) Height(width uint) uint {
	pwidth := p.MaxX - p.MinX
	pheight := p.MaxY - p.MinY
	return uint(float64(width) * pheight / pwidth)
}

func RGB(r, g, b float64) color.Color {
	xr := uint16(0xffff * r)
	xg := uint16(0xffff * g)
	xb := uint16(0xffff * b)
	return color.RGBA64{xr, xg, xb, 0xffff}
}

func HSV(h, s, v float64) color.Color {
	c := s * v
	q := h / 60.0
	x := c * (1.0 - math.Abs(math.Mod(q, 2.0)-1))
	m := v - c
	switch int(q) {
	case 0:
		return RGB(m+c, m+x, m)
	case 1:
		return RGB(m+x, m+c, m)
	case 2:
		return RGB(m, m+c, m+x)
	case 3:
		return RGB(m, m+x, m+c)
	case 4:
		return RGB(m+x, m, m+c)
	case 5:
		return RGB(m+c, m, m+x)
	}
	return RGB(0, 0, 0)
}

const (
	SIGMA = 2.0 / 3.0
)

func Kernel(x float64) float64 {
	norm := math.Sqrt(2*math.Pi) * SIGMA
	return math.Exp(-x*x/(2*SIGMA*SIGMA)) / norm
}

func Alpha(alpha float64, c color.Color) color.Color {
	// I don't think that alpha values should be gamma corrected,
	// but I implemented it like this in some old code, so presumbaly
	// it looks better this way.
	//alpha = math.Pow(alpha, 0.45)
	r, g, b, _ := c.RGBA()
	xr := uint16(alpha * float64(r))
	xg := uint16(alpha * float64(g))
	xb := uint16(alpha * float64(b))
	xa := uint16(0xffff * alpha)
	return color.RGBA64{xr, xg, xb, xa}
}

func Blend(img *image.RGBA, x, y int, c color.Color) {
	sr, sg, sb, sa := img.At(x, y).RGBA()
	dr, dg, db, da := c.RGBA()
	r := uint16(dr + ((0xffff-da)*sr)/0xffff)
	g := uint16(dg + ((0xffff-da)*sg)/0xffff)
	b := uint16(db + ((0xffff-da)*sb)/0xffff)
	a := uint16(da + ((0xffff-da)*sa)/0xffff)
	img.Set(x, y, color.RGBA64{r, g, b, a})
}

func DrawPoint(x, y float64, c color.Color, img *image.RGBA) {
	xi, yi := int(x), int(y)
	fx, fy := x-float64(xi), y-float64(yi)
	gx0 := Kernel(fx)
	gx1 := Kernel(1 - fx)
	gy0 := Kernel(fy)
	gy1 := Kernel(1 - fy)
	Blend(img, xi, yi, Alpha(gx0*gy0, c))
	Blend(img, xi+1, yi, Alpha(gx1*gy0, c))
	Blend(img, xi, yi+1, Alpha(gx0*gy1, c))
	Blend(img, xi+1, yi+1, Alpha(gx1*gy1, c))
}

func DrawCircle(x, y float64, c color.Color, img *image.RGBA) {
	radius := math.Ceil(3.0 * SIGMA)
	ix := x - 0.5
	iy := y - 0.5
	x0 := int(math.Ceil(ix - radius))
	y0 := int(math.Ceil(iy - radius))
	x1 := int(math.Floor(ix + radius))
	y1 := int(math.Floor(iy + radius))
	d := 0.0
	for py := y0; py <= y1; py++ {
		for px := x0; px <= x1; px++ {
			d += Kernel(float64(px)-ix) * Kernel(float64(py)-iy)
		}
	}

	if d == 0.0 {
		return
	}

	for py := y0; py <= y1; py++ {
		for px := x0; px <= x1; px++ {
			w := Kernel(float64(px)-ix) * Kernel(float64(py)-iy)
			Blend(img, px, py, Alpha(0.5*w/d, c))
		}
	}
}

func DrawPartition(g graph.Graph, proj *Projection, c color.Color, img *image.RGBA) {
	for i := 0; i < g.VertexCount(); i++ {
		v := graph.Vertex(i)
		x, y := proj.Project(g.VertexCoordinate(v))
		DrawCircle(x, y, c, img)
		//DrawPoint(x, y, c, img)
	}
}

func main() {
	flag.Parse()
	if InputFile == "" || OutputFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	println("Open cluster graph.")
	h, err := graph.OpenClusterGraph(InputFile, false)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	println("Computing the projection.")
	proj := ComputeProjection(h.Cluster)

	println("Rendering")
	OutputHeight = proj.Height(OutputWidth)
	bounds := image.Rect(0, 0, int(OutputWidth), int(OutputHeight))
	img := image.NewRGBA(bounds)
	for y := 0; y < int(OutputHeight); y++ {
		for x := 0; x < int(OutputWidth); x++ {
			img.Set(x, y, color.RGBA{0x0, 0x0, 0x0, 0xff})
		}
	}

	// Draw all the partitions sequentially
	for i, g := range h.Cluster {
		fmt.Printf(" * Cluster %v/%v\n", i+1, len(h.Cluster))
		f := float64(i) / float64(len(h.Cluster))
		h := math.Mod(36000*f, 360)
		s := 0.9
		v := 0.9 + 0.1*f
		c := HSV(h, s, v)
		DrawPartition(g, proj, c, img)
	}

	// write the result into a png image
	println("Output")
	out, err := os.Create(OutputFile)
	if err != nil {
		panic(err.Error())
	}
	png.Encode(out, img)
	out.Close()
}

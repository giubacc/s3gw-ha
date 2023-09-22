// Copyright © 2023 SUSE LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//***********************************************************************
// The S3WorkloadBars implementation is based on the Peter Paolucci's work.
// https://github.com/pplcc/plotext See vbars.go
//***********************************************************************

package utils

import (
	"image/color"
	"math"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// S3WorkloadBars implements the Plotter interface, drawing
// a S3WorkloadEntry bar plot.
type S3WorkloadBars struct {
	S3WLE *[]S3WorkloadEntry

	//TimeUnit
	TimeUnit int64

	// ColorOK is the color of bars when the S3WorkloadEntry
	// has completed successfully
	ColorOK color.Color

	// ColorErr is the color of bars when the S3WorkloadEntry
	// has completed with an error
	ColorErr color.Color

	// LineStyle is the style used to draw the bars.
	draw.LineStyle
}

// NewBars creates as new bar plotter for
// the given data.
func NewVBars(S3WLE *[]S3WorkloadEntry, timeUnit string) (*S3WorkloadBars, error) {
	return &S3WorkloadBars{
		S3WLE:     S3WLE,
		TimeUnit:  StrTimeUnit2TimeUnit[timeUnit],
		ColorOK:   color.RGBA{R: 0, G: 128, B: 0, A: 255}, // eye is more sensible to green
		ColorErr:  color.RGBA{R: 196, G: 0, B: 0, A: 255},
		LineStyle: plotter.DefaultLineStyle,
	}, nil
}

// Plot implements the Plot method of the plot.Plotter interface.
func (bars *S3WorkloadBars) Plot(c draw.Canvas, plt *plot.Plot) {
	trX, trY := plt.Transforms(&c)
	lineStyle := bars.LineStyle

	for _, it := range *bars.S3WLE {
		if it.ErrDesc != "" {
			lineStyle.Color = bars.ColorErr
		} else {
			lineStyle.Color = bars.ColorOK
		}

		// Transform the data
		// to the corresponding drawing coordinate.
		x := trX(float64(((it.Start - (*bars.S3WLE)[0].Start) / bars.TimeUnit)))
		y0 := trY(0)
		y := trY(it.RTT)

		bar := c.ClipLinesY([]vg.Point{{X: x, Y: y0}, {X: x, Y: y}})
		c.StrokeLines(lineStyle, bar...)

	}
}

// DataRange implements the DataRange method
// of the plot.DataRanger interface.
func (bars *S3WorkloadBars) DataRange() (xmin, xmax, ymin, ymax float64) {
	xmin = math.Inf(1)
	xmax = math.Inf(-1)
	ymin = 0
	ymax = math.Inf(-1)
	for _, it := range *bars.S3WLE {
		xmin = math.Min(xmin, float64(((it.Start - (*bars.S3WLE)[0].Start) / bars.TimeUnit)))
		xmax = math.Max(xmax, float64(((it.Start - (*bars.S3WLE)[0].Start) / bars.TimeUnit)))
		ymax = math.Max(ymax, it.RTT)
	}
	return
}

// GlyphBoxes implements the GlyphBoxes method
// of the plot.GlyphBoxer interface.
func (bars *S3WorkloadBars) GlyphBoxes(plt *plot.Plot) []plot.GlyphBox {
	boxes := make([]plot.GlyphBox, 2)

	xmin, xmax, ymin, ymax := bars.DataRange()

	boxes[0].X = plt.X.Norm(xmin)
	boxes[0].Y = plt.Y.Norm(ymin)
	boxes[0].Rectangle = vg.Rectangle{
		Min: vg.Point{X: -bars.LineStyle.Width / 2, Y: 0},
		Max: vg.Point{X: 0, Y: 0},
	}

	boxes[1].X = plt.X.Norm(xmax)
	boxes[1].Y = plt.Y.Norm(ymax)
	boxes[1].Rectangle = vg.Rectangle{
		Min: vg.Point{X: 0, Y: 0},
		Max: vg.Point{X: +bars.LineStyle.Width / 2, Y: 0},
	}

	return boxes
}

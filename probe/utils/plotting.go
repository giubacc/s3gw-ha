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

package utils

import (
	"github.com/montanaflynn/stats"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

func GenerateRawDataPlot(data RestartRelatedData,
	mark string,
	timeUnit string,
	fileName string,
	genTS string) {

	if restartEvents, hit := data[mark]; hit {
		p := plot.New()
		p.Add(plotter.NewGrid())

		p.Title.Text = "Raw Data Measures: " + mark
		p.X.Label.Text = "Restart ID"
		p.Y.Label.Text = "Duration: " + timeUnit

		_, evtSeriesMainData, evtSeriesFrontedUpData := GetSplitDataForSingleRestartRelatedData(restartEvents, StrTimeUnit2TimeUnit[timeUnit])

		//main

		{
			mainPts := make(plotter.XYs, len(restartEvents))
			for i := range mainPts {
				mainPts[i].X = float64(i)
				mainPts[i].Y = evtSeriesMainData[i]
			}

			lpLineMain, lpPointsMain, err := plotter.NewLinePoints(mainPts)
			if err != nil {
				Logger.Error("GeneratePlot-to-main: NewLinePoints", err.Error())
				return
			}
			lpLineMain.Color = plotutil.Color(0)
			lpPointsMain.Shape = draw.PyramidGlyph{}
			lpPointsMain.Color = plotutil.Color(0)

			p.Add(lpLineMain, lpPointsMain)
			p.Legend.Add("to-main", lpLineMain, lpPointsMain)
		}

		//fronted-up

		{
			frontendUpPts := make(plotter.XYs, len(restartEvents))
			for i := range frontendUpPts {
				frontendUpPts[i].X = float64(i)
				frontendUpPts[i].Y = evtSeriesFrontedUpData[i]
			}

			lpLineFUp, lpPointsFUp, err := plotter.NewLinePoints(frontendUpPts)
			if err != nil {
				Logger.Error("GeneratePlot-to-fronted-up: NewLinePoints", err.Error())
				return
			}
			lpLineFUp.Color = plotutil.Color(1)
			lpPointsFUp.Shape = draw.PyramidGlyph{}
			lpPointsFUp.Color = plotutil.Color(1)

			p.Add(lpLineFUp, lpPointsFUp)
			p.Legend.Add("to-fronted-up", lpLineFUp, lpPointsFUp)
		}

		fName := fileName + "_" + "raw" + "_" + genTS + ".png"

		if err := p.Save(30*vg.Centimeter, 20*vg.Centimeter, fName); err != nil {
			Logger.Error("GeneratePlot: Saving plot:", fName)
		}
	} else {
		Logger.Error("GeneratePlot: no series with mark:", mark)
	}
}

func GeneratePercentilesPlot(data RestartRelatedData,
	mark string,
	timeUnit string,
	fileName string,
	genTS string) {
	if restartEvents, hit := data[mark]; hit {
		p := plot.New()
		p.Add(plotter.NewGrid())

		p.Title.Text = "Percentiles (Nearest Rank): " + mark
		p.X.Label.Text = "Percentile"
		p.Y.Label.Text = "Duration: " + timeUnit

		_, evtSeriesMainData, evtSeriesFrontedUpData := GetSplitDataForSingleRestartRelatedData(restartEvents, StrTimeUnit2TimeUnit[timeUnit])

		//main

		{
			mainPercPts := make(plotter.XYs, 100)
			for i := 1; i <= 100; i++ {
				if val, err := stats.PercentileNearestRank(evtSeriesMainData, float64(i)); err == nil {
					mainPercPts[i-1].X = float64(i)
					mainPercPts[i-1].Y = val
				}
			}

			lpLineMain, lpPointsMain, err := plotter.NewLinePoints(mainPercPts)
			if err != nil {
				Logger.Error("GeneratePercentilesPlot-to-main: NewLinePoints", err.Error())
				return
			}
			lpLineMain.Color = plotutil.Color(0)
			lpPointsMain.Shape = draw.PyramidGlyph{}
			lpPointsMain.Color = plotutil.Color(0)

			p.Add(lpLineMain, lpPointsMain)
			p.Legend.Add("to-main", lpLineMain, lpPointsMain)
		}

		//fronted-up

		{
			fUPPercPts := make(plotter.XYs, 100)
			for i := 1; i <= 100; i++ {
				if val, err := stats.PercentileNearestRank(evtSeriesFrontedUpData, float64(i)); err == nil {
					fUPPercPts[i-1].X = float64(i)
					fUPPercPts[i-1].Y = val
				}
			}

			lpLinefUP, lpPointsfUP, err := plotter.NewLinePoints(fUPPercPts)
			if err != nil {
				Logger.Error("GeneratePercentilesPlot-to-fronted-up: NewLinePoints", err.Error())
				return
			}
			lpLinefUP.Color = plotutil.Color(1)
			lpPointsfUP.Shape = draw.PyramidGlyph{}
			lpPointsfUP.Color = plotutil.Color(1)

			p.Add(lpLinefUP, lpPointsfUP)
			p.Legend.Add("to-fronted-up", lpLinefUP, lpPointsfUP)
		}

		fName := fileName + "_" + "percentiles" + "_" + genTS + ".png"

		if err := p.Save(30*vg.Centimeter, 20*vg.Centimeter, fName); err != nil {
			Logger.Error("GeneratePlot: Saving plot:", fName)
		}
	} else {
		Logger.Error("GeneratePercentilesPlot: no series with mark:", mark)
	}
}

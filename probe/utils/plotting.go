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
	"strconv"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

func GenerateRawDataPlot(data RestartRelatedData,
	mark string,
	timeUnit string,
	fileName string) {

	if restartEvents, hit := data[mark]; hit {
		p := plot.New()

		p.Title.Text = "Raw Data Measures: " + mark
		p.X.Label.Text = "Restart ID"
		p.Y.Label.Text = "Duration"

		_, evtSeriesMainData, evtSeriesFrontedUpData := GetSplitDataForSingleRestartRelatedData(restartEvents, StrTimeUnit2TimeUnit[timeUnit])

		mainPts := make(plotter.XYs, len(restartEvents))
		for i := range mainPts {
			mainPts[i].X = float64(i)
			mainPts[i].Y = evtSeriesMainData[i]
		}

		if err := plotutil.AddLinePoints(p, "to-main", mainPts); err != nil {
			Logger.Error("GeneratePlot-to-main: AddLinePoints", err.Error())
			return
		}

		frontendUpPts := make(plotter.XYs, len(restartEvents))
		for i := range mainPts {
			frontendUpPts[i].X = float64(i)
			frontendUpPts[i].Y = evtSeriesFrontedUpData[i]
		}

		if err := plotutil.AddLinePoints(p, "to-frontend-up", frontendUpPts); err != nil {
			Logger.Error("GeneratePlot-to-frontend-up: AddLinePoints", err.Error())
			return
		}

		if err := p.Save(20*vg.Centimeter, 16*vg.Centimeter, fileName+"_"+strconv.Itoa(int(time.Now().Unix()))+".png"); err != nil {
			Logger.Error("GeneratePlot: Saving plot:", fileName)
		}
	} else {
		Logger.Error("GeneratePlot: no series with mark:", mark)
	}

}

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
	"bytes"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/montanaflynn/stats"
)

const (
	StrNanoS  = "ns"
	StrMicroS = "us"
	StrMilliS = "ms"
	StrSec    = "s"
)

const (
	NanoS  = 1
	MicroS = 1000
	MilliS = 1000000
	Sec    = 1000000000
)

var StrTimeUnit2TimeUnit = map[string]uint64{
	StrNanoS:  NanoS,
	StrMicroS: MicroS,
	StrMilliS: MilliS,
	StrSec:    Sec}

type RestartRelatedData map[string][]RestartEvent

type Probe struct {
	CurrentDeath     *DeathEvent
	CurrentStartList []*StartEvent

	CurrentPendingRestarts uint
	CurrentDeathType       string
	CurrentMark            string
	CurrentId              int

	CollectedRestartRelatedData RestartRelatedData
}

func (p *Probe) SubmitDeath(evt *DeathEvent) {
	if p.CurrentDeath != nil {
		Logger.Error("bad state for submitting a death event")
		return
	}
	p.CurrentDeath = evt
}

func (p *Probe) SubmitStart(evt *StartEvent) {
	if p.CurrentDeath == nil {
		Logger.Error("bad state for submitting a start event")
		return
	}
	p.CurrentStartList = append(p.CurrentStartList, evt)

	if evt.Where == Cfg.CollectRestartAtEvent {
		p.submitRestart()
		if p.CurrentPendingRestarts > 0 {
			time.Sleep(time.Duration(Cfg.WaitMSecsBeforeTriggerDeath) * time.Millisecond)
			p.RequestDie()
		} else {
			p.CurrentId = 0
		}
	}
}

func (p *Probe) findStartEvent(where string) *StartEvent {
	for _, it := range p.CurrentStartList {
		if it.Where == where {
			return it
		}
	}
	return nil
}

func (p *Probe) submitRestart() {
	p.CurrentId = p.CurrentId + 1

	p.CollectedRestartRelatedData[p.CurrentMark] = append(p.CollectedRestartRelatedData[p.CurrentMark],
		RestartEvent{Death: p.CurrentDeath,
			StartMain:       p.findStartEvent("main"),
			StartFrontendUp: p.findStartEvent("frontend-up"),
			Id:              p.CurrentId})

	restartEvt := &p.CollectedRestartRelatedData[p.CurrentMark][len(p.CollectedRestartRelatedData[p.CurrentMark])-1]

	if restartEvt.StartMain == nil {
		restartEvt.StartMain = &StartEvent{Ts: restartEvt.Death.Ts}
	}
	if restartEvt.StartFrontendUp == nil {
		restartEvt.StartFrontendUp = &StartEvent{Ts: restartEvt.Death.Ts}
	}

	Logger.Infof("inserted restart event: mark:%s, death:%d, start-main:%d, start-f-up:%d; collected events:%d",
		p.CurrentMark,
		p.CurrentDeath.Ts,
		restartEvt.StartMain.Ts,
		restartEvt.StartFrontendUp.Ts,
		p.CurrentId)

	if p.CurrentPendingRestarts > 0 {
		p.CurrentPendingRestarts = p.CurrentPendingRestarts - 1
	}
	Logger.Infof("pending restarts: %d", p.CurrentPendingRestarts)
	p.CurrentDeath = nil
	p.CurrentStartList = nil
}

func (p *Probe) RequestDie() {
	req, err := http.NewRequest("PUT", Cfg.EndpointS3GW, bytes.NewReader([]byte("")))
	if err != nil {
		Logger.Errorf("NewRequest:%s", err.Error())
	}
	req.URL.Path = "/admin/bucket"
	req.Header.Set("x-amz-content-sha256", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	q := req.URL.Query()
	q.Add("die", "1")
	q.Add("how", p.CurrentDeathType)
	req.URL.RawQuery = q.Encode()

	creds := credentials.NewEnvCredentials()
	if err = SignHTTPRequestV4(req, creds); err != nil {
		Logger.Errorf("SignHTTPRequestV4:%s", err.Error())
		return
	}

	client := &http.Client{}
	if _, err = client.Do(req); err != nil {
		Logger.Errorf("Do:%s", err.Error())
	}
}

func GetSplitDataForSingleRestartRelatedData(restartEvents []RestartEvent, timeUnit uint64) ([]RestartEntry, []float64, []float64) {
	var evtSeries []RestartEntry
	var evtSeriesMainData []float64
	var evtSeriesFrontedUpData []float64
	for _, evt := range restartEvents {
		evtSeries = append(evtSeries, RestartEntry{Id: evt.Id,
			RestartDurationToMain:       (evt.StartMain.Ts - evt.Death.Ts) / timeUnit,
			RestartDurationToFrontendUp: (evt.StartFrontendUp.Ts - evt.Death.Ts) / timeUnit})

		evtSeriesMainData = append(evtSeriesMainData, float64(evtSeries[len(evtSeries)-1].RestartDurationToMain))
		evtSeriesFrontedUpData = append(evtSeriesFrontedUpData, float64(evtSeries[len(evtSeries)-1].RestartDurationToFrontendUp))
	}
	return evtSeries, evtSeriesMainData, evtSeriesFrontedUpData
}

func (p *Probe) ComputeStats(timeUnit string, dumpAllData bool) Stats {
	result := Stats{TimeUnit: timeUnit}
	for mark, restartEvents := range Prb.CollectedRestartRelatedData {

		evtSeries, evtSeriesMainData, evtSeriesFrontedUpData := GetSplitDataForSingleRestartRelatedData(restartEvents, StrTimeUnit2TimeUnit[timeUnit])

		result.Series = append(result.Series, SeriesEntry{Mark: mark})
		lastSeries := &result.Series[len(result.Series)-1]

		if val, err := stats.Min(evtSeriesMainData); err == nil {
			lastSeries.MinMain = uint64(val)
		}

		if val, err := stats.Max(evtSeriesMainData); err == nil {
			lastSeries.MaxMain = uint64(val)
		}

		if val, err := stats.Mean(evtSeriesMainData); err == nil {
			lastSeries.MeanMain = uint64(val)
		}

		if val, err := stats.Percentile(evtSeriesMainData, 99); err == nil {
			lastSeries.Perc99Main = uint64(val)
		}

		if val, err := stats.Percentile(evtSeriesMainData, 95); err == nil {
			lastSeries.Perc95Main = uint64(val)
		}

		if val, err := stats.PercentileNearestRank(evtSeriesMainData, 99); err == nil {
			lastSeries.PercNR99Main = uint64(val)
		}

		if val, err := stats.PercentileNearestRank(evtSeriesMainData, 95); err == nil {
			lastSeries.PercNR95Main = uint64(val)
		}

		if val, err := stats.Min(evtSeriesFrontedUpData); err == nil {
			lastSeries.MinFrontUp = uint64(val)
		}

		if val, err := stats.Max(evtSeriesFrontedUpData); err == nil {
			lastSeries.MaxFrontUp = uint64(val)
		}

		if val, err := stats.Mean(evtSeriesFrontedUpData); err == nil {
			lastSeries.MeanFrontUp = uint64(val)
		}

		if val, err := stats.Percentile(evtSeriesFrontedUpData, 99); err == nil {
			lastSeries.Perc99FrontUp = uint64(val)
		}

		if val, err := stats.Percentile(evtSeriesFrontedUpData, 95); err == nil {
			lastSeries.Perc95FrontUp = uint64(val)
		}

		if val, err := stats.PercentileNearestRank(evtSeriesFrontedUpData, 99); err == nil {
			lastSeries.PercNR99FrontUp = uint64(val)
		}

		if val, err := stats.PercentileNearestRank(evtSeriesFrontedUpData, 95); err == nil {
			lastSeries.PercNR95FrontUp = uint64(val)
		}

		if dumpAllData {
			lastSeries.Data = evtSeries
		}
	}
	return result
}

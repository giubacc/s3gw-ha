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

package main

import (
	"bytes"
	"flag"
	"net/http"
	. "s3gw-ha/probe/utils"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/gin-gonic/gin"
	"github.com/montanaflynn/stats"
)

type Probe struct {
	CurrentDeath     *DeathEvent
	CurrentStartList []*StartEvent

	CurrentPendingRestarts uint
	CurrentDeathType       string
	CurrentMark            string
	CurrentId              int

	RestartSet map[string][]RestartEvent
}

func (p *Probe) submitDeath(evt *DeathEvent) {
	if p.CurrentDeath != nil {
		Logger.Error("bad state for submitting a death event")
		return
	}
	p.CurrentDeath = evt
}

func (p *Probe) submitStart(evt *StartEvent) {
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

	p.RestartSet[p.CurrentMark] = append(p.RestartSet[p.CurrentMark],
		RestartEvent{Death: p.CurrentDeath,
			StartMain:       p.findStartEvent("main"),
			StartFrontendUp: p.findStartEvent("frontend-up"),
			Id:              p.CurrentId})

	restartEvt := &p.RestartSet[p.CurrentMark][len(p.RestartSet[p.CurrentMark])-1]

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

var Prb Probe

func main() {
	flag.StringVar(&Cfg.EndpointS3GW, "s3gw-endpoint", "http://localhost:7480", "Specify s3gw endpoint")
	flag.UintVar(&Cfg.WaitMSecsBeforeTriggerDeath, "wbtd", 100, "Wait n milliseconds before trigger death")
	flag.StringVar(&Cfg.CollectRestartAtEvent, "collectAt", "frontend-up", "The event where the probe should collect a restart event")
	flag.StringVar(&Cfg.LogLevel, "v", "inf", "Specify logging verbosity [off, trc, inf, wrn, err]")
	flag.UintVar(&Cfg.VerbLevel, "vl", 5, "Verbosity level")

	flag.Parse()

	Prb.RestartSet = make(map[string][]RestartEvent)

	Logger = GetLogger(&Cfg)

	router := gin.Default()

	router.PUT("/death", setDeath)
	router.PUT("/start", setStart)
	router.GET("/stats", getStats)
	router.PUT("/probe", probe)

	Logger.Info("start listening and serving ...")
	router.Run() // listen and serve on 0.0.0.0:8080
}

func setDeath(c *gin.Context) {
	deathType := c.Query("type")
	ts := c.Query("ts")
	if ts, err := strconv.ParseUint(ts, 0, 64); err == nil {
		evt := DeathEvent{Type: deathType, Ts: ts}
		Prb.submitDeath(&evt)
	} else {
		Logger.Errorf("malformed DeathEvent:%s", err.Error())
		c.String(http.StatusBadRequest, err.Error())
	}
}

func setStart(c *gin.Context) {
	ts := c.Query("ts")
	where := c.Query("where")
	if ts, err := strconv.ParseUint(ts, 0, 64); err == nil {
		evt := StartEvent{Ts: ts, Where: where}
		Prb.submitStart(&evt)
	} else {
		Logger.Errorf("malformed StartEvent:%s", err.Error())
		c.String(http.StatusBadRequest, err.Error())
	}
}

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

func getStats(c *gin.Context) {
	timeUnit := c.Query("time_unit")
	if timeUnit == "" {
		timeUnit = StrSec
	}

	fullSeries := false
	fullSeriesPar := c.Query("full_series")
	if fullSeriesPar == "true" {
		fullSeries = true
	}

	result := Stats{TimeUnit: timeUnit}
	for mark, restartEvents := range Prb.RestartSet {
		var evtSeries []RestartEntry
		var evtSeriesMainData []float64
		var evtSeriesFrontedUpData []float64
		for _, evt := range restartEvents {
			evtSeries = append(evtSeries, RestartEntry{Id: evt.Id,
				RestartDurationToMain:       (evt.StartMain.Ts - evt.Death.Ts) / StrTimeUnit2TimeUnit[timeUnit],
				RestartDurationToFrontendUp: (evt.StartFrontendUp.Ts - evt.Death.Ts) / StrTimeUnit2TimeUnit[timeUnit]})

			evtSeriesMainData = append(evtSeriesMainData, float64(evtSeries[len(evtSeries)-1].RestartDurationToMain))
			evtSeriesFrontedUpData = append(evtSeriesFrontedUpData, float64(evtSeries[len(evtSeries)-1].RestartDurationToFrontendUp))
		}

		result.Entries = append(result.Entries, SeriesEntry{Mark: mark})
		lastSeries := &result.Entries[len(result.Entries)-1]

		if min, err := stats.Min(evtSeriesMainData); err == nil {
			lastSeries.Min2Main = uint64(min)
		}

		if max, err := stats.Max(evtSeriesMainData); err == nil {
			lastSeries.Max2Main = uint64(max)
		}

		if min, err := stats.Min(evtSeriesFrontedUpData); err == nil {
			lastSeries.Min2FrontUp = uint64(min)
		}

		if max, err := stats.Max(evtSeriesFrontedUpData); err == nil {
			lastSeries.Max2FrontUp = uint64(max)
		}

		if fullSeries {
			lastSeries.EvtSeries = evtSeries
		}

	}
	c.JSON(http.StatusOK, result)
}

func probe(c *gin.Context) {
	if restarts, err := strconv.ParseUint(c.Query("restarts"), 0, 32); err == nil {
		Prb.CurrentPendingRestarts = uint(restarts)
	} else {
		Logger.Errorf("malformed probe:%s", err.Error())
		c.String(http.StatusBadRequest, err.Error())
	}

	Prb.CurrentDeathType = c.Query("how")
	Prb.CurrentMark = c.Query("mark")

	Prb.RequestDie()
}

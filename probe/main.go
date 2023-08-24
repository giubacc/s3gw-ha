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
	"flag"
	"net/http"
	. "s3gw-ha/probe/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// logger
var Logger *logrus.Logger
var Cfg Config

type Probe struct {
	CurrentDeath *DeathEvent
	CurrentStart *StartEvent

	RestartSet map[int]RestartEvent
}

func (p *Probe) submitDeath(evt *DeathEvent) {
	if p.CurrentDeath != nil {
		Logger.Error("bad state for submitting a death event")
		return
	}
	p.CurrentDeath = evt
}

func (p *Probe) submitStart(evt *StartEvent) {
	if p.CurrentStart != nil {
		Logger.Error("bad state for submitting a start event")
		return
	}
	p.CurrentStart = evt
	p.submitRestart()
}

func (p *Probe) submitRestart() {
	p.RestartSet[len(p.RestartSet)+1] = RestartEvent{Death: p.CurrentDeath, Start: p.CurrentStart}
	Logger.Infof("inserted restart event: death:%d, start:%d; collected events:%d", p.CurrentDeath.Ts, p.CurrentStart.Ts, len(p.RestartSet))
	p.CurrentDeath = nil
	p.CurrentStart = nil
}

var Prober Probe

func main() {
	flag.StringVar(&Cfg.LogLevel, "v", "inf", "Specify logging verbosity [off, trc, inf, wrn, err]")
	flag.UintVar(&Cfg.VerbLevel, "vl", 5, "Verbosity level")

	flag.Parse()

	Prober.RestartSet = make(map[int]RestartEvent)

	Logger = GetLogger(&Cfg)

	router := gin.Default()

	router.PUT("/death", setDeath)
	router.PUT("/start", setStart)
	router.GET("/stats", getStats)

	Logger.Info("start listening and serving ...")
	router.Run() // listen and serve on 0.0.0.0:8080
}

func setDeath(c *gin.Context) {
	deathType := c.Query("type")
	ts := c.Query("ts")
	if ts, err := strconv.ParseUint(ts, 0, 64); err == nil {
		evt := DeathEvent{Type: deathType, Ts: ts}
		Prober.submitDeath(&evt)
	} else {
		Logger.Errorf("malformed DeathEvent:%s", err.Error())
		c.String(http.StatusBadRequest, err.Error())
	}
}

func setStart(c *gin.Context) {
	ts := c.Query("ts")
	if ts, err := strconv.ParseUint(ts, 0, 64); err == nil {
		evt := StartEvent{Ts: ts}
		Prober.submitStart(&evt)
	} else {
		Logger.Errorf("malformed StartEvent:%s", err.Error())
		c.String(http.StatusBadRequest, err.Error())
	}
}

func getStats(c *gin.Context) {
}

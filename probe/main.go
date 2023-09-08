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
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	flag.StringVar(&Cfg.EndpointS3GW, "s3gw-endpoint", "http://localhost:7480", "Specify s3gw endpoint")
	flag.StringVar(&Cfg.SaveDataS3Endpoint, "save-data-endpoint", "http://localhost:7482", "Specify the S3 endpoint where to save results")
	flag.BoolVar(&Cfg.SaveDataS3ForcePathStyle, "save-data-path-style", true, "Force the S3 Path Style")
	flag.StringVar(&Cfg.SaveDataBucket, "save-data-bucket", "s3gw-ha-testing", "The bucket where to save results")
	flag.UintVar(&Cfg.WaitMSecsBeforeTriggerDeath, "wbtd", 100, "Wait n milliseconds before trigger death")
	flag.StringVar(&Cfg.CollectRestartAtEvent, "collectAt", "frontend-up", "The event where the probe should collect a restart event")
	flag.StringVar(&Cfg.LogLevel, "v", "inf", "Specify logging verbosity [off, trc, inf, wrn, err]")
	flag.UintVar(&Cfg.VerbLevel, "vl", 5, "Verbosity level")

	flag.Parse()

	Prb.CollectedRestartRelatedData = make(map[string][]RestartEvent)

	Logger = GetLogger(&Cfg)

	//S3Client

	S3Client = InitS3Client()

	if err := CreateBucket(S3Client, Cfg.SaveDataBucket); err != nil {
		Logger.Errorf("CreateBucket:%s", err.Error())
	}

	//GIN

	router := gin.Default()

	router.PUT("/death", setDeath)
	router.PUT("/start", setStart)
	router.GET("/stats", computeStats)
	router.PUT("/trigger", trigger)
	router.POST("/clear", clear)

	Logger.Info("start listening and serving ...")
	router.Run() // listen and serve on 0.0.0.0:8080
}

func setDeath(c *gin.Context) {
	deathType := c.Query("type")
	ts := c.Query("ts")
	if ts, err := strconv.ParseUint(ts, 0, 64); err == nil {
		evt := DeathEvent{Type: deathType, Ts: ts}
		Prb.SubmitDeath(&evt)
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
		Prb.SubmitStart(&evt)
	} else {
		Logger.Errorf("malformed StartEvent:%s", err.Error())
		c.String(http.StatusBadRequest, err.Error())
	}
}

func computeStats(c *gin.Context) {
	mark := c.Query("mark")
	if mark == "" {
		mark = "all"
	}
	timeUnit := c.Query("time_unit")
	if timeUnit == "" {
		timeUnit = StrSec
	}

	dumpAllData := false
	dumpAllDataPar := c.Query("full_series")
	if dumpAllDataPar == "true" {
		dumpAllData = true
	}

	genTS := strconv.Itoa(int(time.Now().Unix()))
	stats := Prb.ComputeStats(mark, timeUnit, dumpAllData)

	fNames := Prb.Render(genTS, timeUnit, stats)
	SendStatsArtifactsToS3(S3Client, Cfg.SaveDataBucket, fNames)

	c.JSON(http.StatusOK, stats)
}

func trigger(c *gin.Context) {
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

func clear(c *gin.Context) {
	Prb.Clear()
}

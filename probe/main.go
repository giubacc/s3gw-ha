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
	v1 "k8s.io/api/core/v1"
)

func main() {
	flag.StringVar(&Cfg.S3GWEndpoint, "s3gw-endpoint", "http://localhost:7480", "Specify the s3gw endpoint")
	flag.StringVar(&Cfg.S3GWNamespace, "s3gw-ns", "s3gw-ha", "Specify the s3gw namespace")
	flag.StringVar(&Cfg.S3GWDeployment, "s3gw-d", "s3gw-ha", "Specify the s3gw deployment name")
	flag.BoolVar(&Cfg.S3GWS3ForcePathStyle, "s3gw-path-style", true, "Force the s3gw S3 Path Style")
	flag.StringVar(&Cfg.SaveDataS3Endpoint, "save-data-endpoint", "http://localhost:7482", "Specify the save-data endpoint to save results")
	flag.BoolVar(&Cfg.SaveDataS3ForcePathStyle, "save-data-path-style", true, "Force the save-data S3 Path Style")
	flag.StringVar(&Cfg.SaveDataBucket, "save-data-bucket", "s3gw-ha-testing", "The bucket where to save results")
	flag.UintVar(&Cfg.WaitMSecsBeforeTriggerDeath, "wbtd", 100, "Wait n milliseconds before trigger death")
	flag.StringVar(&Cfg.CollectRestartAtEvent, "collectAt", "frontend-up", "The event where the probe should collect a restart event")
	flag.StringVar(&Cfg.LogLevel, "v", "inf", "Specify logging verbosity [off, trc, inf, wrn, err]")
	flag.UintVar(&Cfg.VerbLevel, "vl", 5, "Verbosity level")

	flag.Parse()

	Prb.CollectedRestartRelatedData = make(map[string][]RestartEvent)

	Logger = GetLogger(&Cfg)

	//S3Clients

	S3Client_S3GW = InitS3Client_S3GW()
	S3Client_SaveData = InitS3Client_SaveData()

	if err := CreateBucket(S3Client_SaveData, Cfg.SaveDataBucket); err != nil {
		Logger.Errorf("CreateBucket:%s", err.Error())
	}

	//K8s client

	K8sCli.Init()

	//GIN

	router := gin.Default()

	router.PUT("/death", setDeath)
	router.PUT("/start", setStart)
	router.GET("/stats", computeStats)
	router.PUT("/trigger", trigger)
	router.POST("/clear", clear)
	router.POST("/fill", fill)
	router.POST("/set_replicas", set_replicas)
	router.POST("/set_taint", set_taint)

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
	SendStatsArtifactsToS3(S3Client_SaveData, Cfg.SaveDataBucket, fNames)

	c.JSON(http.StatusOK, stats)
}

func trigger(c *gin.Context) {
	gracePeriod := c.Query("grace")
	if val, err := strconv.ParseUint(gracePeriod, 0, 32); err == nil {
		Prb.CurrentGracePeriod = uint(val)
	} else {
		Prb.CurrentGracePeriod = 0
	}
	if restarts, err := strconv.ParseUint(c.Query("restarts"), 0, 32); err == nil {
		Prb.CurrentPendingRestarts = uint(restarts)
	} else {
		Logger.Errorf("malformed probe:%s", err.Error())
		c.String(http.StatusBadRequest, err.Error())
	}

	Prb.CurrentDeathType = c.Query("how")
	Prb.CurrentMark = c.Query("mark")

	switch interposeFunc := c.Query("interpose"); interposeFunc {
	case "fill":
		Prb.CurrentGinCtx = c.Copy()
		Prb.CurrentInterposeFunc = fill
	}

	Prb.RequestDie()
}

func clear(c *gin.Context) {
	Prb.Clear()
}

func fill(c *gin.Context) {
	bucket := c.Query("bucket")
	objBaseName := c.Query("obj_base_name")
	payload := c.Query("payload")
	objCountPar := c.Query("obj_count")
	erase := c.Query("erase")
	addTSPar := c.Query("add-ts")
	addTS := addTSPar == "1"
	if objCount, err := strconv.ParseUint(objCountPar, 0, 64); err == nil {
		FillWithObjects(S3Client_S3GW, bucket, objBaseName, payload, objCount, addTS)
		if erase == "1" {
			EraseObjects(S3Client_S3GW, bucket, objBaseName, objCount)
		}
	} else {
		Logger.Errorf("malformed objCountPar:%s", err.Error())
		c.String(http.StatusBadRequest, err.Error())
	}
}

func set_replicas(c *gin.Context) {
	namespace := c.Query("ns")
	deployment := c.Query("d_name")
	replicasPar := c.Query("replicas")
	if replicas, err := strconv.ParseInt(replicasPar, 0, 64); err == nil {
		K8sCli.SetReplicasForDeployment(namespace, deployment, int32(replicas))
	} else {
		Logger.Errorf("malformed replicas:%s", err.Error())
		c.String(http.StatusBadRequest, err.Error())
	}
}

func set_taint(c *gin.Context) {
	node := c.Query("node")
	key := c.Query("key")
	val := c.Query("val")
	effect := c.Query("effect")
	remove := c.Query("remove")
	if remove == "1" {
		if err := K8sCli.UnsetTaint(node, key, val, v1.TaintEffect(effect)); err != nil {
			Logger.Errorf("UnsetTaint:%s", err.Error())
			c.String(http.StatusInternalServerError, err.Error())
		}
	} else {
		if err := K8sCli.SetTaint(node, key, val, v1.TaintEffect(effect)); err != nil {
			Logger.Errorf("SetTaint:%s", err.Error())
			c.String(http.StatusInternalServerError, err.Error())
		}
	}
}

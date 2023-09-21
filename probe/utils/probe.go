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
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/gin-gonic/gin"
	"github.com/igrmk/treemap/v2"
	"github.com/montanaflynn/stats"
	v1 "k8s.io/api/core/v1"
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

var StrTimeUnit2TimeUnit = map[string]int64{
	StrNanoS:  NanoS,
	StrMicroS: MicroS,
	StrMilliS: MilliS,
	StrSec:    Sec}

type RestartRelatedData map[string][]RestartEvent

type S3WorkloadConfig struct {
	FuncName  string
	FuncArgs  map[string]string
	Frequency uint //msec
}

func (cfg *S3WorkloadConfig) Reset() {
	cfg.FuncName = ""
	cfg.Frequency = 0
	if cfg.FuncArgs != nil {
		for k := range cfg.FuncArgs {
			delete(cfg.FuncArgs, k)
		}
	}
}

func (cfg *S3WorkloadConfig) GetFArgsMap(args string) error {
	if cfg.FuncArgs == nil {
		cfg.FuncArgs = make(map[string]string)
	} else {
		for k := range cfg.FuncArgs {
			delete(cfg.FuncArgs, k)
		}
	}

	if args != "" {
		for _, parVal := range strings.Split(args, ",") {
			tokens := strings.Split(parVal, "=")
			if len(tokens) == 2 {
				cfg.FuncArgs[tokens[0]] = tokens[1]
				Logger.Infof("--- %s:%s", tokens[0], tokens[1])
			} else {
				Logger.Error("malformed FuncArgs")
			}
		}
	}

	return nil
}

type S3WorkloadRelatedData map[string]*treemap.TreeMap[int64, S3WorkloadEvent]

type Probe struct {
	CurrentDeath     *DeathEvent
	CurrentStartList []*StartEvent

	CurrentPendingRestarts   uint
	CurrentGracePeriod       uint
	CurrentDeathType         string
	CurrentMark              string
	CurrentId                int
	CurrentInterposeFunc     func(*gin.Context)
	CurrentGinCtx            *gin.Context
	CurrentNodeNameList      *[]string
	CurrentNodeNameActiveIdx uint
	CurrentSelectedNode      string
	CurrentSelectedNodeSet   bool
	CurrentS3WorkloadCfg     S3WorkloadConfig
	CurrentS3WorkloadStarted bool
	CurrentS3WorkId          int

	CollectedRestartRelatedData    RestartRelatedData
	CollectedS3WorkloadRelatedData S3WorkloadRelatedData

	S3WorkloadEvtChan chan string
}

func (p *Probe) ResetCurrentState() {
	p.CurrentPendingRestarts = 0
	p.CurrentGracePeriod = 0
	p.CurrentDeathType = ""
	p.CurrentMark = ""
	p.CurrentId = 0
	p.CurrentInterposeFunc = nil
	p.CurrentGinCtx = nil
	if p.CurrentNodeNameList != nil {
		p.SetK8sScheduleAllNodes()
	}
	p.CurrentNodeNameList = nil
	p.CurrentNodeNameActiveIdx = 0
	p.CurrentSelectedNode = ""
	p.CurrentSelectedNodeSet = false

	p.CurrentS3WorkloadCfg.Reset()
	p.CurrentS3WorkloadStarted = false
	p.CurrentS3WorkId = 0
}

func (p *Probe) Clear() {
	p.ResetCurrentState()
	for k := range p.CollectedRestartRelatedData {
		delete(p.CollectedRestartRelatedData, k)
	}
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
			if p.CurrentGracePeriod > 0 {
				Logger.Infof("waiting %d ms...", p.CurrentGracePeriod)
				time.Sleep(time.Duration(p.CurrentGracePeriod) * time.Millisecond)
			}
			if p.CurrentInterposeFunc != nil && p.CurrentGinCtx != nil {
				p.CurrentInterposeFunc(p.CurrentGinCtx)
			}
			p.RequestDie()
		} else {

			if p.CurrentS3WorkloadStarted {
				Logger.Infof("asking S3 workload to stop ...")
				p.S3WorkloadEvtChan <- "stop"
			}

			if p.CurrentMark == "unsolicited" {
				return
			}

			//wrap up results and send those to S3

			timeUnit := "ms"
			genTS := strconv.Itoa(int(time.Now().Unix()))
			stats := Stats{TimeUnit: timeUnit}

			p.ComputeRestartStats(&stats, p.CurrentMark, timeUnit, true)
			p.ComputeS3WorkloadStats(&stats, p.CurrentMark, timeUnit, true)

			SendStatsArtifactsToS3(S3Client_SaveData, Cfg.SaveDataBucket, p.Render(genTS, timeUnit, stats))

			p.ResetCurrentState()
		}
	}
}

func (p *Probe) Render(genTS string, timeUnit string, stats Stats) []string {
	fNames := []string{}

	fStat, _ := SaveStats(p.CurrentMark, genTS, stats)
	fRestartRaw, _ := p.GenerateRestartRawDataPlot(timeUnit, p.CurrentMark, genTS)
	fRestartP1, fRestartP2, fRestartP3, _ := p.GenerateRestartPercentilesPlot(timeUnit, p.CurrentMark, genTS)

	fS3WLRaw, _ := p.GenerateS3WorkloadRawDataPlot(timeUnit, p.CurrentMark, genTS)

	fNames = append(fNames, fStat, fRestartRaw, fRestartP1, fRestartP2, fRestartP3, fS3WLRaw)

	return fNames
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
	if p.CurrentMark == "" {
		p.CurrentMark = "unsolicited"
	}

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

func (p *Probe) AskRadosgwToDie() {
	req, err := http.NewRequest("PUT", Cfg.S3GWEndpoint, bytes.NewReader([]byte("")))
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

func (p *Probe) AskK8sScaleDeployment_0_1() {
	K8sCli.SetReplicasForDeployment(Cfg.S3GWNamespace, Cfg.S3GWDeployment, 0)
	Logger.Info("set replicas=0")
	time.Sleep(time.Duration(Cfg.WaitMSecsBeforeSetReplicas1) * time.Millisecond)
	K8sCli.SetReplicasForDeployment(Cfg.S3GWNamespace, Cfg.S3GWDeployment, 1)
	Logger.Info("set replicas=1")
}

func (p *Probe) SetK8sScheduleNextNode() {
	activeIdx := (p.CurrentNodeNameActiveIdx + 1) % uint(len(*p.CurrentNodeNameList))
	for idx, node := range *p.CurrentNodeNameList {
		if idx != int(activeIdx) {
			K8sCli.SetTaint(node, "noSch", "1", v1.TaintEffectNoSchedule)
		} else {
			K8sCli.UnsetTaint(node, "noSch", "1", v1.TaintEffectNoSchedule)
		}
	}
	p.CurrentNodeNameActiveIdx = activeIdx
}

func (p *Probe) SetK8sNoScheduleAllNodes() {
	nodeNameList, _ := K8sCli.GetNodeNameList()
	for _, node := range *nodeNameList {
		K8sCli.SetTaint(node, "noSch", "1", v1.TaintEffectNoSchedule)
	}
}

func (p *Probe) SetK8sScheduleAllNodes() {
	nodeNameList, _ := K8sCli.GetNodeNameList()
	for _, node := range *nodeNameList {
		K8sCli.UnsetTaint(node, "noSch", "1", v1.TaintEffectNoSchedule)
	}
}

func (p *Probe) RequestDie() {
	var err error
	if !p.CurrentS3WorkloadStarted {
		if p.CurrentS3WorkloadStarted, err = p.TriggerS3ClientWorkload(); err != nil {
			Logger.Errorf("TriggerS3ClientWorkload:%s", err.Error())
		}
	}

	if p.CurrentSelectedNode != "" && !p.CurrentSelectedNodeSet {
		p.SetK8sNoScheduleAllNodes()
		K8sCli.UnsetTaint(p.CurrentSelectedNode, "noSch", "1", v1.TaintEffectNoSchedule)
		p.CurrentSelectedNodeSet = true
	}

	switch p.CurrentDeathType {
	case "k8s_scale_deployment_0_1":
		p.AskK8sScaleDeployment_0_1()
	case "k8s_scale_deployment_0_1_node_rr":
		if p.CurrentNodeNameList == nil {
			var err error
			if p.CurrentNodeNameList, err = K8sCli.GetNodeNameList(); err != nil {
				Logger.Errorf("GetNodeNameList:%s", err.Error())
				return
			}
		}
		p.SetK8sScheduleNextNode()
		p.AskK8sScaleDeployment_0_1()
	default:
		p.AskRadosgwToDie()
	}
}

func (p *Probe) RunS3ClientWorkload_SendObject() {
	Logger.Infof("workload started")

	ticker := time.NewTicker(time.Millisecond * time.Duration(p.CurrentS3WorkloadCfg.Frequency))
	defer ticker.Stop()

	bucketName := p.CurrentS3WorkloadCfg.FuncArgs["bn"]
	objName := p.CurrentS3WorkloadCfg.FuncArgs["on"]
	payload := p.CurrentS3WorkloadCfg.FuncArgs["pl"]

out:
	for {
		select {
		case <-ticker.C:
			start, end, err := SendObject(S3Client_S3GW, bucketName, objName, payload)
			p.CurrentS3WorkId++

			if p.CollectedS3WorkloadRelatedData[p.CurrentMark] == nil {
				p.CollectedS3WorkloadRelatedData[p.CurrentMark] = treemap.New[int64, S3WorkloadEvent]()
			}

			s3WorkloadEvent := S3WorkloadEvent{Id: p.CurrentS3WorkId, StartTs: start, EndTs: end, Error: err}
			p.CollectedS3WorkloadRelatedData[p.CurrentMark].Set(start, s3WorkloadEvent)

			if p.CurrentS3WorkId%100 == 0 {
				Logger.Infof("CollectedS3WorkloadRelatedData[%s] %d", p.CurrentMark, p.CollectedS3WorkloadRelatedData[p.CurrentMark].Len())
			}

		case evt := <-p.S3WorkloadEvtChan:
			switch evt {
			case "stop":
				Logger.Infof("workload stopped")
				break out
			}
		}
	}

	Logger.Infof("CollectedS3WorkloadRelatedData[%s] %d", p.CurrentMark, p.CollectedS3WorkloadRelatedData[p.CurrentMark].Len())
}

func (p *Probe) TriggerS3ClientWorkload() (bool, error) {
	started := false
	if p.CurrentS3WorkloadCfg.FuncName != "" {
		switch p.CurrentS3WorkloadCfg.FuncName {
		case "SendObject":
			go p.RunS3ClientWorkload_SendObject()
			started = true
		default:
			return false, errors.New("UnavailableWorkload")
		}
	}
	return started, nil
}

func GetSplitDataForSingleRestartRelatedData(restartEvents []RestartEvent, timeUnit int64) ([]RestartEntry, []float64, []float64, []float64) {
	var evtSeries []RestartEntry
	var evtSeriesMainData []float64
	var evtSeriesFrontedUpData []float64
	var evtSeriesFUpMainDelta []float64
	for _, evt := range restartEvents {
		evtSeries = append(evtSeries, RestartEntry{Id: evt.Id,
			RestartDurationToMain:       (evt.StartMain.Ts - evt.Death.Ts) / timeUnit,
			RestartDurationToFrontendUp: (evt.StartFrontendUp.Ts - evt.Death.Ts) / timeUnit,
			FUpMainDelta:                (evt.StartFrontendUp.Ts - evt.StartMain.Ts) / timeUnit})

		evtSeriesMainData = append(evtSeriesMainData, float64(evtSeries[len(evtSeries)-1].RestartDurationToMain))
		evtSeriesFrontedUpData = append(evtSeriesFrontedUpData, float64(evtSeries[len(evtSeries)-1].RestartDurationToFrontendUp))
		evtSeriesFUpMainDelta = append(evtSeriesFUpMainDelta, float64(evtSeries[len(evtSeries)-1].FUpMainDelta))
	}
	return evtSeries, evtSeriesMainData, evtSeriesFrontedUpData, evtSeriesFUpMainDelta
}

func (p *Probe) ComputeRestartStats(sts *Stats, markPar string, timeUnit string, dumpAllData bool) *Stats {
	for mark, restartEvents := range Prb.CollectedRestartRelatedData {

		if markPar != "all" && markPar != mark {
			continue
		}

		evtSeries,
			evtSeriesMainData,
			evtSeriesFrontedUpData,
			evtSeriesFUpMainDelta := GetSplitDataForSingleRestartRelatedData(restartEvents, StrTimeUnit2TimeUnit[timeUnit])

		sts.SeriesRestart = append(sts.SeriesRestart, SeriesRestartEntry{Mark: mark})
		lastSeries := &sts.SeriesRestart[len(sts.SeriesRestart)-1]

		if val, err := stats.Min(evtSeriesMainData); err == nil {
			lastSeries.MinMain = int64(val)
		}

		if val, err := stats.Max(evtSeriesMainData); err == nil {
			lastSeries.MaxMain = int64(val)
		}

		if val, err := stats.Mean(evtSeriesMainData); err == nil {
			lastSeries.MeanMain = int64(val)
		}

		if val, err := stats.Percentile(evtSeriesMainData, 99); err == nil {
			lastSeries.Perc99Main = int64(val)
		}

		if val, err := stats.Percentile(evtSeriesMainData, 95); err == nil {
			lastSeries.Perc95Main = int64(val)
		}

		if val, err := stats.PercentileNearestRank(evtSeriesMainData, 99); err == nil {
			lastSeries.PercNR99Main = int64(val)
		}

		if val, err := stats.PercentileNearestRank(evtSeriesMainData, 95); err == nil {
			lastSeries.PercNR95Main = int64(val)
		}

		if val, err := stats.Min(evtSeriesFrontedUpData); err == nil {
			lastSeries.MinFrontUp = int64(val)
		}

		if val, err := stats.Max(evtSeriesFrontedUpData); err == nil {
			lastSeries.MaxFrontUp = int64(val)
		}

		if val, err := stats.Mean(evtSeriesFrontedUpData); err == nil {
			lastSeries.MeanFrontUp = int64(val)
		}

		if val, err := stats.Percentile(evtSeriesFrontedUpData, 99); err == nil {
			lastSeries.Perc99FrontUp = int64(val)
		}

		if val, err := stats.Percentile(evtSeriesFrontedUpData, 95); err == nil {
			lastSeries.Perc95FrontUp = int64(val)
		}

		if val, err := stats.PercentileNearestRank(evtSeriesFrontedUpData, 99); err == nil {
			lastSeries.PercNR99FrontUp = int64(val)
		}

		if val, err := stats.PercentileNearestRank(evtSeriesFrontedUpData, 95); err == nil {
			lastSeries.PercNR95FrontUp = int64(val)
		}

		if val, err := stats.Min(evtSeriesFUpMainDelta); err == nil {
			lastSeries.MinFUpMainD = int64(val)
		}

		if val, err := stats.Max(evtSeriesFUpMainDelta); err == nil {
			lastSeries.MaxFUpMainD = int64(val)
		}

		if val, err := stats.Mean(evtSeriesFUpMainDelta); err == nil {
			lastSeries.MeanFUpMainD = int64(val)
		}

		if val, err := stats.Percentile(evtSeriesFUpMainDelta, 99); err == nil {
			lastSeries.Perc99FUpMainD = int64(val)
		}

		if val, err := stats.Percentile(evtSeriesFUpMainDelta, 95); err == nil {
			lastSeries.Perc95FUpMainD = int64(val)
		}

		if val, err := stats.PercentileNearestRank(evtSeriesFUpMainDelta, 99); err == nil {
			lastSeries.PercNR99FUpMainD = int64(val)
		}

		if val, err := stats.PercentileNearestRank(evtSeriesFUpMainDelta, 95); err == nil {
			lastSeries.PercNR95FUpMainD = int64(val)
		}

		if dumpAllData {
			lastSeries.Data = evtSeries
		}
	}
	return sts
}

func SaveStats(mark string, genTS string, stats Stats) (string, error) {
	if resultFile, err := json.MarshalIndent(stats, "", " "); err != nil {
		Logger.Errorf("json.MarshalIndent:%s", err.Error())
		return "", err
	} else {
		fName := mark + "_stats_" + genTS + ".json"
		if err := os.WriteFile(fName, resultFile, 0644); err != nil {
			Logger.Errorf("os.WriteFile:%s", err.Error())
			return "", err
		}
		return fName, nil
	}
}

func GetSplitDataForSingleS3WorkloadRelatedData(restartEvents *treemap.TreeMap[int64, S3WorkloadEvent], timeUnit int64) ([]S3WorkloadEntry, []float64) {
	var evtSeries []S3WorkloadEntry
	var evtSeriesRTTData []float64
	for it := restartEvents.Iterator(); it.Valid(); it.Next() {
		val := it.Value()
		evtSeries = append(evtSeries, S3WorkloadEntry{Id: val.Id,
			Start: val.StartTs,
			End:   val.EndTs,
			RTT:   (val.EndTs - val.StartTs) / timeUnit})

		evtSeriesRTTData = append(evtSeriesRTTData, float64(evtSeries[len(evtSeries)-1].RTT))
	}
	return evtSeries, evtSeriesRTTData
}

func (p *Probe) ComputeS3WorkloadStats(sts *Stats, markPar string, timeUnit string, dumpAllData bool) *Stats {
	for mark, s3WLEvents := range Prb.CollectedS3WorkloadRelatedData {

		if markPar != "all" && markPar != mark {
			continue
		}

		evtSeries,
			evtSeriesRTTData := GetSplitDataForSingleS3WorkloadRelatedData(s3WLEvents, StrTimeUnit2TimeUnit[timeUnit])

		sts.SeriesS3Workload = append(sts.SeriesS3Workload, SeriesS3WorkloadEntry{Mark: mark})
		lastSeries := &sts.SeriesS3Workload[len(sts.SeriesS3Workload)-1]

		if val, err := stats.Min(evtSeriesRTTData); err == nil {
			lastSeries.MinRTT = int64(val)
		}

		if val, err := stats.Max(evtSeriesRTTData); err == nil {
			lastSeries.MaxRTT = int64(val)
		}

		if val, err := stats.Mean(evtSeriesRTTData); err == nil {
			lastSeries.MeanRTT = int64(val)
		}

		if val, err := stats.Percentile(evtSeriesRTTData, 99); err == nil {
			lastSeries.Perc99RTT = int64(val)
		}

		if val, err := stats.Percentile(evtSeriesRTTData, 95); err == nil {
			lastSeries.Perc95RTT = int64(val)
		}

		if val, err := stats.PercentileNearestRank(evtSeriesRTTData, 99); err == nil {
			lastSeries.PercNR99RTT = int64(val)
		}

		if val, err := stats.PercentileNearestRank(evtSeriesRTTData, 95); err == nil {
			lastSeries.PercNR95RTT = int64(val)
		}

		if dumpAllData {
			lastSeries.Data = evtSeries
		}
	}

	return sts
}

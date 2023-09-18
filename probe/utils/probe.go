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
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/gin-gonic/gin"
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

var StrTimeUnit2TimeUnit = map[string]uint64{
	StrNanoS:  NanoS,
	StrMicroS: MicroS,
	StrMilliS: MilliS,
	StrSec:    Sec}

type RestartRelatedData map[string][]RestartEvent

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

	CollectedRestartRelatedData RestartRelatedData
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
				Logger.Infof("waiting %d secs...", p.CurrentGracePeriod)
				time.Sleep(time.Duration(p.CurrentGracePeriod) * time.Second)
			}
			if p.CurrentInterposeFunc != nil && p.CurrentGinCtx != nil {
				p.CurrentInterposeFunc(p.CurrentGinCtx)
			}
			p.RequestDie()
		} else {

			if p.CurrentMark == "unsolicited" {
				return
			}

			timeUnit := "ms"
			genTS := strconv.Itoa(int(time.Now().Unix()))
			stats := p.ComputeStats(p.CurrentMark, timeUnit, true)

			SendStatsArtifactsToS3(S3Client_SaveData, Cfg.SaveDataBucket, p.Render(genTS, timeUnit, stats))

			p.ResetCurrentState()
		}
	}
}

func (p *Probe) Render(genTS string, timeUnit string, stats Stats) []string {
	fNames := []string{}

	fStat, _ := SaveStats(p.CurrentMark, genTS, stats)
	fRaw, _ := GenerateRawDataPlot(Prb.CollectedRestartRelatedData, p.CurrentMark, timeUnit, p.CurrentMark, genTS)
	fP1, fP2, fP3, _ := GeneratePercentilesPlot(Prb.CollectedRestartRelatedData, p.CurrentMark, timeUnit, p.CurrentMark, genTS)

	fNames = append(fNames, fStat, fRaw, fP1, fP2, fP3)

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

func GetSplitDataForSingleRestartRelatedData(restartEvents []RestartEvent, timeUnit uint64) ([]RestartEntry, []float64, []float64, []float64) {
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

func (p *Probe) ComputeStats(markPar string, timeUnit string, dumpAllData bool) Stats {
	result := Stats{TimeUnit: timeUnit}
	for mark, restartEvents := range Prb.CollectedRestartRelatedData {

		if markPar != "all" && markPar != mark {
			continue
		}

		evtSeries,
			evtSeriesMainData,
			evtSeriesFrontedUpData,
			evtSeriesFUpMainDelta := GetSplitDataForSingleRestartRelatedData(restartEvents, StrTimeUnit2TimeUnit[timeUnit])

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

		if val, err := stats.Min(evtSeriesFUpMainDelta); err == nil {
			lastSeries.MinFUpMainD = uint64(val)
		}

		if val, err := stats.Max(evtSeriesFUpMainDelta); err == nil {
			lastSeries.MaxFUpMainD = uint64(val)
		}

		if val, err := stats.Mean(evtSeriesFUpMainDelta); err == nil {
			lastSeries.MeanFUpMainD = uint64(val)
		}

		if val, err := stats.Percentile(evtSeriesFUpMainDelta, 99); err == nil {
			lastSeries.Perc99FUpMainD = uint64(val)
		}

		if val, err := stats.Percentile(evtSeriesFUpMainDelta, 95); err == nil {
			lastSeries.Perc95FUpMainD = uint64(val)
		}

		if val, err := stats.PercentileNearestRank(evtSeriesFUpMainDelta, 99); err == nil {
			lastSeries.PercNR99FUpMainD = uint64(val)
		}

		if val, err := stats.PercentileNearestRank(evtSeriesFUpMainDelta, 95); err == nil {
			lastSeries.PercNR95FUpMainD = uint64(val)
		}

		if dumpAllData {
			lastSeries.Data = evtSeries
		}
	}
	return result
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

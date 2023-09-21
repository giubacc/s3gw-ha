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

type Config struct {
	//logging
	LogLevel                    string
	VerbLevel                   uint
	S3GWEndpoint                string
	S3GWNamespace               string
	S3GWDeployment              string
	S3GWS3ForcePathStyle        bool
	WaitMSecsBeforeTriggerDeath uint //msec
	WaitMSecsBeforeSetReplicas1 uint //msec
	CollectRestartAtEvent       string
	SaveDataS3Endpoint          string
	SaveDataS3ForcePathStyle    bool
	SaveDataBucket              string
}

type S3WorkloadEvent struct {
	Id      int
	StartTs int64 `json:"start_ts"` //start timestamp of this event
	EndTs   int64 `json:"end_ts"`   //end timestamp of this event
	Error   error `json:"error"`    //error for this event
}

type DeathEvent struct {
	Type string `json:"type"`
	Ts   int64  `json:"ts"` //timestamp of this event
}

type StartEvent struct {
	Ts    int64  `json:"ts"`    //timestamp of this event
	Where string `json:"where"` //point where the radosgw sent the event
}

type RestartEvent struct {
	Id              int
	Death           *DeathEvent
	StartMain       *StartEvent
	StartFrontendUp *StartEvent
}

type RestartEntry struct {
	Id                          int   `json:"restart_id"`
	RestartDurationToMain       int64 `json:"duration_to_main"`
	RestartDurationToFrontendUp int64 `json:"duration_to_frontend_up"`
	FUpMainDelta                int64 `json:"frontend_up_main_delta"`
}

type SeriesRestartEntry struct {
	Mark             string         `json:"mark"`
	MinMain          int64          `json:"min_to_main"`
	MaxMain          int64          `json:"max_to_main"`
	MeanMain         int64          `json:"mean_to_main"`
	Perc99Main       int64          `json:"99p_to_main"`
	Perc95Main       int64          `json:"95p_to_main"`
	PercNR99Main     int64          `json:"99pNR_to_main"`
	PercNR95Main     int64          `json:"95pNR_to_main"`
	MinFrontUp       int64          `json:"min_to_frontend_up"`
	MaxFrontUp       int64          `json:"max_to_frontend_up"`
	MeanFrontUp      int64          `json:"mean_to_frontend_up"`
	Perc99FrontUp    int64          `json:"99p_to_frontend_up"`
	Perc95FrontUp    int64          `json:"95p_to_frontend_up"`
	PercNR99FrontUp  int64          `json:"99pNR_to_frontend_up"`
	PercNR95FrontUp  int64          `json:"95pNR_to_frontend_up"`
	MinFUpMainD      int64          `json:"min_frontend_up_main_delta"`
	MaxFUpMainD      int64          `json:"max_frontend_up_main_delta"`
	MeanFUpMainD     int64          `json:"mean_frontend_up_main_delta"`
	Perc99FUpMainD   int64          `json:"99p_frontend_up_main_delta"`
	Perc95FUpMainD   int64          `json:"95p_frontend_up_main_delta"`
	PercNR99FUpMainD int64          `json:"99pNR_frontend_up_main_delta"`
	PercNR95FUpMainD int64          `json:"95pNR_frontend_up_main_delta"`
	Data             []RestartEntry `json:"data"`
}

type S3WorkloadEntry struct {
	Id    int     `json:"restart_id"`
	Start int64   `json:"start"`
	End   int64   `json:"end"`
	RTT   float64 `json:"rtt"`
	Err   error
}
type SeriesS3WorkloadEntry struct {
	Mark        string            `json:"mark"`
	MinRTT      int64             `json:"min_to_main"`
	MaxRTT      int64             `json:"max_to_main"`
	MeanRTT     int64             `json:"mean_to_main"`
	Perc99RTT   int64             `json:"99p_to_main"`
	Perc95RTT   int64             `json:"95p_to_main"`
	PercNR99RTT int64             `json:"99pNR_to_main"`
	PercNR95RTT int64             `json:"95pNR_to_main"`
	Data        []S3WorkloadEntry `json:"data"`
}

type Stats struct {
	SeriesRestartCount    uint                    `json:"series_restart_count"`
	SeriesRestart         []SeriesRestartEntry    `json:"series_restart"`
	SeriesS3WorkloadCount uint                    `json:"series_s3_workload_count"`
	SeriesS3Workload      []SeriesS3WorkloadEntry `json:"series_s3_workload"`
	TimeUnit              string                  `json:"time_unit"`
}

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
	EndpointS3GW                string
	WaitMSecsBeforeTriggerDeath uint //msec
	CollectRestartAtEvent       string
	SaveDataS3Endpoint          string
	SaveDataS3ForcePathStyle    bool
	SaveDataBucket              string
}

type DeathEvent struct {
	Type string `json:"type"`
	Ts   uint64 `json:"ts"` //timestamp of this event
}

type StartEvent struct {
	Ts    uint64 `json:"ts"`    //timestamp of this event
	Where string `json:"where"` //point where the radosgw sent the event
}

type RestartEvent struct {
	Id              int
	Death           *DeathEvent
	StartMain       *StartEvent
	StartFrontendUp *StartEvent
}

type RestartEntry struct {
	Id                          int    `json:"restart_id"`
	RestartDurationToMain       uint64 `json:"duration_to_main"`
	RestartDurationToFrontendUp uint64 `json:"duration_to_frontend_up"`
}

type SeriesEntry struct {
	Mark            string         `json:"mark"`
	MinMain         uint64         `json:"min_to_main"`
	MaxMain         uint64         `json:"max_to_main"`
	MeanMain        uint64         `json:"mean_to_main"`
	Perc99Main      uint64         `json:"99p_to_main"`
	Perc95Main      uint64         `json:"95p_to_main"`
	PercNR99Main    uint64         `json:"99pNR_to_main"`
	PercNR95Main    uint64         `json:"95pNR_to_main"`
	MinFrontUp      uint64         `json:"min_to_frontend_up"`
	MaxFrontUp      uint64         `json:"max_to_frontend_up"`
	MeanFrontUp     uint64         `json:"mean_to_frontend_up"`
	Perc99FrontUp   uint64         `json:"99p_to_frontend_up"`
	Perc95FrontUp   uint64         `json:"95p_to_frontend_up"`
	PercNR99FrontUp uint64         `json:"99pNR_to_frontend_up"`
	PercNR95FrontUp uint64         `json:"95pNR_to_frontend_up"`
	Data            []RestartEntry `json:"data"`
}

type Stats struct {
	SeriesCount uint          `json:"series_count"`
	Series      []SeriesEntry `json:"series"`
	TimeUnit    string        `json:"time_unit"`
}

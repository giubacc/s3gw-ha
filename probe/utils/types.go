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
	Mark        string         `json:"mark"`
	Min2Main    uint64         `json:"min_to_main"`
	Max2Main    uint64         `json:"max_to_main"`
	Min2FrontUp uint64         `json:"min_to_front_up"`
	Max2FrontUp uint64         `json:"max_to_front_up"`
	EvtSeries   []RestartEntry `json:"series"`
}

type Stats struct {
	Entries  []SeriesEntry `json:"probes"`
	TimeUnit string        `json:"time_unit"`
}

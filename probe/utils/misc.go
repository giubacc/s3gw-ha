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
	"net/http"
	"os"
	"strconv"
	"time"

	. "github.com/aws/aws-sdk-go/aws/credentials"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/sirupsen/logrus"
)

// logger
var Logger *logrus.Logger

// config
var Cfg Config

// probe
var Prb Probe

func EpochTime() time.Time { return time.Unix(0, 0) }

func NanoSecName(base string) string {
	return base + strconv.Itoa(int(time.Now().Nanosecond()))
}

type LogLevelStr string

const (
	TraceStr = "trc"
	InfoStr  = "inf"
	WarnStr  = "wrn"
	ErrStr   = "err"
	FatalStr = "fat"
)

var LogLevelStr2LvL = map[string]logrus.Level{
	TraceStr: logrus.TraceLevel,
	InfoStr:  logrus.InfoLevel,
	WarnStr:  logrus.WarnLevel,
	ErrStr:   logrus.ErrorLevel,
	FatalStr: logrus.FatalLevel}

func GetLogger(cfg *Config) *logrus.Logger {
	logger := &logrus.Logger{
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     LogLevelStr2LvL[cfg.LogLevel],
		Out:       os.Stdout,
	}

	return logger
}

func SignHTTPRequestV4(request *http.Request, creds *Credentials) error {
	signer := v4.NewSigner(creds)
	//_, err := signer.Sign(request, bytes.NewReader([]byte("")), "S3", "us-east-1", time.Now())
	_, err := signer.Sign(request, nil, "S3", "us-east-1", time.Now())
	if err != nil {
		return err
	}
	return nil
}

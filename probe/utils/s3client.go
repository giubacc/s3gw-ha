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
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var S3Client_SaveData *s3.S3
var S3Client_S3GW *s3.S3

func CreateBucket(client *s3.S3, bucketName string) error {
	_, err := client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})

	return err
}

func SendObject(client *s3.S3, bucketName string, objName string, payload string) error {
	if _, err := client.PutObject(&s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &objName,
		Body:   bytes.NewReader([]byte(payload))}); err != nil {
		Logger.Errorf("PutObject:%s", err.Error())
		return err
	}
	return nil
}

func EraseObject(client *s3.S3, bucketName string, objName string) error {
	if _, err := client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &bucketName,
		Key:    &objName}); err != nil {
		Logger.Errorf("DeleteObject:%s", err.Error())
		return err
	}
	return nil
}

func SendObjectFromFile(client *s3.S3, bucketName string, fileName string) error {
	if file, err := os.Open(fileName); err != nil {
		Logger.Errorf("Open:%s", err.Error())
		return err
	} else {
		if _, err := client.PutObject(&s3.PutObjectInput{Bucket: &bucketName, Key: &fileName, Body: file}); err != nil {
			Logger.Errorf("PutObject:%s", err.Error())
			return err
		}
	}
	return nil
}

func SendStatsArtifactsToS3(client *s3.S3, bucketName string, fNames []string) {
	for _, fName := range fNames {
		SendObjectFromFile(client, bucketName, fName)
	}
}

func FillWithObjects(client *s3.S3, bucketName string, objBaseName string, payload string, objCount uint64, addTS bool) error {
	CreateBucket(client, bucketName)
	for i := uint64(0); i < objCount; i++ {
		var objName string
		if addTS {
			objName = objBaseName + "_" + strconv.FormatUint(i, 10) + "_" + strconv.Itoa(int(time.Now().Unix()))
		} else {
			objName = objBaseName + "_" + strconv.FormatUint(i, 10)
		}

		SendObject(client, bucketName, objName, payload)
	}
	return nil
}

func EraseObjects(client *s3.S3, bucketName string, objBaseName string, objCount uint64) error {
	for i := uint64(0); i < objCount; i++ {
		objName := objBaseName + "_" + strconv.FormatUint(i, 10)
		EraseObject(client, bucketName, objName)
	}
	return nil
}

func InitS3Client_SaveData() *s3.S3 {
	session, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			S3ForcePathStyle: &Cfg.SaveDataS3ForcePathStyle,
			Endpoint:         &Cfg.SaveDataS3Endpoint,
			Region:           aws.String("US"),
		},
	})

	if err != nil {
		Logger.Errorf("InitS3Client_SaveData: Failed to initialize new session:%s", err.Error())
		return nil
	}

	s3Client := s3.New(session)
	return s3Client
}

func InitS3Client_S3GW() *s3.S3 {
	session, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			S3ForcePathStyle: &Cfg.S3GWS3ForcePathStyle,
			Endpoint:         &Cfg.S3GWEndpoint,
			Region:           aws.String("US"),
		},
	})

	if err != nil {
		Logger.Errorf("InitS3Client_S3GW: Failed to initialize new session:%s", err.Error())
		return nil
	}

	s3Client := s3.New(session)
	return s3Client
}

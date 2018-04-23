/*
Copyright (C) 2018 Expedia Group.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tests

import (
	"fmt"
	"github.com/HotelsDotCom/go-docker-client"
	"github.com/HotelsDotCom/go-logger"
	"net"
	"os"
	"strconv"
)

var flyteApiUrl string

const (
	flyteApiDefaultImage = "hotelsdotcom/flyte:1.81"
	flyteApiImageEnvName = "FLYTE_API_IMAGE"
)

type Flyte struct {
	flyteContainer docker.Container
}

func StartFlyte(mongo Mongo) (*Flyte, error) {
	flyteApiPort := getPort()
	flyteApiUrl = "http://localhost:" + flyteApiPort

	mongoHost, err := mongo.GetIP()
	if err != nil {
		return nil, err
	}

	d, err := docker.NewDocker()
	if err != nil {
		return nil, err
	}

	os.Setenv("FLYTE_API", flyteApiUrl)

	flyteContainer, err := d.Run("flyte", getFlyteImagePath(),
		[]string{fmt.Sprintf("FLYTE_MGO_HOST=%s", mongoHost), fmt.Sprintf("FLYTE_PORT=%s", flyteApiPort)},
		[]string{flyteApiPort + ":" + flyteApiPort})
	if err != nil {
		return nil, err
	}

	return &Flyte{flyteContainer}, nil
}

func getPort() string {
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		logger.Fatalf("Cannot start flyte. Error: %+v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	if err != nil {
		logger.Fatalf("Cannot start flyte. Error: %+v", err)
	}

	return strconv.Itoa(port)
}

func (f *Flyte) Stop() error {
	return f.flyteContainer.StopAndRemove()
}

func getFlyteImagePath() string {

	flyteImage := os.Getenv(flyteApiImageEnvName)

	if flyteImage == "" {
		logger.Infof("%v environment variable is not set, setting to default of %v", flyteApiImageEnvName, flyteApiDefaultImage)
		return flyteApiDefaultImage
	}

	logger.Infof("Using %v as value for %v", flyteImage, flyteApiImageEnvName)
	return flyteImage
}

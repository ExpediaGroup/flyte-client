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

package config

import (
	"github.com/HotelsDotCom/go-logger"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	apiTimeoutOutDefault   = time.Second * 10
	flyteApiEnvName        = "FLYTE_API"
	FlyteJWTEnvName        = "FLYTE_JWT"
	flyteLabelsEnvName     = "FLYTE_LABELS"
	flyteApiTimeOutEnvName = "FLYTE_API_TIMEOUT"
)

var GetEnv = os.Getenv

type Values struct {
	Labels      map[string]string
	FlyteApiUrl *url.URL
	Timeout     time.Duration
}

// returns the environment values
func FromEnvironment() Values {
	return Values{FlyteApiUrl: getFlyteApiUrl(), Labels: getLabels(), Timeout: getApiTimeOut()}
}

// checks that the flyteApi env FLYTE_API is set
func getFlyteApiUrl() *url.URL {
	apiEnvUrl := GetEnv(flyteApiEnvName)
	if apiEnvUrl == "" {
		logger.Fatalf("%s environment variable is not set", flyteApiEnvName)
	}

	flyteApiUrl, err := url.Parse(apiEnvUrl)
	if err != nil {
		logger.Fatalf("%s environment variable is not set to a valid URL: %v", flyteApiEnvName, err)
	}

	return flyteApiUrl
}

// checks that FLYTE_LABELS is set and it's value(s) are correct
func getLabels() (labels map[string]string) {
	labelsString := GetEnv(flyteLabelsEnvName)
	labels = make(map[string]string)

	if labelsString == "" {
		logger.Infof("%s environment variable is not set", flyteLabelsEnvName)
		return labels
	}

	// labels format: 'key=value,key=value'
	for _, label := range strings.Split(labelsString, ",") {
		items := strings.SplitN(label, "=", 2)
		if len(items) != 2 {
			logger.Fatalf("invalid format of %s environment variable: %v", flyteLabelsEnvName, labelsString)
		}
		labels[strings.TrimSpace(items[0])] = strings.TrimSpace(items[1])
	}
	return labels
}

// checks that the FLYTE_API_TIMEOUT is set, and if not sets to the default value.
func getApiTimeOut() time.Duration {

	apiTimeOut := GetEnv(flyteApiTimeOutEnvName)

	if apiTimeOut == "" {
		logger.Infof("FLYTE_API_TIMEOUT environment variable is not set, setting to default of %v", apiTimeoutOutDefault)
		return apiTimeoutOutDefault
	}

	apiTimeOutInt, err := strconv.Atoi(apiTimeOut)
	if err != nil {
		logger.Fatalf("%s is an invalid integer value: %v", flyteApiTimeOutEnvName, err)
	}

	if apiTimeOutInt < 0 {
		logger.Fatalf("%s has been set to an invalid value: %v", flyteApiTimeOutEnvName, apiTimeOutInt)
	}

	return time.Second * time.Duration(apiTimeOutInt)
}

func GetJWT() string {
	jwt := GetEnv(FlyteJWTEnvName)
	if jwt != "" {
		logger.Infof("%s environment variable is set.", FlyteJWTEnvName)
	}
	return jwt
}
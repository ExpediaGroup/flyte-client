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
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
	"time"
)

var envvars = map[string]string{}
var origGetEnv = GetEnv

func initTestEnv() {
	GetEnv = func(name string) string {
		return envvars[name]
	}
}

func restoreGetEnvFunc() {
	GetEnv = origGetEnv
}

func setEnv(name, value string) {
	envvars[name] = value
}

func clearEnv() {
	envvars = map[string]string{}
}

func TestShouldSetStandardSettingsFromEnvironment(t *testing.T) {
	defer restoreGetEnvFunc()
	defer clearEnv()
	initTestEnv()

	setEnv(flyteApiEnvName, "http://localhost:8080")
	setEnv(flyteApiTimeOutEnvName, "10")
	setEnv(flyteLabelsEnvName, "ABC=123,DEF=456")

	cfg := FromEnvironment()

	expectedURL, _ := url.Parse("http://localhost:8080")
	assert.Equal(t, expectedURL, cfg.FlyteApiUrl)

	assert.Equal(t, 10*time.Second, cfg.Timeout)

	expectedLabels := map[string]string{"ABC": "123", "DEF": "456"}
	assert.Equal(t, expectedLabels, cfg.Labels)
}

func TestShouldGetJWTFromEnvironment(t *testing.T) {
	defer restoreGetEnvFunc()
	defer clearEnv()
	initTestEnv()

	setEnv(FlyteJWTEnvName, "a.jwt.token")

	assert.Equal(t, "a.jwt.token", GetJWT())
}

func TestShouldNotGetJWTFromEnvironment(t *testing.T) {
	defer restoreGetEnvFunc()
	defer clearEnv()
	initTestEnv()

	// no jwt set in environment

	assert.Equal(t, "", GetJWT())
}

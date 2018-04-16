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
	"github.com/HotelsDotCom/go-logger/loggertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
	"time"
)

var envvars = map[string]string{}
var origGetEnv = getEnv

func initTestEnv() {
	getEnv = func(name string) string {
		return envvars[name]
	}
}

func restoreGetEnvFunc() {
	getEnv = origGetEnv
}

func setEnv(name, value string) {
	envvars[name] = value
}

func clearEnv() {
	envvars = map[string]string{}
}

func TestShouldSetStandardSettingsFromEnvironment(t *testing.T) {
	defer restoreGetEnvFunc()
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

func TestShouldLogFatalMessageWhenApiUrlIsNotSet(t *testing.T) {
	defer restoreGetEnvFunc()
	initTestEnv()

	// setup loggertest
	loggertest.Init(loggertest.LogLevelFatal)
	defer loggertest.Reset()

	// fatal log will now cause a panic so that we can verify the output
	defer func() {
		if r := recover(); r != nil {
			// assert logging behaviour - should have logged e.g. `[FATAL] 2017/10/03 14:26:48 foo.go:14: Oops "bad times"`
			logMessages := loggertest.GetLogMessages()
			require.Len(t, logMessages, 1)
			assert.Contains(t, logMessages[0].RawMessage, "FLYTE_API environment variable is not set")
		}
	}()

	clearEnv()

	getFlyteApiUrl()
}

func TestShouldLogFatalWhenApiUrlIsInvalid(t *testing.T) {
	defer restoreGetEnvFunc()
	initTestEnv()

	setEnv(flyteApiEnvName, "://locahost:8080")

	// setup loggertest
	loggertest.Init(loggertest.LogLevelFatal)
	defer loggertest.Reset()

	// fatal log will now cause a panic so that we can verify the output
	require.Panics(t, func() { getFlyteApiUrl() })

	// assert logging behaviour - should have logged e.g. `[FATAL] 2017/10/03 14:26:48 foo.go:14: Oops "bad times"`
	logMessages := loggertest.GetLogMessages()
	require.Len(t, logMessages, 1)
	assert.Contains(t, logMessages[0].RawMessage, "FLYTE_API environment variable is not set")

	clearEnv()

}

func TestShouldSetApiTimeOutToDefaultValueAndLogThatWhenEnvironmentVariableNotSet(t *testing.T) {
	defer restoreGetEnvFunc()
	initTestEnv()

	loggertest.Init(loggertest.LogLevelInfo)
	defer loggertest.Reset()

	expectedMessage := "FLYTE_API_TIMEOUT environment variable is not set, setting to default of 10s"

	getApiTimeOut()

	// assert logging behaviour - should have logged e.g. `[INFO] 2017/10/03 14:13:31 foo.go:6: You passed "hello"`
	logMessages := loggertest.GetLogMessages()
	require.Len(t, logMessages, 1)
	assert.Equal(t, expectedMessage, logMessages[0].Message)

	clearEnv()
}

func TestShouldFailWhenApiTimeOutIsInvalidValue(t *testing.T) {
	defer restoreGetEnvFunc()
	initTestEnv()

	setEnv(flyteApiTimeOutEnvName, "a")

	// setup loggertest
	loggertest.Init(loggertest.LogLevelFatal)
	defer loggertest.Reset()

	expectedMessage := "FLYTE_API_TIMEOUT is an invalid integer value: strconv.Atoi: parsing \"a\": invalid syntax"

	require.Panics(t, func() { getApiTimeOut() })

	logMessages := loggertest.GetLogMessages()
	require.Len(t, logMessages, 1)
	assert.Contains(t, logMessages[0].RawMessage, expectedMessage)

	clearEnv()
}

func TestShouldLogFatalWhenApiTimeOutValueIsSetToLessThanZero(t *testing.T) {
	defer restoreGetEnvFunc()
	initTestEnv()

	loggertest.Init(loggertest.LogLevelFatal)
	defer loggertest.Reset()

	setEnv(flyteApiTimeOutEnvName, "-1")

	expectedMessage := "FLYTE_API_TIMEOUT has been set to an invalid value: -1"

	require.Panics(t, func() { getApiTimeOut() })

	logMessages := loggertest.GetLogMessages()
	require.Len(t, logMessages, 1)
	assert.Contains(t, logMessages[0].RawMessage, expectedMessage)

	clearEnv()
}

func TestShouldLogFatalErrorWhenLabelsEnvironmentVariableIsSetToAnInvalidValue(t *testing.T) {
	defer restoreGetEnvFunc()
	initTestEnv()

	loggertest.Init(loggertest.LogLevelFatal)
	defer loggertest.Reset()

	setEnv(flyteLabelsEnvName, "ABC=123,DEF")

	expectedMessage := "invalid format of FLYTE_LABELS environment variable: ABC=123,DEF"

	require.Panics(t, func() { getLabels() })

	logMessages := loggertest.GetLogMessages()
	require.Len(t, logMessages, 1)
	assert.Contains(t, logMessages[0].RawMessage, expectedMessage)

	clearEnv()
}

func TestShouldLogInfoThatLabelsEnvironmentVariableIsNotSet(t *testing.T) {
	defer restoreGetEnvFunc()
	initTestEnv()

	loggertest.Init(loggertest.LogLevelInfo)
	defer loggertest.Reset()

	setEnv(flyteLabelsEnvName, "")
	expectedMessage := "FLYTE_LABELS environment variable is not set"
	getLabels()

	logMessages := loggertest.GetLogMessages()
	require.Len(t, logMessages, 1)
	assert.Equal(t, expectedMessage, logMessages[0].Message)

	clearEnv()
}

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

package healthcheck

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheck_shouldReturn200AndValidJsonResponse_whenAllHealthChecksAreSuccessful(t *testing.T) {
	// given these healthchecks
	endPointCheck := func() (name string, health Health) {
		return "EndPointCheck", Health{Healthy: true, Status: "All good"}
	}
	otherCheck := func() (name string, health Health) {
		return "OtherCheck", Health{Healthy: true, Status: "Ok"}
	}
	healthChecks := []HealthCheck{endPointCheck, otherCheck}

	request := httptest.NewRequest("GET", "/", nil)
	responseWriter := httptest.NewRecorder()

	// when the healthcheck on the pack is called
	handler(healthChecks)(responseWriter, request)

	// then
	assert.Equal(t, http.StatusOK, responseWriter.Code)
	assert.Equal(t, "application/json; charset=utf-8", responseWriter.Header().Get("Content-Type"))
	assert.Equal(t, `{"EndPointCheck":{"healthy":true,"status":"All good"},"OtherCheck":{"healthy":true,"status":"Ok"}}`, responseWriter.Body.String())
}

func TestHealthCheck_shouldReturn200AndLogMessage_whenNoHealthChecksAreRegistered(t *testing.T) {
	// given there are no healthchecks registered
	healthChecks := []HealthCheck{}

	request := httptest.NewRequest("GET", "/", nil)
	responseWriter := httptest.NewRecorder()

	// when the healthcheck on the pack is called
	handler(healthChecks)(responseWriter, request)

	// then
	assert.Equal(t, http.StatusOK, responseWriter.Code)
}

func TestHealthCheck_shouldReturn500AndValidJsonResponse_whenAHealthCheckFails(t *testing.T) {
	// given these healthchecks - with one failing
	endPointCheck := func() (name string, health Health) {
		return "EndPointCheck", Health{Healthy: true, Status: "All good"}
	}
	otherCheck := func() (name string, health Health) {
		return "OtherCheck", Health{Healthy: false, Status: "Oh No!!"}
	}
	healthChecks := []HealthCheck{endPointCheck, otherCheck}

	request := httptest.NewRequest("GET", "/", nil)
	responseWriter := httptest.NewRecorder()

	// when the healthcheck on the pack is called
	handler(healthChecks)(responseWriter, request)

	// then
	assert.Equal(t, http.StatusInternalServerError, responseWriter.Code)
	assert.Equal(t, "application/json; charset=utf-8", responseWriter.Header().Get("Content-Type"))
	assert.Equal(t, `{"EndPointCheck":{"healthy":true,"status":"All good"},"OtherCheck":{"healthy":false,"status":"Oh No!!"}}`, responseWriter.Body.String())
}

func TestHealthCheck_shouldReturn500HeaderResponse_whenJsonMarshallingError(t *testing.T) {
	// given these healthchecks - that will return invalid JSON
	endPointCheck := func() (name string, health Health) {
		return "EndPointCheck", Health{Healthy: true, Status: func() {}}
	}
	healthChecks := []HealthCheck{endPointCheck}

	request := httptest.NewRequest("GET", "/", nil)
	responseWriter := httptest.NewRecorder()

	// when the healthcheck on the pack is called
	handler(healthChecks)(responseWriter, request)

	// then
	assert.Equal(t, http.StatusInternalServerError, responseWriter.Code)
	assert.Equal(t, "application/json; charset=utf-8", responseWriter.Header().Get("Content-Type"))
}

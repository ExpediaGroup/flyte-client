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
	"encoding/json"
	"fmt"
	"net/http"
	"github.com/HotelsDotCom/go-logger"
	"time"
)

const Port = "8090"

// Each healthcheck should populate and return this struct with the result of the healthcheck.
type Health struct {
	Healthy bool        `json:"healthy"`
	Status  interface{} `json:"status"`
}

// This is the function you implement for your healthcheck/s.
type HealthCheck func() (name string, health Health)

// Start will take the health checks you provide and start a web server to handle them.
func Start(healthChecks []HealthCheck) *http.Server {
	srv := &http.Server{Addr: fmt.Sprintf(":%s", Port)}
	logger.Infof("starting healthcheck server on port %s", Port)
	http.HandleFunc("/", handler(healthChecks))
	go func(s *http.Server) {
		if err := s.ListenAndServe(); err != nil {
			logger.Errorf("Healthcheck: ListenAndServe: %v", err)
		}
	}(srv)
	time.Sleep(3 * time.Millisecond)
	return srv
}

// The handler will run the healthchecks passed in and output the results in JSON format, and will also write a 200
// http header response code if all checks are successful. A 500 is returned if any checks fail or on error.
// On error, no JSON will be returned but the error will be logged.
// If no healthchecks are registered, a successful header response will be returned.
func handler(healthChecks []HealthCheck) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		if len(healthChecks) == 0 {
			logger.Info("no healthchecks registered")
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		healthCheckResults := make(map[string]Health)
		for _, healthCheck := range healthChecks {
			name, health := healthCheck()
			healthCheckResults[name] = health
		}

		jsonResponse, err := json.Marshal(healthCheckResults)
		if err != nil {
			logger.Errorf("json marshalling error. healthCheckResults: %+v. error: %s", healthCheckResults, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		for _, hcResult := range healthCheckResults {
			if !hcResult.Healthy {
				w.WriteHeader(http.StatusInternalServerError)
				break
			}
		}
		w.Write(jsonResponse)
	}
}

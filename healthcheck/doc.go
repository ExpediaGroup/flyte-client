// Copyright (C) 2018 Expedia Group.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package healthcheck starts up a web server to handle pack healthchecks. On hitting the endpoint,
the healthchecks are run and the response body will be populated with the results in JSON format with either a 200
http header response code to indicate all checks have passed, or a 500 response code if one or more have failed.
On error such as a JSON marshalling error, a 500 response code will be returned and the error will be logged.
If no healthchecks are passed in, the healthcheck server will always return a healthy response.


Example

  func main() {

    // some code to set up your pack
    ...

    // example healthchecks
    dbCheck := func() (name string, health healthcheck.Health) {
		return "db", pingDb()
	}
	jiraCheck := func() (name string, health healthcheck.Health) {
		return "jira", pingJiraServer()
	}

	c := client.NewClient(createURL("http://example.com"), 10 * time.Second)
	p := flyte.NewPack(packDef, c, dbCheck, jiraCheck) // healthchecks are optional

    // now you can start your pack - this will also start your health check webserver
	p.Start()
  }

  // dummy healthcheck implementation that returns a hardcoded (healthy) result
  func pingDb() healthcheck.Health {
    return healthcheck.Health{Healthy:true, Status: "Good"}
  }

  // dummy healthcheck implementation that returns a hardcoded (healthy) result
  func pingJiraServer() healthcheck.Health {
    return healthcheck.Health{Healthy:true, Status: "Ok"}
  }

JSON Output

  {
    "db": {
      "healthy": true,
      "status": "Good"
    },
    "jira": {
      "healthy": true,
      "status": "Ok"
    }
  }

In this example because all checks are healthy, a 200 http header response code will also be returned.

*/
package healthcheck

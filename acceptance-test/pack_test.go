// +build acceptance

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
	"encoding/json"
	"net/url"
	"github.com/HotelsDotCom/flyte-client/client"
	"github.com/HotelsDotCom/flyte-client/config"
	"github.com/HotelsDotCom/flyte-client/flyte"
	"github.com/HotelsDotCom/flyte-client/healthcheck"
	"sync"
	"testing"
	"time"
	"net/http"
	"io/ioutil"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

var PackFeatures = []Test{
	{"ShouldRegisterAndStartNewPack", ShouldRegisterAndStartNewPack},
	{"ShouldHandleEventsAndExecutionOfCommands", ShouldHandleEventsAndExecutionOfCommands},
}

func ShouldRegisterAndStartNewPack(t *testing.T) {

	cfg := config.FromEnvironment()

	issueCreatedEventDef := flyte.EventDef{
		Name:    "IssueCreated",
		HelpURL: createURL("http://jirapack/help#issue-created", t),
	}

	issueCreationFailedEventDef := flyte.EventDef{
		Name:    "IssueCreationFailed",
		HelpURL: createURL("http://jirapack/help#issue-creation-failed", t),
	}

	issueClosedEventDef := flyte.EventDef{
		Name:    "IssueClosed",
		HelpURL: createURL("http://jirapack/help#issue-closed", t),
	}

	issueEditedEventDef := flyte.EventDef{
		Name: "IssueEdited",
	}

	createIssueCommand := flyte.Command{
		Name:         "createIssue",
		OutputEvents: []flyte.EventDef{issueCreatedEventDef, issueCreationFailedEventDef},
		HelpURL:      createURL("http://jirapack/help#create-issue-command", t),
		Handler:      func(message json.RawMessage) flyte.Event { return flyte.Event{} },
	}

	closeIssueCommand := flyte.Command{
		Name:         "closeIssue",
		OutputEvents: []flyte.EventDef{issueClosedEventDef},
		Handler:      func(message json.RawMessage) flyte.Event { return flyte.Event{} },
	}

	helpURL, _ := url.Parse("http://jirapack/help")
	packDef := flyte.PackDef{
		Name:      "JiraPack1",
		HelpURL:   helpURL,
		EventDefs: []flyte.EventDef{issueEditedEventDef},
		Commands: []flyte.Command{
			createIssueCommand,
			closeIssueCommand,
		},
	}

	c := client.NewClient(cfg.FlyteApiUrl, 10*time.Second)
	p := flyte.NewPack(packDef, c)
	p.Start()

	// now request the pack details from flyte-api
	resp, err := http.Get(flyteApiUrl + "/v1/packs/JiraPack1")
	defer resp.Body.Close()

	// unmarshall
	b, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	var pack = client.Pack{}
	err = json.Unmarshal(b, &pack)
	require.NoError(t, err)

	// check the details are as we expect
	assert.Equal(t, "JiraPack1", pack.Name)
	assert.Equal(t, "createIssue", pack.Commands[0].Name)
	assert.Equal(t, "closeIssue", pack.Commands[1].Name)
	assert.Equal(t, "IssueCreated", pack.Commands[0].EventNames[0])
	assert.Equal(t, "IssueCreationFailed", pack.Commands[0].EventNames[1])
	assert.Equal(t, "IssueClosed", pack.Commands[1].EventNames[0])

	// links
	assert.Equal(t, "http://jirapack/help", pack.Links[0].Href.String())
	assert.Equal(t, "help", pack.Links[0].Rel)
	assert.Equal(t, flyteApiUrl+"/v1/packs/JiraPack1", pack.Links[1].Href.String())
	assert.Equal(t, "self", pack.Links[1].Rel)
	assert.Equal(t, flyteApiUrl+"/v1/packs", pack.Links[2].Href.String())
	assert.Equal(t, "up", pack.Links[2].Rel)
	assert.Equal(t, flyteApiUrl+"/v1/packs/JiraPack1/actions/take", pack.Links[3].Href.String())
	assert.Equal(t, flyteApiUrl+"/swagger#!/action/takeAction", pack.Links[3].Rel)
	assert.Equal(t, flyteApiUrl+"/v1/packs/JiraPack1/events", pack.Links[4].Href.String())
	assert.Equal(t, flyteApiUrl+"/swagger#/event", pack.Links[4].Rel)

	// health check
	assertDefaultPackHealthCheckHasBeenAddedAndIsWorkingAsExpected(t, c)
}

func assertDefaultPackHealthCheckHasBeenAddedAndIsWorkingAsExpected(t *testing.T, c client.Client) {
	// when we call the pack health check
	resp, err := http.Get("http://localhost:" + healthcheck.Port)
	require.NoError(t, err)

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	// then we expect the following response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	assert.Equal(t, `{"DefaultCheck":{"healthy":true,"status":"Pack is running."}}`, string(body))
}

func ShouldHandleEventsAndExecutionOfCommands(t *testing.T) {

	cfg := config.FromEnvironment()

	// create flow - triggered on IssueCreated event and invokes closeIssue command
	r := createFlowDefStruct()
	postFlow(r, cfg.FlyteApiUrl, t)

	issueCreatedEventDef := flyte.EventDef{
		Name:    "IssueCreated",
		HelpURL: createURL("http://jirapack/help#issue-created", t),
	}

	issueClosedEventDef := flyte.EventDef{
		Name:    "IssueClosed",
		HelpURL: createURL("http://jirapack/help#issue-closed", t),
	}

	// the flow means that for the 5 events we fire below we should get this handler invoked 5 times
	expectedNoOfActions := 5
	var wg sync.WaitGroup
	wg.Add(expectedNoOfActions)

	closeIssueHandler := func(input json.RawMessage) flyte.Event {
		// countdown the wait group by 1
		defer wg.Done()

		var i DeleteIssueInput
		if err := json.Unmarshal(input, &i); err != nil {
			t.Error(err)
		}

		if i.IssueId != "AUTO-8" {
			t.Errorf("Expected issueId 'AUTO-8', received: %q", i.IssueId)
		}

		if i.ForceDelete != "false" {
			t.Errorf("Expected ForceDelete 'false', received: %q", i.ForceDelete)
		}

		return flyte.Event{
			EventDef: issueClosedEventDef,
			Payload: IssueDeletedPayload{
				IssueId: i.IssueId,
			},
		}
	}

	closeIssueCommand := flyte.Command{
		Name:         "closeIssue",
		OutputEvents: []flyte.EventDef{issueClosedEventDef},
		HelpURL:      createURL("http://jirapack/help#create-issue-command", t),
		Handler:      closeIssueHandler,
	}

	helpURL, _ := url.Parse("http://jirapack/help")
	packDef := flyte.PackDef{
		Name:    "JiraPack",
		HelpURL: helpURL,
		Commands: []flyte.Command{
			closeIssueCommand,
		},
	}
	p := flyte.NewPack(packDef, client.NewClient(cfg.FlyteApiUrl, 10*time.Second))
	flyte.StartHealthCheckServer = false // we do this to stop multiple registrations of the health check server via the various tests
	p.Start()

	// fire the 5 events
	for i := 0; i < expectedNoOfActions; i++ {
		p.SendEvent(flyte.Event{
			EventDef: issueCreatedEventDef,
			Payload:  createIssueCreatedPayload("AUTO-8"),
		})
	}

	// wait for the 5 command invocations to occur
	wg.Wait()
}

func createIssueCreatedPayload(issueId string) IssueCreatedPayload {
	return IssueCreatedPayload{
		IssueId:     issueId,
		Success:     true,
		JiraProject: "FLYTE",
		Description: "blah blah blah blah...",
		States:      []string{"new"},
		LinkedIssues: []LinkedIssue{
			{IssueId: "SWAT-1234", LinkType: "related_to"},
			{IssueId: "AUTO-6", LinkType: "depends_on"},
			{IssueId: "AUTO-12", LinkType: "related_to"},
		},
	}
}

func createURL(rawURL string, t *testing.T) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("Could not parse url %q:%s", rawURL, err)
	}
	return u
}

func flyteApiHealthCheck(c client.Client) healthcheck.HealthCheck {
	return func() (name string, health healthcheck.Health) {
		return "FlyteApiCheck", healthcheck.FlyteApiHealthCheck(c)
	}
}
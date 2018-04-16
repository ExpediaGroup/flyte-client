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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"github.com/HotelsDotCom/flyte-client/client"
	"testing"
	"time"
)

// creates a flow that will delete any tickets that depend on "AUTO-6"
func createFlowDefStruct() FlowDef {
	return FlowDef{
		Name:        "ticket-deleter",
		Description: "blaah blah",
		Steps: []Step{
			{
				Id:          "delete_ticket",
				Event:       Event{PackName: "JiraPack", Name: "IssueCreated"},
				CriteriaDef: createStepCriteria(),
				CommandDef:  createStepCommand(),
			},
		},
	}
}

func createStepCriteria() string {
	return `{{ Event.Payload.linkedIssues.1.issueId == 'AUTO-6' }}`
}

func createStepCommand() CommandDef {
	input := DeleteIssueInput{
		IssueId:     "{{ Event.Payload.issueId }}",
		ForceDelete: "false",
	}
	return CommandDef{
		PackName: "JiraPack",
		Name:     "closeIssue",
		Input:    input}
}

type FlowDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Steps       []Step `json:"steps"`
}

type Step struct {
	Id          string            `json:"id,omitempty"`
	DependsOn   []string          `json:"dependsOn,omitempty"`
	Event       Event             `json:"event"`
	ContextDef  map[string]string `json:"context,omitempty"`
	CriteriaDef string            `json:"criteria,omitempty"`
	CommandDef  CommandDef        `json:"command" bson:"command"`
}

type Event struct {
	PackName   string            `json:"packName"`
	PackLabels map[string]string `json:"packLabels"`
	Name       string            `json:"name"`
}

type CommandDef struct {
	PackName   string            `json:"packName"`
	PackLabels map[string]string `json:"packLabels"`
	Name       string            `json:"name"`
	Input      interface{}       `json:"input"`
}

type Matcher struct {
	Arg1 string `json:"arg1"`
	Type string `json:"type"`
	Arg2 string `json:"arg2"`
}

type DeleteIssueInput struct {
	IssueId     string `json:"issueId"`
	ForceDelete string `json:"forceDelete"`
}

type IssueDeletedPayload struct {
	IssueId     string `json:"issueId"`
	Success     bool   `json:"status"`
	JiraProject string `json:"jiraProject"`
}

func postFlow(flow FlowDef, apiURL *url.URL, t *testing.T) {
	baseURL := getBaseURL(*apiURL)

	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}

	// delete the flow if it exists already (cannot overwrite an existing flow)
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/flows/%s", baseURL.String(), flow.Name), nil)
	delResp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to delete flow: %s", err)
	}
	defer delResp.Body.Close()

	jsonBytes, err := json.Marshal(flow)
	if err != nil {
		t.Fatalf("Failed to marshal flow %s", err)
	}

	url := fmt.Sprintf("%s/flows", baseURL)
	req, _ = http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")

	createResp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to create flow %s", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create flow (at %q) , resp: %+v", url, createResp)
	}
}

func getBaseURL(u url.URL) *url.URL {
	u.Path = path.Join(u.Path, client.ApiVersion)
	return &u
}

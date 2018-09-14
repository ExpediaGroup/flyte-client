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
	"github.com/HotelsDotCom/flyte-client/client"
	"github.com/HotelsDotCom/flyte-client/config"
	"net/url"
	"testing"
	"time"
)

var ClientFeatures = []Test{
	{"ShouldCreatePack", ShouldCreatePack},
	{"ShouldPostEventToPack", ShouldPostEventToPack},
	{"ShouldTakeAndCompleteActions", ShouldTakeAndCompleteActions},
	{"TakeActionShouldReturnNilIfNoActionAvailable", TakeActionShouldReturnNilIfNoActionAvailable},
}

func ShouldCreatePack(t *testing.T) {
	var cfg = config.FromEnvironment()

	p := createPackStruct()
	c := client.NewClient(cfg.FlyteApiUrl, 10*time.Second,false)

	if err := c.CreatePack(p); err != nil {
		t.Fatalf("Failed to create pack: %s", err)
	}
}

func createPackStruct() client.Pack {
	u, _ := url.Parse("http://something/somewhere")
	link := client.Link{
		Href: u,
		Rel:  "something",
	}

	issueClosedEventDef := client.EventDef{
		Name:  "IssueClosed",
		Links: []client.Link{link},
	}

	issueCreatedEventDef := client.EventDef{
		Name: "IssueCreated",
	}

	createIssueCommand := client.Command{
		Name:       "createIssue",
		EventNames: []string{issueCreatedEventDef.Name},
	}

	closeIssueCommand := client.Command{
		Name:       "closeIssue",
		EventNames: []string{issueClosedEventDef.Name},
		Links:      []client.Link{link},
	}

	return client.Pack{
		Name:      "JiraPack",
		EventDefs: []client.EventDef{issueCreatedEventDef, issueClosedEventDef},
		Commands:  []client.Command{createIssueCommand, closeIssueCommand},
		Links:     []client.Link{link},
	}
}

func ShouldPostEventToPack(t *testing.T) {
	var cfg = config.FromEnvironment()

	p := createPackStruct()

	c := client.NewClient(cfg.FlyteApiUrl, 10*time.Second,false)
	if err := c.CreatePack(p); err != nil {
		t.Fatalf("Failed to create pack: %s", err)
	}
	e := createIssueCreatedEventStruct("AUTO-8")

	if err := c.PostEvent(e); err != nil {
		t.Fatalf("Failed to post event: %s", err)
	}
}

func createIssueCreatedEventStruct(issueId string) client.Event {
	payload := IssueCreatedPayload{
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

	return client.Event{
		Name:    "IssueCreated",
		Payload: payload,
	}
}

type IssueCreatedPayload struct {
	IssueId      string        `json:"issueId"`
	Success      bool          `json:"status"`
	JiraProject  string        `json:"jiraProject"`
	Description  string        `json:"description"`
	States       []string      `json:"states"`
	LinkedIssues []LinkedIssue `json:"linkedIssues"`
}

type LinkedIssue struct {
	IssueId  string `json:"issueId"`
	LinkType string `json:"type"`
}

func ShouldTakeAndCompleteActions(t *testing.T) {
	// create pack
	var cfg = config.FromEnvironment()

	p := createPackStruct()
	c := client.NewClient(cfg.FlyteApiUrl, 10*time.Second,false)
	if err := c.CreatePack(p); err != nil {
		t.Fatalf("Failed to create pack: %s", err)
	}

	// create flow
	r := createFlowDefStruct()

	postFlow(r, cfg.FlyteApiUrl, t)

	// send event that should match flow step
	e := createIssueCreatedEventStruct("AUTO-8")
	if err := c.PostEvent(e); err != nil {
		t.Fatalf("Failed to post event: %s", err)
	}

	// wait for event to be matched by api
	time.Sleep(5 * time.Second)

	// should be an action available
	a, err := c.TakeAction()
	if a == nil || err != nil {
		t.Fatalf("Failed to take action: %s", err)
	}

	if a.CommandName != "closeIssue" {
		t.Fatalf("Expecting closeIssue action but received: %+v", a)
	}

	if err := c.CompleteAction(*a, createIssueDeletedEventStruct("AUTO-8")); err != nil {
		t.Fatalf("flyte api did not successfully mark action as complete: %s", err)
	}
}

func createIssueDeletedEventStruct(issueId string) client.Event {
	payload := IssueDeletedPayload{
		IssueId:     issueId,
		Success:     true,
		JiraProject: "FLYTE",
	}

	return client.Event{
		Name:    "IssueDeleted",
		Payload: payload,
	}
}

func TakeActionShouldReturnNilIfNoActionAvailable(t *testing.T) {
	var cfg = config.FromEnvironment()

	p := createPackStruct()
	c := client.NewClient(cfg.FlyteApiUrl, 10*time.Second,false)
	if err := c.CreatePack(p); err != nil {
		t.Fatalf("Failed to create pack: %s", err)
	}

	if a, err := c.TakeAction(); a != nil && err != nil {
		t.Fatalf("No actions should be available so action (%v) and err (%v) should both be nil", a, err)
	}
}

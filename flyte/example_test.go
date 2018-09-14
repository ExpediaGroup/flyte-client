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

package flyte_test

import (
	"encoding/json"
	"fmt"
	"github.com/HotelsDotCom/flyte-client/client"
	"github.com/HotelsDotCom/flyte-client/flyte"
	"net/url"
	"time"
)

func ExampleNewPack() {
	// First we create event definitions that describe what events our pack can raise.
	// An EventDef contains the name of the event (mandatory) and a help URL (optional)
	issueCreatedEventDef := flyte.EventDef{
		Name: "IssueCreated",
	}
	issueCreationErrorEventDef := flyte.EventDef{
		Name: "IssueCreationError",
	}

	// This pack has a single "createIssue" command. To implement a command we must provide a "CommandHandler" function matching the signature below.
	// The client will call this handler every time it receives a "createIssue" action from the Flyte api. The handler will take the input JSON from the action
	// and must return a flyte.Event. Handlers are where the functionality of the pack is implemented so will likely form the bulk of most packs.
	createIssueHandler := func(input json.RawMessage) flyte.Event {

		// deserialize the raw JSON into our domain struct
		var createIssueInput CreateIssueInput
		json.Unmarshal(input, &createIssueInput)

		// call some ticket creation code...
		// ...

		// if it succeeds then return something like the following. The payload will be serialised to JSON and sent to the Flyte api server.
		return flyte.Event{
			EventDef: issueCreatedEventDef,
			Payload: IssueCreatedPayload{
				Project:  "FOO",
				IssueId:  "123",
				Location: createURL("http://jira/FOO/123"),
			},
		}
	}

	// Next we create a struct that defines the "createIssue" command. Note the handler above is passed to it.
	// Also note that we specify what events the command can output.
	// The help URL is optional
	createIssueCommand := flyte.Command{
		Name: "createIssue",
		OutputEvents: []flyte.EventDef{
			issueCreatedEventDef,
			issueCreationErrorEventDef,
		},
		HelpURL: createURL("http://jirapack/help#create-issue-command"),
		Handler: createIssueHandler,
	}

	// The final struct we must define is the PackDef struct which pulls together the above structs to give the full definition of the pack.
	packDef := flyte.PackDef{
		Name:     "JiraPack",
		Commands: []flyte.Command{createIssueCommand},
		HelpURL:  createURL("http://jirapack/help#create-issue-command"),
	}

	// Finally we call NewPack() to create a pack struct. This can then be started by calling Start()
	p := flyte.NewPack(packDef, client.NewClient(createURL("http://example.com"), 10*time.Second, false))
	// p.Start() is not blocking, it is user's responsibility to make sure that the program does not exit immediately
	p.Start()
}

type CreateIssueInput struct {
	Project    string `json:"project"`
	TicketText string `json:"ticketText"`
}

type IssueCreatedPayload struct {
	Project  string   `json:"project"`
	IssueId  string   `json:"issueId"`
	Location *url.URL `json:"location"`
}

type IssueCreationErrorPayload struct {
	Project   string `json:"project"`
	ErrorText string `json:"error"`
}

func createURL(u string) *url.URL {
	url, _ := url.Parse(u)
	return url
}

func ExampleNewFatalEvent() {
	payload := fmt.Errorf("some error").Error()
	fmt.Printf("%v", flyte.NewFatalEvent(payload))
}

func ExampleCommand() {
	// to create a command, we need to create and populate at least one output event. often, these will represent
	// a success or failure case
	issueCreatedEventDef := flyte.EventDef{
		Name: "IssueCreated",
	}
	issueCreationErrorEventDef := flyte.EventDef{
		Name: "IssueCreationError",
	}

	// we also need a command handler. this is where the functionality of the command will be implemented
	createIssueHandler := func(input json.RawMessage) flyte.Event {

		// deserialize the raw JSON into our domain struct
		var createIssueInput CreateIssueInput
		json.Unmarshal(input, &createIssueInput)

		// call some ticket creation code...
		// ...

		// if it succeeds then return something like the following. The payload will be serialised to JSON and sent to the Flyte api server.
		return flyte.Event{
			EventDef: issueCreatedEventDef,
			Payload: IssueCreatedPayload{
				Project:  "FOO",
				IssueId:  "123",
				Location: createURL("http://jira/FOO/123"),
			},
		}
	}

	// you can also provide an optional help URL. this will help document your command to your end users
	helpUrl, _ := url.Parse("http://jirapack/help#create-issue-command")

	// you can now give the command a name. it is now ready to be passed into the pack definition
	createIssueCommand := flyte.Command{
		Name: "createIssue",
		OutputEvents: []flyte.EventDef{
			issueCreatedEventDef,
			issueCreationErrorEventDef,
		},
		Handler: createIssueHandler,
		HelpURL: helpUrl,
	}
	fmt.Printf("%+v", createIssueCommand)
}

func ExampleCommandHandler() {
	// In this example the client will call this handler every time it receives a "createIssue" action from the Flyte api. The handler will take the input JSON from the action
	// and must return a flyte.Event. Handlers are where the functionality of the pack is implemented so will likely form the bulk of most packs.

	// the JSON input passed into the handler will be deserialized into this struct
	type CreateIssueInput struct {
		Project    string `json:"project"`
		TicketText string `json:"ticketText"`
	}

	// this event definition will be passed into the returned flyte.Event below
	issueCreatedEventDef := flyte.EventDef{
		Name: "IssueCreated",
	}

	// this payload struct will used in the returned flyte.Event below
	type IssueCreatedPayload struct {
		Project string `json:"project"`
		IssueId string `json:"issueId"`
	}

	// the handler once created, can now be used in a flyte.Command struct
	createIssueHandler := func(input json.RawMessage) flyte.Event {

		// deserialize the raw JSON into our domain struct
		var createIssueInput CreateIssueInput
		json.Unmarshal(input, &createIssueInput)

		// call some ticket creation code...
		// ...

		// if it succeeds then return something like the following. The payload will be serialised to JSON and sent to the Flyte api server.
		return flyte.Event{
			EventDef: issueCreatedEventDef,
			Payload: IssueCreatedPayload{
				Project: "FOO",
				IssueId: "123",
			},
		}
	}
	fmt.Printf("%+v", createIssueHandler(json.RawMessage{}))
}

func ExampleEvent() {
	// to create an event, you first need to create a named event definition
	// to help document your event to your end users, you can provide an optional help URL
	helpUrl, _ := url.Parse("http://jirapack/help#create-issue-command")
	issueCreatedEventDef := flyte.EventDef{
		Name:    "IssueCreated",
		HelpURL: helpUrl, // optional
	}

	// you will also need to provide a payload containing the relevant data for the event.
	// the payload will be marshalled into JSON, so should be annotated appropriately
	type IssueCreatedPayload struct {
		Project string `json:"project"`
		IssueId string `json:"issueId"`
	}

	// then simply pass in the event definition and a populated payload struct to create the event
	event := flyte.Event{
		EventDef: issueCreatedEventDef,
		Payload: IssueCreatedPayload{
			Project: "FOO",
			IssueId: "123",
		},
	}
	fmt.Printf("%+v", event)
}

func ExampleEventDef() {
	// an event definition defines an event. to help the end user, you can provide an optional help URL
	helpUrl, _ := url.Parse("http://jirapack/help#issue-created-event-def")

	// once created, the event definition is ready to be passed into a flyte.Event struct
	issueCreatedEventDef := flyte.EventDef{
		Name:    "IssueCreated",
		HelpURL: helpUrl,
	}
	fmt.Printf("%+v", issueCreatedEventDef)
}

func ExamplePackDef() {
	// First we create event definitions that describe what events our pack can raise.
	// An event definition contains the name of the event (mandatory) and a help URL (optional)
	issueCreatedEventDef := flyte.EventDef{
		Name: "IssueCreated",
	}
	issueCreationErrorEventDef := flyte.EventDef{
		Name: "IssueCreationError",
	}

	// This pack has a single "createIssue" command. To implement a command we must provide a "CommandHandler" function matching the signature below.
	// The client will call this handler every time it receives a "createIssue" action from the Flyte api. The handler will take the input JSON from the action
	// and must return a flyte.Event. Handlers are where the functionality of the pack is implemented so will likely form the bulk of most packs.
	createIssueHandler := func(input json.RawMessage) flyte.Event {

		// deserialize the raw JSON into our domain struct
		var createIssueInput CreateIssueInput
		json.Unmarshal(input, &createIssueInput)

		// call some ticket creation code...
		// ...

		// if it succeeds then return something like the following. The payload will be serialised to JSON and sent to the Flyte api server.
		return flyte.Event{
			EventDef: issueCreatedEventDef,
			Payload: IssueCreatedPayload{
				Project: "FOO",
				IssueId: "123",
			},
		}
	}

	// Next we create a struct that defines the "createIssue" command. Note the handler above is passed to it.
	// Also note that we specify what events the command can output.
	createIssueCommand := flyte.Command{
		Name: "createIssue",
		OutputEvents: []flyte.EventDef{
			issueCreatedEventDef,
			issueCreationErrorEventDef,
		},
		Handler: createIssueHandler,
	}

	// you can add (optional) labels that act as a filter that determines when the pack will execute against a flow
	// in this example the pack will only run in a 'production' environment
	labels := make(map[string]string)
	labels["env"] = "prod"

	// you can provide a help URL that describes what the pack does and is used for
	helpUrl, _ := url.Parse("http://jirapack/help#create-issue-command")

	// now we are ready to create the PackDef struct which pulls together the above structs to give the full definition of the pack.
	packDef := flyte.PackDef{
		Name:   "JiraPack",
		Labels: labels,
		EventDefs: []flyte.EventDef{
			issueCreatedEventDef,
			issueCreationErrorEventDef,
		},
		Commands: []flyte.Command{createIssueCommand},
		HelpURL:  helpUrl,
	}
	fmt.Printf("%+v", packDef)
}

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

package flyte

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/url"
	"github.com/HotelsDotCom/flyte-client/client"
	"github.com/HotelsDotCom/go-logger"
	"sync"
	"testing"
	"time"
)

func Test_NewPack_ShouldRetryRegistrationOnError(t *testing.T) {
	StartHealthCheckServer = false // we need this to stop multiple registrations of the healthcheck server

	issueCreatedEventDef := EventDef{
		Name:    "IssueCreated",
		HelpURL: createURL("http://jirapack/help#issue-created", t),
	}

	createIssueCommand := Command{
		Name:         "createIssue",
		OutputEvents: []EventDef{issueCreatedEventDef},
		HelpURL:      createURL("http://jirapack/help#create-issue-command", t),
		Handler: func(message json.RawMessage) Event {
			return Event{}
		},
	}

	packDef := PackDef{
		Name:     "JiraPack",
		HelpURL:  createURL("http://jirapack/help", t),
		Commands: []Command{createIssueCommand},
	}

	registerFailCount := 1
	c := MockClient{
		createPack: func(client.Pack) error {
			if registerFailCount > 0 {
				registerFailCount -= registerFailCount
				return errors.New("Failed to register pack with flyte service")
			}
			return nil
		},
		takeAction: func() (*client.Action, error) {
			return nil, nil
		},
	}

	logMsg := ""
	loggerFn := logger.Errorf
	logger.Errorf = func(msg string, args ...interface{}) { logMsg = fmt.Sprintf(msg, args...) }
	defer func() { logger.Errorf = loggerFn }()

	p := NewPack(packDef, c)
	p.Start()
	assert.Equal(t, "cannot register pack: Failed to register pack with flyte service", logMsg)
}

func Test_SendEvent(t *testing.T) {
	StartHealthCheckServer = false // we need this to stop multiple registrations of the healthcheck server

	buildSucessEventDef := EventDef{
		Name: "BuildSuccess",
	}

	packDef := PackDef{
		Name:      "BambooPack",
		EventDefs: []EventDef{buildSucessEventDef},
	}

	e := Event{
		EventDef: buildSucessEventDef,
		Payload:  "blah blah",
	}

	c := MockClient{
		createPack: func(p client.Pack) error {
			return nil
		},
		postEvent: func(event client.Event) error {
			if event.Name != e.EventDef.Name {
				t.Fatalf("Expected event with name %q, but received %q", e.EventDef.Name, event.Name)
			}
			if event.Payload != e.Payload {
				t.Fatalf("Expected event with payload %v, but received %v", e.Payload, event.Payload)
			}
			return nil
		},
	}

	p := NewPack(packDef, c)
	p.Start()

	if err := p.SendEvent(e); err != nil {
		assert.Fail(t, fmt.Sprintf("Unexpected error sending event: %s", err))
	}
}

func Test_ErrorSendingEvent(t *testing.T) {
	StartHealthCheckServer = false // we need this to stop multiple registrations of the healthcheck server

	buildSucessEventDef := EventDef{
		Name: "BuildSuccess",
	}

	packDef := PackDef{
		Name:      "BambooPack2",
		EventDefs: []EventDef{buildSucessEventDef},
	}

	e := Event{
		EventDef: buildSucessEventDef,
		Payload:  "blah blah",
	}

	c := MockClient{
		createPack: func(p client.Pack) error {
			return nil
		},
		postEvent: func(event client.Event) error {
			return fmt.Errorf("Failed to send Event %+v", event)
		},
	}

	p := NewPack(packDef, c)
	p.Start()

	if err := p.SendEvent(e); err == nil {
		assert.Fail(t, "Unexpected error sending event")
	}
}

func createURL(rawURL string, t *testing.T) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		assert.Fail(t, fmt.Sprintf("Could not parse url %q:%s", rawURL, err))
	}
	return u
}

func Test_EventAndCommandAggregationAndConversion(t *testing.T) {
	StartHealthCheckServer = false // we need this to stop multiple registrations of the healthcheck server

	messageReceivedEventDef := EventDef{
		Name: "MessageReceived",
	}

	messageSentEventDef := EventDef{
		Name: "MessageSent",
	}

	messageSendFailEventDef := EventDef{
		Name: "MessageSendFailure",
	}

	sendMessageCommand := Command{
		Name:         "sendMessage",
		OutputEvents: []EventDef{messageSentEventDef, messageSendFailEventDef},
		Handler: func(message json.RawMessage) Event {
			return Event{}
		},
	}

	packDef := PackDef{
		Name:      "HipchatPack",
		EventDefs: []EventDef{messageReceivedEventDef, messageSentEventDef},
		Commands:  []Command{sendMessageCommand},
	}
	// - "messageReceivedEventDef" passed just once as standalone eventdef
	// - "messageSendFailEventDef" passed just once as part of command
	// - "messageSentEventDef" passed twice (as standalone event def and as part of command)
	// - all should appear once and only once in the registered pack
	c := MockClient{
		createPack: func(p client.Pack) error {
			assert.Equal(t, packDef.Name, p.Name)
			assert.Equal(t, 1, len(p.Commands))
			assert.Equal(t, sendMessageCommand.Name, p.Commands[0].Name)
			assert.Equal(t, 3, len(p.EventDefs))

			eventNamesSet := make(map[string]bool)
			for _, e := range p.EventDefs {
				eventNamesSet[e.Name] = true
			}

			assert.True(t, eventNamesSet["MessageReceived"])
			assert.True(t, eventNamesSet["MessageSent"])
			assert.True(t, eventNamesSet["MessageSendFailure"])
			return nil
		},
		takeAction: func() (*client.Action, error) {
			return nil, nil
		},
	}

	p := NewPack(packDef, c)
	p.Start()
}

func Test_ShouldMoveOnToNextAction_IfErrorProcessingAction(t *testing.T) {
	StartHealthCheckServer = false // we need this to stop multiple registrations of the healthcheck server

	messageSentEventDef := EventDef{Name: "MessageSent"}
	eventToReturn := Event{
		EventDef: messageSentEventDef,
	}
	sendMessageCommand := Command{
		Name:         "sendMessage",
		OutputEvents: []EventDef{},
		Handler: func(message json.RawMessage) Event {
			return eventToReturn
		},
	}

	packDef := PackDef{
		Name:     "HipchatPack2",
		Commands: []Command{sendMessageCommand},
	}

	var wg sync.WaitGroup
	wg.Add(2) // should be 1 call from takeAction and 1 from completeAction
	takeActionCount := 0
	c := MockClient{
		createPack: func(p client.Pack) error {
			return nil
		},
		takeAction: func() (*client.Action, error) {
			takeActionCount++
			if takeActionCount == 2 {
				wg.Done()
				return &client.Action{CommandName: "sendMessage"}, nil
			}
			return nil, errors.New("error taking action")
		},
		completeAction: func(action client.Action, event client.Event) error {
			defer wg.Done()
			assert.Equal(t, eventToReturn.EventDef.Name, event.Name)
			assert.Equal(t, eventToReturn.Payload, event.Payload)
			return nil
		},
	}

	p := NewPack(packDef, c)
	p.Start()

	wg.Wait()
}

func Test_PackWithNoCommands_ShouldNotGetActionsFromFlyteServer(t *testing.T) {
	StartHealthCheckServer = false // we need this to stop multiple registrations of the healthcheck server

	buildSucessEventDef := EventDef{
		Name: "BuildSuccess",
	}

	packDef := PackDef{
		Name:      "BambooPack3",
		EventDefs: []EventDef{buildSucessEventDef},
	}

	c := MockClient{
		createPack: func(p client.Pack) error {
			return nil
		},
		takeAction: func() (*client.Action, error) {
			assert.Fail(t, "takeAction called unexepectedely")
			return nil, nil
		},
	}

	p := NewPack(packDef, c)
	p.Start()
	// allow time for pack to start communicating with flyte server
	time.Sleep(1 * time.Second)
}

func Test_PanickingCommandHandlerSendsFatalEvent(t *testing.T) {
	StartHealthCheckServer = false // we need this to stop multiple registrations of the healthcheck server

	actionGenerated := false
	completeChannel := make(chan bool)
	buildSucessEventDef := EventDef{Name: "BuildSuccess"}
	panicMessage := "ARGGHHHHH!"
	command := Command{
		Name: "RunBuild",
		Handler: func(input json.RawMessage) Event {
			panic(panicMessage)
		}}

	client := MockClient{
		createPack: func(p client.Pack) error {
			return nil
		},
		takeAction: func() (*client.Action, error) {
			if !actionGenerated {
				actionGenerated = true
				return &client.Action{CommandName: command.Name}, nil
			}
			return nil, nil
		},
		completeAction: func(action client.Action, e client.Event) error {
			assert.Equal(t, fatalEventName, e.Name)
			assert.Equal(t, panicMessage, e.Payload.(string))
			completeChannel <- true
			return nil
		},

	}

	p := NewPack(PackDef{Name: "BambooPack4", EventDefs: []EventDef{buildSucessEventDef}, Commands: []Command{command}}, client)
	p.Start()

	if err := waitForChannelOrTimeout(completeChannel, time.Second*1); err != nil {
		assert.Fail(t, fmt.Sprintf("Should call complete action with %q event", fatalEventName))
	}
}

func Test_PanickingCommandHandlerDoesnotKillThePack(t *testing.T) {
	StartHealthCheckServer = false // we need this to stop multiple registrations of the healthcheck server

	actionGenerated := false
	var eventSent *client.Event
	waitForPanic := make(chan bool)
	eventDef := EventDef{Name: "BuildSuccess"}
	command := Command{
		Name: "RunBuild",
		Handler: func(input json.RawMessage) Event {
			defer func() {
				waitForPanic <- true
			}()
			panic(errors.New("Whatever man!!!!"))
		},
	}

	client := MockClient{
		createPack: func(p client.Pack) error {
			return nil
		},
		postEvent: func(e client.Event) error {
			eventSent = &e
			return nil
		},
		takeAction: func() (*client.Action, error) {
			if !actionGenerated {
				actionGenerated = true
				return &client.Action{CommandName: command.Name}, nil
			}
			return nil, nil
		},
		completeAction: func(a client.Action, e client.Event) error {
			return nil
		},
	}

	p := NewPack(PackDef{Name: "BambooPack4", EventDefs: []EventDef{eventDef}, Commands: []Command{command}}, client)
	p.Start()

	if err := waitForChannelOrTimeout(waitForPanic, time.Second*1); err != nil {
		assert.Fail(t, "Should panic during command execution")
	}

	if err := p.SendEvent(Event{EventDef: eventDef}); err != nil {
		assert.Fail(t, "Unexpected error sending event. Pack should be still up even though the handler panics: %s")
	}
	if eventSent == nil {
		assert.Fail(t, "No event was sent as a result of the error")
	}
	if eventSent.Name != "BuildSuccess" {
		assert.Fail(t, "Captured wrong event")
	}
}

func Test_HandleActionShouldSendFatalEvent_WhenThereCommandHandlerIsNil(t *testing.T) {
	StartHealthCheckServer = false // we need this to stop multiple registrations of the healthcheck server

	actionGenerated := false
	completeChannel := make(chan bool)
	buildSucessEventDef := EventDef{Name: "BuildSuccess"}
	command := Command{
		Name:    "RunBuild",
		Handler: nil,
	}

	client := MockClient{
		createPack: func(p client.Pack) error {
			return nil
		},
		takeAction: func() (*client.Action, error) {
			if !actionGenerated {
				actionGenerated = true
				return &client.Action{CommandName: command.Name}, nil
			}
			return nil, nil
		},
		completeAction: func(action client.Action, e client.Event) error {
			assert.Equal(t, fatalEventName, e.Name)
			assert.NotNil(t, e.Payload)
			completeChannel <- true
			return nil
		},
	}

	p := NewPack(PackDef{Name: "BambooPack4", EventDefs: []EventDef{buildSucessEventDef}, Commands: []Command{command}}, client)
	p.Start()

	if err := waitForChannelOrTimeout(completeChannel, time.Second*1); err != nil {
		t.Fatalf("Should call complete action with %q event", fatalEventName)
	}
}

func Test_HandleAction_ShouldSendFatalEvent_WhenThereIsNoCommandHandler(t *testing.T) {
	StartHealthCheckServer = false // we need this to stop multiple registrations of the healthcheck server

	actionGenerated := false
	completeChannel := make(chan bool)
	command := Command{
		Name: "ExistingCommand",
	}

	client := MockClient{
		createPack: func(p client.Pack) error {
			return nil
		},
		takeAction: func() (*client.Action, error) {
			if !actionGenerated {
				actionGenerated = true
				return &client.Action{CommandName: "PhantomCommand"}, nil
			}
			return nil, nil
		},
		completeAction: func(action client.Action, e client.Event) error {
			completeChannel <- true
			assert.Equal(t, fatalEventName, e.Name)
			return nil
		},
	}

	p := NewPack(PackDef{Name: "BambooPack4", Commands: []Command{command}}, client)
	p.Start()

	if err := waitForChannelOrTimeout(completeChannel, time.Second*1); err != nil {
		assert.Fail(t, fmt.Sprintf("Should call complete action with %q event", fatalEventName))
	}
}

type createPack func(client.Pack) error
type postEvent func(client.Event) error
type takeAction func() (*client.Action, error)
type completeAction func(action client.Action, event client.Event) error

type MockClient struct {
	createPack     createPack
	postEvent      postEvent
	takeAction     takeAction
	completeAction completeAction
}

func (c MockClient) CreatePack(pack client.Pack) error {
	return c.createPack(pack)
}

func (c MockClient) PostEvent(event client.Event) error {
	return c.postEvent(event)
}

func (c MockClient) TakeAction() (*client.Action, error) {
	return c.takeAction()
}

func (c MockClient) CompleteAction(action client.Action, event client.Event) error {
	return c.completeAction(action, event)
}

func (c MockClient) GetFlyteHealthCheckURL() (*url.URL, error) {
	return nil, nil
}

func waitForChannelOrTimeout(c chan bool, duration time.Duration) error {
	select {
	case <-c:
		return nil
	case <-time.After(duration):
		return errors.New("Timed out waiting for channel")
	}
}

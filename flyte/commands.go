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
	"fmt"
	"github.com/HotelsDotCom/flyte-client/client"
	"github.com/HotelsDotCom/go-logger"
	"time"
)

func (p pack) handleCommands() {
	if len(p.Commands) > 0 {
		go p.handleCommandActions()
	}
}

// repeatedly takes the next incoming action from the flyte server, passes to the appropriate handler and
// sends the output event to the flyte server
func (p pack) handleCommandActions() {
	handlers := p.createHandlersMap()
	for {
		a := p.getNextAction()
		// concurrently handle the incoming actions
		go p.handleAction(a, handlers)
	}
}

// creates map of commandName -> handler, so incoming actions can be routed easily
func (p pack) createHandlersMap() map[string]CommandHandler {
	handlers := make(map[string]CommandHandler)
	for _, c := range p.Commands {
		handlers[c.Name] = c.Handler
	}
	return handlers
}

// gets the next action to process from the flyte server, if no action immediately available will start polling
func (p pack) getNextAction() *client.Action {
	for {
		a, err := p.client.TakeAction()
		if err != nil {
			if _, ok := err.(client.NotFoundError); ok {
				logger.Fatal("Pack not found while polling for actions. Exiting.")
			}
			logger.Infof("could not take action: %s", err)
		}
		if a == nil || err != nil {
			time.Sleep(p.pollingFrequency)
			continue
		}
		return a
	}
}

// invokes the relevant handler using the action input JSON and completes the action by posting the result to the flyte api
// if no handler found, then the action will be completed using a fatal event
func (p pack) handleAction(a *client.Action, handlers map[string]CommandHandler) {
	// ensure that a panicking CommandHandler is captured and handled
	defer p.handlePanic(a)

	handler, ok := handlers[a.CommandName]
	if !ok {
		err := fmt.Errorf("no handler could be found for command %q in %v", a.CommandName, handlers)
		p.completeAction(a, NewFatalEvent(err.Error()))
		logger.Error(err)
		return
	}

	outputEvent := handler(a.Input)
	p.completeAction(a, outputEvent)
}

// used to ensure panicing command handlers can be recovered gracefully by completing the action with a new fatal event
// populated by the error message returned
func (p pack) handlePanic(a *client.Action) {
	if r := recover(); r != nil {
		p.completeAction(a, NewFatalEvent(fmt.Sprintf("%v", r)))
		logger.Errorf("command handler for %q raised a panic: %s", a.CommandName, r)
	}
}

// completes the action by posting an event to the flyte api
func (p pack) completeAction(a *client.Action, event Event) {
	e := client.Event{
		Name:    event.EventDef.Name,
		Payload: event.Payload,
	}
	if err := p.client.CompleteAction(*a, e); err != nil {
		logger.Errorf("could not complete action %+v with event %+v: %s", a, e, err)
	}
}

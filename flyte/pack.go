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

// Package flyte is where you find the main public api for the flyte client. Almost all of your interactions with the
// client will occur here.
package flyte

import (
	"encoding/json"
	"net/url"
	"github.com/HotelsDotCom/flyte-client/client"
	"github.com/HotelsDotCom/go-logger"
	"time"
	"github.com/HotelsDotCom/flyte-client/healthcheck"
)

const (
	fatalEventName    = "FATAL"
	registerRetryWait = 3 * time.Second
)

type Pack interface {
	// Start will register the pack with the flyte server and will begin handling actions and invoking commands.
	// The pack will also be available to send observed events.
	Start()

	// SendEvent spontaneously sends an event that the pack has observed to the flyte server.
	SendEvent(Event) error
}

type pack struct {
	PackDef
	client           client.Client
	pollingFrequency time.Duration
	healthChecks 	 []healthcheck.HealthCheck
}

// Creates a Pack struct with the details from the pack definition and a connection to the flyte api through the client.
// Optionally, you can also pass in pack health checks
func NewPack(packDef PackDef, client client.Client, healthChecks ...healthcheck.HealthCheck) Pack {
	return pack{
		PackDef: packDef,
		client:  client,
		// Agreed that for now pack devs won't be able to/won't want to configure this polling rate.
		// If we get a use case that requires this to be changed then we can expose it then
		// (bearing in mind that this polling rate only comes into play if no actions are immediately available
		// - if actions are available then the pack/client will consume them as quickly as it can)
		pollingFrequency: 5 * time.Second,
		healthChecks: addDefaultHealthCheckIfNoneExist(healthChecks),
	}
}

func addDefaultHealthCheckIfNoneExist(healthChecks []healthcheck.HealthCheck) []healthcheck.HealthCheck {
	if len(healthChecks) == 0 {
		healthChecks = append(healthChecks, func() (name string, health healthcheck.Health) {
			return "DefaultCheck", healthcheck.Health{Healthy:true, Status: "Pack is running."}
		})
	}
	return healthChecks
}

// Registers the pack with the flyte server and starts handling actions from the flyte server and invoking the necessary commands.
// Once started the Pack is also available to send observed events.
// This will also start up a pack health check server.
func (p pack) Start() {
	if err := p.register(); err != nil {
		logger.Errorf("cannot register pack: %v", err)
		time.Sleep(registerRetryWait)
		p.Start()
		return
	}
	p.handleCommands()
	p.startHealthCheckServer()
}


// Spontaneously sends an event that the pack has observed to the flyte server.
func (p pack) SendEvent(event Event) error {
	return p.client.PostEvent(client.Event{
		Name:    event.EventDef.Name,
		Payload: event.Payload,
	})
}

var StartHealthCheckServer = true // this is only overridden for testing purposes

func (p pack) startHealthCheckServer() {
	if StartHealthCheckServer == true {
		healthcheck.Start(p.healthChecks)
	}
}

// The main configuration struct for defining a pack.
type PackDef struct {
	Name      		 string // the pack name
	Labels    		 map[string]string // the pack labels. These act as a filter that determines when the pack will execute against a flow
	EventDefs 		 []EventDef // the event definitions of a pack. These can be events a pack observes and sends spontaneously
	Commands  		 []Command // the commands a pack exposes
	HelpURL   		 *url.URL // a help url to a page that describes what the pack does and how it is used
}

// Defines an event. The help URL is optional.
type EventDef struct {
	Name    string
	HelpURL *url.URL
}

// Defines a command - its name, the events it can output and a handler for incoming actions. The help URL is optional.
type Command struct {
	Name         string // the name of the command
	OutputEvents []EventDef // the events a pack can output
	Handler      CommandHandler // the handler is where the functionality of a pack is implemented when a command is called
	HelpURL      *url.URL // optional
}

// Command handlers will be invoked with the input JSON when they are invoked from a flow step in the flyte server.
type CommandHandler func(input json.RawMessage) Event

// The event data the pack can send for events it observes (using SendEvent()) or from commands that have been called.
// The payload will be marshalled into JSON, so should be annotated appropriately.
type Event struct {
	EventDef EventDef
	Payload  interface{}
}

// This is the preferred way for packs to handle serious errors within the handler.
func NewFatalEvent(payload interface{}) Event {
	return Event{
		EventDef: EventDef{Name: fatalEventName},
		Payload:  payload,
	}
}

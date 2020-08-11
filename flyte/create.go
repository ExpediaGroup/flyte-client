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
	"github.com/ExpediaGroup/flyte-client/client"
	"net/url"
)

// this registers the pack with the flyte server
func (p pack) register() error {
	eventDefs, commands := aggregateAndConvert(p.EventDefs, p.Commands)
	return p.client.CreatePack(client.Pack{
		Name:      p.Name,
		Labels:    p.Labels,
		Links:     []client.Link{createLink(p.HelpURL, "help")},
		EventDefs: eventDefs,
		Commands:  commands,
	})
}

// deduplicates & converts the event definitions and converts the pack commands to the client commands type.
func aggregateAndConvert(eventDefs []EventDef, commands []Command) ([]client.EventDef, []client.Command) {
	eventDefsSet := make(map[string]client.EventDef)
	c := processCommands(commands, eventDefsSet)
	processEventDefs(eventDefs, eventDefsSet)
	return toSlice(eventDefsSet), c
}

// converts the commands to the client data type and adds the eventDefs returned by the commands to the set passed in
func processCommands(commands []Command, eventDefsSet map[string]client.EventDef) []client.Command {
	c := make([]client.Command, len(commands))
	for i, command := range commands {
		clientCommand := client.Command{
			Name:       command.Name,
			EventNames: processCommandEventDefs(command.OutputEvents, eventDefsSet),
		}
		if command.HelpURL != nil {
			clientCommand.Links = []client.Link{createLink(command.HelpURL, "help")}
		}
		c[i] = clientCommand
	}
	return c
}

// creates the slice of event names the command returns and also adds the eventDefs to the set
func processCommandEventDefs(eventDefs []EventDef, eventDefsSet map[string]client.EventDef) (eventNames []string) {
	eventNames = make([]string, len(eventDefs))
	for i, eventDef := range eventDefs {
		eventNames[i] = eventDef.Name
		addToEventDefsSet(eventDef, eventDefsSet)
	}
	return
}

// adds the event definitions to the eventDefsSet
func processEventDefs(eventDefs []EventDef, eventDefsSet map[string]client.EventDef) {
	for _, eventDef := range eventDefs {
		addToEventDefsSet(eventDef, eventDefsSet)
	}
}

// creates a client EventDef from a flyte EventDef passed in to it, and adds it to the eventDefsSet also passed in
func addToEventDefsSet(eventDef EventDef, eventDefsSet map[string]client.EventDef) {
	clientEventDef := client.EventDef{Name: eventDef.Name}
	if eventDef.HelpURL != nil {
		clientEventDef.Links = []client.Link{createLink(eventDef.HelpURL, "help")}
	}
	eventDefsSet[eventDef.Name] = clientEventDef
}

// creates a client.Link struct from the url passed in
func createLink(u *url.URL, rel string) client.Link {
	return client.Link{
		Href: u,
		Rel:  rel,
	}
}

// converts a map of client.EventDef's to a slice of client.EventDef's
func toSlice(m map[string]client.EventDef) []client.EventDef {
	var s []client.EventDef
	for _, eventDef := range m {
		s = append(s, eventDef)
	}
	return s
}

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

package client

import (
	"encoding/json"
	"net/url"
)

// the client Pack struct is used when registering with the flyte api.
type Pack struct {
	Name      string            `json:"name"`               // pack name
	Labels    map[string]string `json:"labels,omitempty"`   // pack labels - these act as a filter that determines when the pack will execute against a flow
	EventDefs []EventDef        `json:"events"`             // the event definitions of a pack. These can be events a pack observes and sends spontaneously
	Commands  []Command         `json:"commands,omitempty"` // the commands a pack exposes
	Links     []Link            `json:"links, omitempty"`   // contains links the pack uses, such as the take action url and the events url, or links the pack exposes such as the pack help url
}

// the event definition, this describes events a pack can send
type EventDef struct {
	Name  string `json:"name"`            // the event name
	Links []Link `json:"links,omitempty"` // the event link/s, optional. Could be a help link or anything related to the event
}

// the command struct represents the commands a pack exposes
type Command struct {
	Name       string   `json:"name"`            // command name
	EventNames []string `json:"events"`          // the command output events
	Links      []Link   `json:"links,omitempty"` // the command link/s, optional. Normally a help url
}

type Link struct {
	Href *url.URL
	Rel  string
}

// custom marshaller to avoid marshalling all url.URL fields
func (l Link) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Href string `json:"href"`
		Rel  string `json:"rel"`
	}{
		Href: l.Href.String(),
		Rel:  l.Rel,
	})
}

func (l *Link) UnmarshalJSON(data []byte) error {
	linkRaw := &struct {
		Href string `json:"href"`
		Rel  string `json:"rel"`
	}{}
	if err := json.Unmarshal(data, linkRaw); err != nil {
		return err
	}
	href, err := url.Parse(linkRaw.Href)
	if err != nil {
		return err
	}
	l.Href = href
	l.Rel = linkRaw.Rel
	return nil
}

type Event struct {
	Name    string      `json:"event"`
	Payload interface{} `json:"payload"`
}

type Action struct {
	CommandName string          `json:"command"`
	Input       json.RawMessage `json:"input"`
	Links       []Link          `json:"links"`
}

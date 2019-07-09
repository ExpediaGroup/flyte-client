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

// Package client is where the interaction between the client and the flyte api occurs.
// Normally you will only need to reference the client.NewClient(...) function.
package client

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/HotelsDotCom/flyte-client/config"
	"github.com/HotelsDotCom/go-logger"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type Client interface {
	// CreatePack is responsible for posting your pack to the flyte server.
	CreatePack(Pack) error
	// PostEvent posts events to the flyte server.
	PostEvent(Event) error
	// TakeAction takes the next action the pack should process. If no action is available, nil is returned.
	TakeAction() (*Action, error)
	// CompleteAction posts the action result to the flyte server.
	CompleteAction(Action, Event) error
	// GetFlyteHealthCheckURL gets the flyte api healthcheck url
	GetFlyteHealthCheckURL() (*url.URL, error)
}

type client struct {
	eventsURL     *url.URL
	baseURL       *url.URL
	takeActionURL *url.URL
	apiLinks      map[string][]Link
	httpClient    *http.Client
	jwt			  string
}

const (
	ApiVersion        = "v1"
	flyteApiRetryWait = 3 * time.Second
)

// To create a new client, please provide the url of the flyte server and the timeout.
// timeout specifies a time limit for requests made by this
// client. A timeout of zero means no timeout.
// Insecure mode is either true or false
func NewClient(rootURL *url.URL, timeout time.Duration) Client {
	return newClient(rootURL, timeout, false)
}

func NewInsecureClient(rootURL *url.URL, timeout time.Duration) Client {
	return newClient(rootURL, timeout, true)
}

func newClient(rootURL *url.URL, timeout time.Duration, isInsecure bool) Client {
	baseUrl := getBaseURL(*rootURL)

	client := &client{
		baseURL: baseUrl,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: isInsecure},
			},
		},
		jwt: config.GetJWT(),
	}
	client.getApiLinks()
	return client
}


// getBaseURL creates a url from the url path passed in and the apiVersion
func getBaseURL(u url.URL) *url.URL {
	u.Path = path.Join(u.Path, ApiVersion)
	return &u
}

// getApiLinks retrieves links from the flyte api server that are useful to the client such as packs url and health url and so on
func (c *client) getApiLinks() {
	var links map[string][]Link

	if err := c.getStruct(c.baseURL, &links); err != nil {
		logger.Errorf("cannot get api links: '%v'", err)
		time.Sleep(flyteApiRetryWait)
		c.getApiLinks()
		return
	}
	c.apiLinks = links
}

// CreatePack is responsible for posting your pack to the flyte server, making it available to be used by the flows.
func (c *client) CreatePack(pack Pack) error {

	var err error
	if err = c.registerPack(&pack); err != nil {
		return err
	}

	if c.eventsURL, err = findURLByRel(pack.Links, "event"); err != nil {
		return err
	}

	if c.takeActionURL, err = findURLByRel(pack.Links, "takeAction"); err != nil {
		return err
	}

	return nil
}

// registerPack posts the pack, and handles the response
func (c *client) registerPack(pack *Pack) error {
	packsURL, err := c.getPacksURL()
	if err != nil {
		return err
	}

	resp, err := c.post(packsURL, pack)
	if err != nil {
		return fmt.Errorf("error posting pack %+v to %s: %v", pack, packsURL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("pack not created, response was: %+v", resp)
	}

	err = json.NewDecoder(resp.Body).Decode(pack)
	if err != nil {
		return fmt.Errorf("could not deserialise response: %s", err)
	}

	return nil
}

// getPacksURL finds out where packs should be posted to
func (c *client) getPacksURL() (*url.URL, error) {
	return findURLByRel(c.apiLinks["links"], "pack/listPacks")
}

// GetFlyteHealthCheckURL finds out the flyte healthcheck url
func (c *client) GetFlyteHealthCheckURL() (*url.URL, error) {
	return findURLByRel(c.apiLinks["links"], "info/health")
}

// PostEvent posts events to the flyte server
func (c client) PostEvent(event Event) error {
	if c.eventsURL == nil {
		return errors.New("eventsURL not initialised - you must post a pack def first")
	}
	resp, err := c.post(c.eventsURL, event)
	if err != nil {
		return fmt.Errorf("error posting event %+v to %s: %v", event, c.eventsURL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("event %+v not accepted, response was: %+v", event, resp)
	}
	return nil
}

// TakeAction takes the next action the pack should process. If no action is available, nil is returned.
func (c client) TakeAction() (*Action, error) {
	if c.takeActionURL == nil {
		return nil, errors.New("takeActionURL not initialised - you must post a pack def first")
	}

	resp, err := c.post(c.takeActionURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error taking action from %s: %v", c.takeActionURL.String(), err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		a := &Action{}
		err = json.NewDecoder(resp.Body).Decode(a)
		return a, err
	case http.StatusNoContent:
		return nil, nil
	case http.StatusNotFound:
		return nil, NotFoundError{fmt.Sprintf("resource not found at %s", c.takeActionURL.String())}
	default:
		return nil, fmt.Errorf("error taking action from %s, response was: %+v", c.takeActionURL.String(), resp)
	}
}

// CompleteAction posts the action result to the flyte server.
func (c client) CompleteAction(action Action, event Event) error {
	resultURL, err := findURLByRel(action.Links, "actionResult")
	if err != nil {
		return err
	}
	resp, err := c.post(resultURL, event)
	if err != nil {
		return fmt.Errorf("error posting action result %+v to %s: %v", event, resultURL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("action result event %+v not processed successfully by flyte api, response was: %+v", event, resp)
	}
	return nil
}

// findURLByRel returns a link URL if found from the links passed in, else it will return an error.
func findURLByRel(links []Link, rel string) (*url.URL, error) {
	for _, l := range links {
		if strings.HasSuffix(l.Rel, rel) {
			return l.Href, nil
		}
	}
	return nil, fmt.Errorf("could not find link with rel %q in %v", rel, links)
}

type NotFoundError struct {
	Message string
}

func (e NotFoundError) Error() string {
	return e.Message
}

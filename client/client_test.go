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
	"bytes"
	"fmt"
	"github.com/HotelsDotCom/flyte-client/config"
	"github.com/HotelsDotCom/go-logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

/**
	NewClient, InsecureNewClient tests
 */

func Test_NewClient_ShouldSendAuthorizationHeaderWhenRetrievingApiLinks(t *testing.T) {
	// given the expected environment variable exists
	defer restoreGetEnvFunc()
	defer clearEnv()
	initTestEnv()
	setEnv(config.FlyteJWTEnvName, "a.jwt.token")

	// and we have a running server set to respond with flyte api links
	ts, rec := mockServerWithRecorder(http.StatusCreated, flyteApiLinksResponse)
	defer ts.Close()

	// when we create a new client
	baseUrl, _ := url.Parse(ts.URL)
	NewClient(baseUrl, 10*time.Second)

	// then
	require.NotEmpty(t, rec.reqs, "A http request must be set!")
	assert.Equal(t, "Bearer a.jwt.token", rec.reqs[0].Header.Get("Authorization"))
}

func Test_NewClient_ShouldNotSendAuthorizationHeaderWhenRetrievingApiLinks(t *testing.T) {
	// given the jwt environment variable does not exist and we have a running server set to respond with flyte api links
	ts, rec := mockServerWithRecorder(http.StatusCreated, flyteApiLinksResponse)
	defer ts.Close()

	// when we create a new client
	baseUrl, _ := url.Parse(ts.URL)
	NewClient(baseUrl, 10*time.Second)

	// then
	require.NotEmpty(t, rec.reqs, "A http request must be set!")
	assert.Equal(t, "", rec.reqs[0].Header.Get("Authorization"))
}

func Test_NewClient_ShouldRetryOnErrorGettingFlyteApiLinks(t *testing.T) {
	// given the mock flyte-api will first return an error response getting api links...then after retrying will return the expected response
	apiLinksFailCount := 1
	handler := func(w http.ResponseWriter, r *http.Request) {
		if apiLinksFailCount > 0 {
			apiLinksFailCount -= apiLinksFailCount
			w.Write(bytes.NewBufferString(flyteApiErrorResponse).Bytes())
			return
		}
		w.Write(bytes.NewBufferString(flyteApiLinksResponse).Bytes())
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// and code to record the log message/s
	logMsg := ""
	loggerFn := logger.Errorf
	logger.Errorf = func(msg string, args ...interface{}) { logMsg = fmt.Sprintf(msg, args...) }
	defer func() { logger.Errorf = loggerFn }()

	baseUrl, _ := url.Parse(server.URL)

	// when
	client := NewClient(baseUrl, 10*time.Second)

	// then a log error message will have been recorded...
	assert.Contains(t, logMsg, "cannot get api links:")
	// ...but the links are available after the retry
	healthCheckURL, _ := client.GetFlyteHealthCheckURL()
	assert.Equal(t, "http://example.com/v1/health", healthCheckURL.String())
}

func Test_InsecureNewClient_ShouldNotLogFatalWhenJWTIsNotProvided(t *testing.T) {
	// given no jwt exists in the environment var and server is set up
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write(bytes.NewBufferString(flyteApiLinksResponse).Bytes())
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	baseUrl, _ := url.Parse(server.URL)

	// when
	client := NewInsecureClient(baseUrl, 10*time.Second)

	// then log.Fatal is not called and client works as expected
	healthCheckURL, _ := client.GetFlyteHealthCheckURL()
	assert.Equal(t, "http://example.com/v1/health", healthCheckURL.String())
}

func Test_InsecureNewClient_ShouldRetryOnErrorGettingFlyteApiLinks(t *testing.T) {
	// given the mock flyte-api will first return an error response getting api links...then after retrying will return the expected response
	apiLinksFailCount := 1
	handler := func(w http.ResponseWriter, r *http.Request) {
		if apiLinksFailCount > 0 {
			apiLinksFailCount -= apiLinksFailCount
			w.Write(bytes.NewBufferString(flyteApiErrorResponse).Bytes())
			return
		}
		w.Write(bytes.NewBufferString(flyteApiLinksResponse).Bytes())
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// and code to record the log message/s
	logMsg := ""
	loggerFn := logger.Errorf
	logger.Errorf = func(msg string, args ...interface{}) { logMsg = fmt.Sprintf(msg, args...) }
	defer func() { logger.Errorf = loggerFn }()

	baseUrl, _ := url.Parse(server.URL)

	// when
	client := NewInsecureClient(baseUrl, 10*time.Second)

	// then a log error message will have been recorded...
	assert.Contains(t, logMsg, "cannot get api links:")
	// ...but the links are available after the retry
	healthCheckURL, _ := client.GetFlyteHealthCheckURL()
	assert.Equal(t, "http://example.com/v1/health", healthCheckURL.String())
}

/**
	CreatePack tests
 */

func Test_CreatePack_ShouldSendAuthorizationHeaderWhenRegisteringPack(t *testing.T) {
	// given we have a running server set to respond with a pack json
	ts, rec := mockServerWithRecorder(http.StatusCreated, slackPackResponse)
	defer ts.Close()

	// and the jwt environment variable exists
	defer restoreGetEnvFunc()
	defer clearEnv()
	initTestEnv()
	setEnv(config.FlyteJWTEnvName, "a.jwt.token")

	// and a client
	c := newTestClient(ts.URL, t)

	// when we create a pack
	err := c.CreatePack(Pack{Name: "Slack"})
	require.NoError(t, err)

	// then
	require.NotEmpty(t, rec.reqs, "A http request must be set!")
	assert.Equal(t, "Bearer a.jwt.token", rec.reqs[0].Header.Get("Authorization"))
}

func Test_CreatePack_ShouldNotSendAuthorizationHeaderWhenRegisteringPack(t *testing.T) {
	// given we have a running server set to respond with a pack json
	ts, rec := mockServerWithRecorder(http.StatusCreated, slackPackResponse)
	defer ts.Close()

	// and a client without the token set
	c := newTestClient(ts.URL, t)

	// when we create a pack
	err := c.CreatePack(Pack{Name: "Slack"})
	require.NoError(t, err)

	// then
	require.NotEmpty(t, rec.reqs, "A http request must be set!")
	assert.Equal(t, "", rec.reqs[0].Header.Get("Authorization"))
}

func Test_CreatePack_ShouldRegisterPackWithApiAndPopulateClientWithLinks(t *testing.T) {
	ts, rec := mockServerWithRecorder(http.StatusCreated, slackPackResponse)
	defer ts.Close()

	c := newTestClient(ts.URL, t)

	err := c.CreatePack(Pack{Name: "Slack"})
	require.NoError(t, err)

	assert.NotNil(t, c.takeActionURL)
	assert.Equal(t, "http://example.com/v1/packs/Slack/actions/take", c.takeActionURL.String())
	assert.Len(t, rec.reqs, 1)

	assert.NotNil(t, c.eventsURL)
	assert.Equal(t, "http://example.com/v1/packs/Slack/events", c.eventsURL.String())
}

func Test_CreatePack_ShouldReturnErrorIfTakeActionsLinksAreNotSet(t *testing.T) {
	ts := mockServer(http.StatusCreated, slackPackResponseWithNoTakeActionLink)
	defer ts.Close()

	c := newTestClient(ts.URL, t)

	err := c.CreatePack(Pack{Name: "Slack"})
	assert.Equal(t, "could not find link with rel \"takeAction\" in [{http://example.com/v1/packs/Slack/events http://example.com/swagger#/event}]", err.Error())
}

func Test_CreatePack_ShouldReturnErrorIfEventLinksAreNotSet(t *testing.T) {
	ts := mockServer(http.StatusCreated, slackPackResponseWithNoEventsLinks)
	defer ts.Close()

	c := newTestClient(ts.URL, t)

	err := c.CreatePack(Pack{Name: "Slack"})
	assert.Equal(t, "could not find link with rel \"event\" in [{http://example.com/v1/packs/Slack/actions/take http://example.com/swagger#!/action/takeAction}]", err.Error())
}

func Test_CreatePack_ShouldReturnErrorWhenLinksDoNotContainCorrectRel(t *testing.T) {
	ts := mockServer(http.StatusCreated, slackPackResponse)
	defer ts.Close()

	emptyLinks := map[string][]Link{}

	c := &client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		apiLinks:   emptyLinks,
	}

	err := c.CreatePack(Pack{Name: "Slack"})
	assert.Contains(t, err.Error(), `could not find link with rel "pack/listPacks"`)
}

func Test_CreatePack_ShouldReturnErrorIfPostingPackFails(t *testing.T) {
	// given a server with a handler that will timeout when called
	timeout := 10 * time.Millisecond
	handler := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(timeout + 1*time.Millisecond) // this will force a timeout on the http client call so it returns an error
	}
	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	// and a client setup to call the above server
	baseUrl, _ := url.Parse(ts.URL)
	c := &client{
		baseURL: baseUrl,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		apiLinks: map[string][]Link{"links": {{Href: baseUrl, Rel: "pack/listPacks"}}},
	}

	// when
	err := c.CreatePack(Pack{Name: "Slack"})

	// then
	assert.Contains(t, err.Error(), "error posting pack")
	assert.Contains(t, err.Error(), "Client.Timeout exceeded while awaiting headers")
}

func Test_CreatePack_ShouldReturnErrorIfStatusCodeIsNotStatusCreated(t *testing.T) {
	ts := mockServer(http.StatusNotFound, slackPackResponseWithNoEventsLinks)
	defer ts.Close()

	c := newTestClient(ts.URL, t)

	err := c.CreatePack(Pack{Name: "Slack"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pack not created, response was")
}

func Test_CreatePack_ShouldReturnErrorIfResponseCannotBeDecoded(t *testing.T) {
	ts := mockServer(http.StatusCreated, "invalidjson")
	defer ts.Close()

	c := newTestClient(ts.URL, t)

	err := c.CreatePack(Pack{Name: "Slack"})
	assert.EqualError(t, err, "could not deserialise response: invalid character 'i' looking for beginning of value")
}

/**
	PostEvent tests
 */

func Test_PostEvent_ShouldSendAuthorizationHeader(t *testing.T) {
	// given we have a running server
	ts, rec := mockServerWithRecorder(http.StatusAccepted, `{"some":"response"}`)
	defer ts.Close()

	// and the jwt environment variable exists
	defer restoreGetEnvFunc()
	defer clearEnv()
	initTestEnv()
	setEnv(config.FlyteJWTEnvName, "a.jwt.token")

	// and a client
	c := newTestClient(ts.URL, t)

	// and an events url set
	u, _ := url.Parse(fmt.Sprintf("%s/v1/packs/Slack/events", ts.URL))
	c.eventsURL = u

	// when
	err := c.PostEvent(Event{Name:"Dave", Payload:`{"some":"thing"}`})

	// then
	require.NoError(t, err)
	require.NotEmpty(t, rec.reqs, "A http request must be set!")
	assert.Equal(t, "Bearer a.jwt.token", rec.reqs[0].Header.Get("Authorization"))
}

func Test_PostEvent_ShouldNotSendAuthorizationHeader(t *testing.T) {
	// given we have a running server
	ts, rec := mockServerWithRecorder(http.StatusAccepted, `{"some":"response"}`)
	defer ts.Close()

	// and a client without the token set
	c := newTestClient(ts.URL, t)

	// and an events url set
	u, _ := url.Parse(fmt.Sprintf("%s/v1/packs/Slack/events", ts.URL))
	c.eventsURL = u

	// when
	err := c.PostEvent(Event{Name:"Dave", Payload:`{"some":"thing"}`})

	// then
	require.NoError(t, err)
	require.NotEmpty(t, rec.reqs, "A http request must be set!")
	assert.Equal(t, "", rec.reqs[0].Header.Get("Authorization"))
}

 /**
	TakeAction tests
 */

func Test_TakeAction_ShouldSendAuthorizationHeader(t *testing.T) {
	// given we have a running server
	ts, rec := mockServerWithRecorder(http.StatusOK, `{"some":"response"}`)
	defer ts.Close()

	// and the jwt environment variable exists
	defer restoreGetEnvFunc()
	defer clearEnv()
	initTestEnv()
	setEnv(config.FlyteJWTEnvName, "a.jwt.token")

	// and a client
	c := newTestClient(ts.URL, t)

	// and a take action url set
	u, _ := url.Parse(fmt.Sprintf("%s/v1/packs/Slack/actions/take", ts.URL))
	c.takeActionURL = u

	// when
	_, err := c.TakeAction()

	// then
	require.NoError(t, err)
	require.NotEmpty(t, rec.reqs, "A http request must be set!")
	assert.Equal(t, "Bearer a.jwt.token", rec.reqs[0].Header.Get("Authorization"))
}

func Test_TakeAction_ShouldNotSendAuthorizationHeader(t *testing.T) {
	// given we have a running server
	ts, rec := mockServerWithRecorder(http.StatusOK, `{"some":"response"}`)
	defer ts.Close()

	// and a client without the token set
	c := newTestClient(ts.URL, t)

	// and a take action url set
	u, _ := url.Parse(fmt.Sprintf("%s/v1/packs/Slack/actions/take", ts.URL))
	c.takeActionURL = u

	// when
	_, err := c.TakeAction()

	// then
	require.NoError(t, err)
	require.NotEmpty(t, rec.reqs, "A http request must be set!")
	assert.Equal(t, "", rec.reqs[0].Header.Get("Authorization"))
}

func Test_TakeAction_ShouldReturnSpecificErrorTypeAndMessageWhenResourceIsNotFound(t *testing.T) {
	ts := mockServer(http.StatusNotFound, "")
	defer ts.Close()

	c := newTestClient(ts.URL, t)
	u, err := url.Parse(ts.URL + "/take/action/url")
	require.NoError(t, err)

	c.takeActionURL = u
	_, err = c.TakeAction()

	require.IsType(t, NotFoundError{}, err)
	assert.EqualError(t, err, fmt.Sprintf("resource not found at %s/take/action/url", ts.URL))
}

/**
   CompleteAction tests
*/

func Test_CompleteAction_ShouldSendAuthorizationHeader(t *testing.T) {
	// given we have a running server
	ts, rec := mockServerWithRecorder(http.StatusAccepted, `{"some":"response"}`)
	defer ts.Close()

	// and the jwt environment variable exists
	defer restoreGetEnvFunc()
	defer clearEnv()
	initTestEnv()
	setEnv(config.FlyteJWTEnvName, "a.jwt.token")

	// and a client
	c := newTestClient(ts.URL, t)

	// and an action result url set
	actionResultUrl, _ := url.Parse(fmt.Sprintf("%s/v1/actionResult", ts.URL))
	action := Action{Links:[]Link{{Href:actionResultUrl, Rel:"actionResult"}}}

	// when
	err := c.CompleteAction(action, Event{Name:"Dave", Payload:`{"some":"thing"}`})

	// then
	require.NoError(t, err)
	require.NotEmpty(t, rec.reqs, "A http request must be set!")
	assert.Equal(t, "Bearer a.jwt.token", rec.reqs[0].Header.Get("Authorization"))
}

func Test_CompleteAction_ShouldNotSendAuthorizationHeader(t *testing.T) {
	// given we have a running server
	ts, rec := mockServerWithRecorder(http.StatusAccepted, `{"some":"response"}`)
	defer ts.Close()

	// and a client without the token set
	c := newTestClient(ts.URL, t)

	// and an action result url set
	actionResultUrl, _ := url.Parse(fmt.Sprintf("%s/v1/actionResult", ts.URL))
	action := Action{Links:[]Link{{Href:actionResultUrl, Rel:"actionResult"}}}

	// when
	err := c.CompleteAction(action, Event{Name:"Dave", Payload:`{"some":"thing"}`})

	// then
	require.NoError(t, err)
	require.NotEmpty(t, rec.reqs, "A http request must be set!")
	assert.Equal(t, "", rec.reqs[0].Header.Get("Authorization"))
}

/**
   GetFlyteHealthCheckURL tests
*/

func Test_GetFlyteHealthCheckURL_ShouldSelectFlyteHealthCheckUrlFromFlyteApiLinks(t *testing.T) {
	// given
	ts := mockServer(http.StatusOK, flyteApiLinksResponse)
	defer ts.Close()

	baseUrl, _ := url.Parse(ts.URL)
	client := NewClient(baseUrl, 10*time.Second)

	// when
	healthCheckURL, err := client.GetFlyteHealthCheckURL()

	// then
	require.NoError(t, err)
	assert.Equal(t, "http://example.com/v1/health", healthCheckURL.String())
}

func Test_GetFlyteHealthCheckURL_ShouldReturnErrorWhenItCannotGetHealthCheckURLFromFlyteApiLinks(t *testing.T) {
	// given
	ts := mockServer(http.StatusOK, flyteApiNoLinksResponse)
	defer ts.Close()

	baseUrl, _ := url.Parse(ts.URL)
	client := NewClient(baseUrl, 10*time.Second)

	// when
	_, err := client.GetFlyteHealthCheckURL()

	// then
	assert.Equal(t, "could not find link with rel \"info/health\" in []", err.Error())
}

var flyteApiLinksResponse = `{
	"links": [
		{
			"href": "http://example.com/v1",
			"rel": "self"
		},
		{
			"href": "http://example.com/",
			"rel": "up"
		},
		{
			"href": "http://example.com/swagger#!/info/v1",
			"rel": "help"
		},
		{
			"href": "http://example.com/v1/health",
			"rel": "http://example.com/swagger#!/info/health"
		},
		{
			"href": "http://example.com/v1/packs",
			"rel": "http://example.com/swagger#!/pack/listPacks"
		},
		{
			"href": "http://example.com/v1/flows",
			"rel": "http://example.com/swagger#!/flow/listFlows"
		},
		{
			"href": "http://example.com/v1/datastore",
			"rel": "http://example.com/swagger#!/datastore/listDataItems"
		},
		{
			"href": "http://example.com/v1/audit/flows",
			"rel": "http://example.com/swagger#!/audit/findFlows"
		},
		{
			"href": "http://example.com/v1/swagger",
			"rel": "http://example.com/swagger"
		}
	]
}`

var flyteApiNoLinksResponse = `{
	"links": []
}`

var flyteApiErrorResponse = `{
	"error!"
}`

var slackPackResponse = `
{
    "id": "Slack",
    "name": "Slack",
    "links": [
        {
            "href": "http://example.com/v1/packs/Slack/actions/take",
            "rel": "http://example.com/swagger#!/action/takeAction"
        },
        {
            "href": "http://example.com/v1/packs/Slack/events",
            "rel": "http://example.com/swagger#/event"
        }
    ]
}
`

var slackPackResponseWithNoTakeActionLink = `
{
    "id": "Slack",
    "name": "Slack",
    "links": [
        {
            "href": "http://example.com/v1/packs/Slack/events",
            "rel": "http://example.com/swagger#/event"
        }
    ]
}
`

var slackPackResponseWithNoEventsLinks = `
{
    "id": "Slack",
    "name": "Slack",
    "links": [
        {
            "href": "http://example.com/v1/packs/Slack/actions/take",
            "rel": "http://example.com/swagger#!/action/takeAction"
        }
    ]
}
`

func mockServer(status int, body string) *httptest.Server {
	ts, _ := mockServerWithRecorder(status, body)
	return ts
}

func mockServerWithRecorder(status int, body string) (*httptest.Server, *requestsRec) {
	rec := &requestsRec{
		reqs: []*http.Request{},
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)

		w.WriteHeader(status)
		w.Write([]byte(body))
	}
	return httptest.NewServer(http.HandlerFunc(handler)), rec
}

func newTestClient(serverURL string, t *testing.T) *client {
	u, err := url.Parse(serverURL)
	require.NoError(t, err)

	return &client{
		httpClient: newHttpClient(5 * time.Second, false),
		apiLinks:   map[string][]Link{"links": {{Href: u, Rel: "pack/listPacks"}}},
	}
}

type requestsRec struct {
	reqs []*http.Request
}

func (rr *requestsRec) add(r *http.Request) {
	rr.reqs = append(rr.reqs, r)
}

// environment variable help

var envvars = map[string]string{}
var origGetEnv = config.GetEnv

func initTestEnv() {
	config.GetEnv = func(name string) string {
		return envvars[name]
	}
}

func restoreGetEnvFunc() {
	config.GetEnv = origGetEnv
}

func setEnv(name, value string) {
	envvars[name] = value
}

func clearEnv() {
	envvars = map[string]string{}
}


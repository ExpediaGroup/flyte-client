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
	"github.com/HotelsDotCom/go-logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

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

func Test_GetFlyteHealthCheckURL_ShouldSelectFlyteHealthCheckUrlFromFlyteApiLinks(t *testing.T) {
	// given
	ts := mockServer(http.StatusOK, string(bytes.NewBufferString(flyteApiLinksResponse).Bytes()))
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
	ts := mockServer(http.StatusOK, string(bytes.NewBufferString(flyteApiNoLinksResponse).Bytes()))
	defer ts.Close()

	baseUrl, _ := url.Parse(ts.URL)
	client := NewClient(baseUrl, 10*time.Second)

	// when
	_, err := client.GetFlyteHealthCheckURL()

	// then
	assert.Equal(t, "could not find link with rel \"info/health\" in []", err.Error())
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
	assert.EqualError(t, err, fmt.Sprintf("Resource not found at %s/take/action/url", ts.URL))
}

//should register pack with API, and have populated links
func TestClient_CreatePack(t *testing.T) {

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
	assert.Len(t, rec.reqs, 1)

}

func TestCreatePackShouldReturnErrorIfTakeActionsLinksAreNotSet(t *testing.T) {
	ts := mockServer(http.StatusCreated, slackPackResponseWithNoTakeAction)
	defer ts.Close()

	c := newTestClient(ts.URL, t)

	err := c.CreatePack(Pack{Name: "Slack"})
	assert.Equal(t, "could not find link with rel \"takeAction\" in [{http://example.com/v1/packs/Slack/events http://example.com/swagger#/event}]", err.Error())
}

func TestCreatePackShouldReturnErrorIfEventLinksAreNotSet(t *testing.T) {
	ts := mockServer(http.StatusCreated, slackPackResponseWithNoEvents)
	defer ts.Close()

	c := newTestClient(ts.URL, t)

	err := c.CreatePack(Pack{Name: "Slack"})
	assert.Equal(t, "could not find link with rel \"event\" in [{http://example.com/v1/packs/Slack/actions/take http://example.com/swagger#!/action/takeAction}]", err.Error())
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

var slackPackResponseWithNoTakeAction = `
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

var slackPackResponseWithNoEvents = `
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
		httpClient: &http.Client{Timeout: 5 * time.Second},
		apiLinks:   map[string][]Link{"links": {{Href: u, Rel: "pack/listPacks"}}},
	}
}

type requestsRec struct {
	reqs []*http.Request
}

func (rr *requestsRec) add(r *http.Request) {
	rr.reqs = append(rr.reqs, r)
}

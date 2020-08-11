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
	"github.com/ExpediaGroup/flyte-client/client"
	"github.com/HotelsDotCom/go-logger"
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
	"time"
)

type mockClient struct {
	takeAction func() (*client.Action, error)
}

func (m mockClient) TakeAction() (*client.Action, error) {
	return m.takeAction()
}

func TestGetNextActionShouldReturnActionOnSuccess(t *testing.T) {
	mock := mockClient{takeAction: func() (*client.Action, error) {
		return &client.Action{}, nil
	}}

	pack := pack{client: mock, pollingFrequency: 1 * time.Millisecond}

	action := pack.getNextAction()

	assert.NotNil(t, action)
}

func TestGetNextActionShouldContinuePollingWhileReceivingANoContentResponse(t *testing.T) {
	counter := 0
	mock := mockClient{takeAction: func() (*client.Action, error) {
		counter++
		if counter == 5 { // overkill, but proves it polls multiple times!
			return &client.Action{}, nil
		}
		return nil, nil // what TakeAction returns on '204 No Content'
	}}

	pack := pack{client: mock, pollingFrequency: 1 * time.Millisecond}

	action := pack.getNextAction()

	if assert.NotNil(t, action) {
		assert.Equal(t, 5, counter, "getNextAction: should have polled 5 times but only polled %d time(s)")
	}
}

func TestGetNextActionShouldContinuePollingWhileReceivingUnexpectedErrorResponses(t *testing.T) {
	counter := 0
	mock := mockClient{takeAction: func() (*client.Action, error) {
		counter++
		if counter == 5 { // overkill, but proves it polls multiple times!
			return &client.Action{}, nil
		}
		return nil, fmt.Errorf("some random error: %d", counter) // just to show that it changes
	}}

	pack := pack{client: mock, pollingFrequency: 100 * time.Millisecond}

	action := pack.getNextAction()

	if assert.NotNil(t, action) {
		assert.Equal(t, 5, counter, "getNextAction: should have polled 5 times but only polled %d time(s)")
	}
}

func TestGetNextActionShouldLogFatalErrorAndDieOn404FromResource(t *testing.T) {
	counter := 0
	mock := mockClient{takeAction: func() (*client.Action, error) {
		counter++
		if counter > 1 {
			return &client.Action{}, nil // exits the method which would have been terminated
		}
		return nil, client.NotFoundError{"404 from resource ... this will be seen in logs because logger.Fatalf() drops through in test."}
	}}

	pack := pack{client: mock, pollingFrequency: 100 * time.Millisecond}

	origFunc := logger.Fatal
	defer func() { logger.Fatal = origFunc }()

	loggerCalled := false
	var exitMessage string
	logger.Fatal = func(args ...interface{}) {
		loggerCalled = true
		exitMessage = fmt.Sprint(args...)
	}

	pack.getNextAction()

	assert.True(t, loggerCalled)
	assert.Equal(t, "Pack not found while polling for actions. Exiting.", exitMessage)
}

// Rest of methods required for Client interface

func (mockClient) CreatePack(client.Pack) error {
	return nil
}

func (mockClient) PostEvent(client.Event) error {
	return nil
}

func (mockClient) CompleteAction(client.Action, client.Event) error {
	return nil
}

func (mockClient) GetFlyteHealthCheckURL() (*url.URL, error) {
	return nil, nil
}

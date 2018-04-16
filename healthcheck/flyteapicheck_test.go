package healthcheck

import (
	"testing"
	"net/http"
	"github.com/stretchr/testify/assert"
	"errors"
	"net/url"
	"github.com/HotelsDotCom/flyte-client/client"
	"net/http/httptest"
	"time"
)

func Test_FlyteApiHealthCheck_ShouldReturnHealthyResponse(t *testing.T) {
	// given a mock http server to represent the flyte-api healthcheck call
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// and a mock client that contains the healthcheck url
	flyteApiHealthCheckURL := server.URL + "/health"
	client := MockClient{
		healthCheckURL: createURL(flyteApiHealthCheckURL),
	}

	// when
	health := FlyteApiHealthCheck(client)

	// then
	assert.Equal(t, true, health.Healthy)
	assert.Equal(t, "flyte-api is up and responding to requests. url: '" + flyteApiHealthCheckURL + "'", health.Status)
}

func Test_FlyteApiHealthCheck_ShouldReturnErrorMessage_WhenErrorGettingHealthCheckURLFromFlyteApi(t *testing.T) {
	// given a mock client that cannot retrieve the healthcheck url
	client := MockClient{
		err: errors.New("flyte-api down!"),
	}

	// when
	health := FlyteApiHealthCheck(client)

	// then
	assert.Equal(t, true, health.Healthy)
	assert.Equal(t, "cannot perform flyte-api healthcheck. error getting flyte-api healthcheck url. error: 'flyte-api down!'", health.Status)
}

func Test_FlyteApiHealthCheck_ShouldReturnErrorMessage_WhenHttpRequestToFlyteApiHealthCheckReturnsAnError(t *testing.T) {
	// given a mock http server to represent the flyte-api healthcheck call
	handler := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(timeout + 1) // this will force a timeout on the http client call so it returns an error
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// and a mock client that contains the healthcheck url
	flyteApiHealthCheckURL := server.URL + "/health"
	client := MockClient{
		healthCheckURL: createURL(flyteApiHealthCheckURL),
	}

	// when
	health := FlyteApiHealthCheck(client)

	// then
	assert.Equal(t, true, health.Healthy)
	assert.Contains(t, health.Status, "error in http call to flyte-api:")
	assert.Contains(t, health.Status, "url: '" + flyteApiHealthCheckURL + "'")
}

func Test_FlyteApiHealthCheck_ShouldReturnErrorMessage_WhenHttpStatusReturnedFromFlyteApiHealthCheckIsNot200(t *testing.T) {
	// given a mock http server to represent the flyte-api healthcheck call
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// and a mock client that contains the healthcheck url
	flyteApiHealthCheckURL := server.URL + "/health"
	client := MockClient{
		healthCheckURL: createURL(flyteApiHealthCheckURL),
	}

	// when
	health := FlyteApiHealthCheck(client)

	// then
	assert.Equal(t, true, health.Healthy)
	assert.Equal(t, "flyte-api is not responding as expected. http status: '500 Internal Server Error'. url: '" + flyteApiHealthCheckURL + "'", health.Status)
}

func createURL(u string) *url.URL {
	url, _ := url.Parse(u)
	return url
}

// client.Client
type MockClient struct {
	healthCheckURL *url.URL
	err      	   error
}

func (c MockClient) GetFlyteHealthCheckURL()(*url.URL, error) {
	return c.healthCheckURL, c.err
}

func (c MockClient) CreatePack(client.Pack) error {
	return nil
}

func (c MockClient) PostEvent(client.Event) error {
	return nil
}

func (c MockClient) TakeAction() (*client.Action, error) {
	return nil, nil
}

func (c MockClient) CompleteAction(client.Action, client.Event) error {
	return nil
}

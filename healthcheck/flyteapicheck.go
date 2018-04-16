package healthcheck

import (
	"net/http"
	"fmt"
	"time"
	"github.com/HotelsDotCom/flyte-client/client"
)

const timeout = time.Duration(5) * time.Second

func FlyteApiHealthCheck(c client.Client) Health {
	healthCheckURL, err := c.GetFlyteHealthCheckURL()
	if err != nil {
		return Health{Healthy:true, Status: fmt.Sprintf("cannot perform flyte-api healthcheck. error getting flyte-api healthcheck url. error: '%s'", err.Error())}
	}

	httpClient := &http.Client{
		Timeout: timeout,
	}

	r, err := httpClient.Get(healthCheckURL.String())
	if err != nil {
		return Health{Healthy:true, Status: fmt.Sprintf("error in http call to flyte-api: '%s'. url: '%s'", err.Error(), healthCheckURL)}
	}
	if r.StatusCode != http.StatusOK {
		return Health{Healthy:true, Status: fmt.Sprintf("flyte-api is not responding as expected. http status: '%s'. url: '%s'", r.Status, healthCheckURL)}
	}
	return Health{Healthy:true, Status: fmt.Sprintf("flyte-api is up and responding to requests. url: '%s'", healthCheckURL)}
}

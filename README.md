# flyte-client

[![Build Status](https://travis-ci.org/ExpediaGroup/flyte-client.svg?branch=master)](https://travis-ci.org/ExpediaGroup/flyte-client)

flyte-client is a Go library designed to make the writing of flyte packs simple. 
The client handles the registration of a pack with the flyte server, consuming and handling command actions, and gives the ability to send
pack events to the flyte server. This allows the pack writer to concentrate solely on the functionality of their pack.


# Getting started

## Install
```
go get github.com/ExpediaGroup/flyte-client
```

## Usage
```
import "github.com/ExpediaGroup/flyte-client"
```

* Clone this repo
The 'entry point' for the client is the flyte package (specifically flyte/pack.go). To build the client locally therefore run:

```
go build -v ./flyte
```

#### Go Docs
The Go docs can be found at:

```
# start godoc server
godoc -http=:6060
# navigate to
http://localhost:6060/pkg/github.com/ExpediaGroup/flyte-client
```

#### Structs
The structs that a pack developer interacts with can be found in flyte/pack.go.
The main ones are as follows:

- **PackDef**: defines the pack - it's name, what commands it has and what events it can raise.
- **EventDef**: defines an event - it's name and an optional help URL to describe the event in more detail
- **Command**: defines a command that can be called on the pack - its name, what event it returns, and an optional help URL to describe the event in more detail. 
Also includes a **CommandHandler** which is the function that will be executed when the command is called.
- **Event**: this struct is sent from the pack to the flyte server api - it contains the name of the event and it's payload 


#### Events

Packs can send events to the flyte server in 3 ways:

1. The pack can observe something happening and spontaneously send an event to the flyte server. 
For example a chat-ops pack, may observe an instant message being sent and raise a "MessageSent" event to the flyte server. 
It would do this by calling the `SendEvent()` function on the `Pack` interface.

1. A flow on the flyte server creates an action for the pack to execute. 
The client will poll for this action and invoke the relevant `CommandHandler` that the pack dev has defined. 
This handler will return an event that the client will then send to the flyte server. 
For example the same IM pack as above may have a 'sendMessage' command that would return either a 'MessageSent' or 'MessageSendFailure' event.

1. The client will produce `FATAL` events as a result of panic happening while handling a `Command`. This will be intercepted by the
client and it will recover. If they need to, packs can also produce `FATAL` events themselves as a result of the handling if they detect
any errors using `NewFatalEvent(payload interface{})`. This is preferred over panicking in the handler. E.g.
```go
func handle(message json.RawMessage) Event {
        ...
        // Preferred
        event := NewFatalEvent(Foo{"Error message"})
        // Over
        panic(Foo{"Error message"})
        ...
}
```

When defining the above pack, you will notice that 'EventDefs' are defined at the pack level (PackDef.EventDefs) and at the command level (PackDef.Commands.EventDef).
The 'EventDefs' field on a Command is mandatory, so for the example pack above you would have to specify the eventdefs for both 'MessageSent' and 'MessageSendFailure' on the'sendMessage' Command struct.
The 'EventDefs' on the PackDef are optional. Here you would specify any events that the pack observes and sends spontaneously. 
If the event you want to define is already defined in a command (as with 'MessageSent' above) then you are not required to add it to the separate EventDefs section - however there is no harm in doing so.

#### Health checks

You can add health checks to your pack in the following way:

```go
    // example (hardcoded) healthchecks
    someCheck := func() (name string, health healthcheck.Health) {
      return "Some Check", healthcheck.Health{Healthy:true, Status: "All good"}
    }
    otherCheck := func() (name string, health healthcheck.Health) {
      return "Other Check", healthcheck.Health{Healthy:true, Status: "Ok"}
    }
      
    // now pass in to 'flyte.NewPack(...)'. when the pack is started this will also start a webserver ready to call your healthchecks.
    // if no healthchecks are passed in the pack health check URL will always return a default healthy response.
    p := flyte.NewPack(packDef, client, someCheck, otherCheck) // healthchecks are optional

```

Then simply go to the pack health check URL i.e. 'http://localhost:8090' and you will be presented with a json response:

```json
    {
        "Some Check": {
            "healthy": true,
            "status": "All good"
        },
        "Other Check": {
            "healthy": true,
            "status": "Ok"
        }
    }
```

The following http header response codes will also be returned:

- 200: All health checks passed.
- 500: One or more of the health checks failed. JSON results will be returned as normal in the response body.
- 500: JSON marshalling error.

Errors will also be logged.

A simple health check has been provided for you to check on the status of flyte-api: healthcheck.FlyteApiHealthCheck(c client.Client) and
can be used in the following way:

```go
    ...
    
    c := client.NewClient(createURL("http://example.com"), 10 * time.Second)
    
    flyteApiHealthCheck := func() (name string, health healthcheck.Health) {
      return "FlyteApiCheck", healthcheck.FlyteApiHealthCheck(c)
    }
    
    p := flyte.NewPack(packDef, c, flyteApiHealthCheck)
    p.Start()

```


#### JWT Authorisation

If your pack needs to send a JSON Web Token along with each http request, please set the JWT string value in the following 
environment variable:

-  FLYTE_JWT

If not provided no authorisation will occur.

Note: You are strongly advised to only use JWT authorisation over https.

#### Help URLs

You will notice that a `helpURL` field is present in 3 locations - PackDef, Command, and EventDef. 
These correspond to the URLs visible in the json on the flyte server at that level.
The help URLs are all optional, though you should generally always have the pack level one. 
It's up to pack developers to provide which ones they think are the most useful. 
For a simple pack you'd probably just provide a single pack level help link. 
For a more complex pack you might want to deep link commands & event definitions to a specific piece of documentation.

The URL should link to a page that describes what a flow writer needs to know i.e. what the pack does, the format of the json in the event payloads and the format of the json for command inputs.
It's up to the pack dev where and how they host their help docs - for example it could be a link to a README file or a hosted web page.

#### Example Pack

The example below shows how to create a simplified "Jira Pack". The pack exposes a "createIssue" command allowing users to create tickets.
If this command succeeds it returns an "IssueCreated" event, if it fails it returns a "IssueCreationError" event.



```go
package main

import (
	"encoding/json"
	"log"
	"net/url"
	"github.com/ExpediaGroup/flyte-client/client"
	"github.com/ExpediaGroup/flyte-client/flyte"
	"github.com/ExpediaGroup/flyte-client/healthcheck"
	"time"
)

func main() {

    // First we create EventDefs that describe what events our pack can raise. 
    // An EventDef contains the name of the event (mandatory) and a help URL (optional)
	issueCreatedEventDef := flyte.EventDef{
		Name: "IssueCreated",
	}

	issueCreationErrorEventDef := flyte.EventDef{
		Name: "IssueCreationError",
	}

    // This pack has a single "createIssue" command. To implement a command we must provide a "CommandHandler" function matching the signature below.
    // The client will call this handler every time it receives a "createIssue" action from the flyte api. The handler will take the input JSON from the action
    // and must return a flyte.Event. Handlers are where the functionality of the pack is implemented so will likely form the bulk of most packs. 
	createIssueHandler := func(input json.RawMessage) flyte.Event {
	    
	    // deserialize the raw JSON into our domain struct
		var createIssueInput CreateIssueInput
		json.Unmarshal(input, &createIssueInput)

		// call some ticket creation code...
        // ...
        
        // if it succeeds then return something like the following. The payload will be serialised to JSON and sent to the flyte api server.
		return flyte.Event{
			EventDef: issueCreatedEventDef,
			Payload: IssueCreatedPayload{
				Project:  "FOO",
				IssueId:  "123",
				Location: createURL("http://jira/FOO/123"),
			},
		}
	}

    // Next we create a struct that defines the "createIssue" command. Note the handler above is passed to it.
    // Also note that we specify what events the command can output.
    // The help URL is optional
	createIssueCommand := flyte.Command{
		Name: "createIssue",
		OutputEvents: []flyte.EventDef{
			issueCreatedEventDef,
			issueCreationErrorEventDef,
		},
		HelpURL: createURL("http://jirapack/help#create-issue-command"),
		Handler: createIssueHandler,
	}
	
	// The final struct we must define is the PackDef struct which pulls together the above structs to give the full definition of the pack.
	packDef := flyte.PackDef{
		Name:     "JiraPack",
		Commands: []flyte.Command{createIssueCommand},
		HelpURL:  createURL("http://jirapack/help#create-issue-command"),
	}
	
    // Finally we call NewPack() to create a pack struct. This can then be started by calling Start()
    // Start() will register the pack with the Flyte api server and will invoke the relevant CommandHandler for any actions the api posts.
    p := flyte.NewPack(packDef, client.NewClient(createURL("http://example.com"), 10*time.Second))
	p.Start()
	select{} // p.Start() doesn't block anymore, so we need to block here.
}

type CreateIssueInput struct {
	Project    string `json:"project"`
	TicketText string `json:"ticketText"`
}

type IssueCreatedPayload struct {
	Project  string   `json:"project"`
	IssueId  string   `json:"issueId"`
	Location *url.URL `json:"location"`
}

type IssueCreationErrorPayload struct {
	Project   string `json:"project"`
	ErrorText string `json:"error"`
}

func createURL(u string) *url.URL {
	url, _ := url.Parse(u)
	return url
}

func flyteApiHealthCheck(c client.Client) healthcheck.HealthCheck {
	// a provided healthcheck for you to use and check on the status of the flyte-api
    return func() (name string, health healthcheck.Health) {
      return "FlyteApiCheck", healthcheck.FlyteApiHealthCheck(c)
    }
}

func exampleHealthCheck() healthcheck.HealthCheck {
	// a hardcoded example of a healthcheck
	return func() (name string, health healthcheck.Health) {
      return "Some Check", healthcheck.Health{Healthy:true, Status: "All good"}
    }
}
```

## Running Tests

To run just the unit tests use:

```
go test ./...
```

To run all tests, then use:

```
go test ./... -tags=acceptance
```

If you want to provide a custom Flyte-API image to run tests against, set the environment variable FLYTE_API_IMAGE. E.g:
```
export FLYTE_API_IMAGE=flyte:test
```
If this value is not set, it will default to a predefined value found in `acceptance-test/flyteapi.go`, as `flyteApiDefaultImage`.

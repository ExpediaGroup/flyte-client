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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// marshalls the body passed in into JSON then posts to the specified url, returning a http response
// will return error if cannot marshall JSON, cannot create a http request or for a httpClient posting error
func (c client) post(u *url.URL, body interface{}) (*http.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal body '%+v': %v", body, err)
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

// performs a http get on the specified url, returning the http response.
// will return error if there is a problem creating the http request or if there is a httpClient error
func (c client) get(u *url.URL) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %v", err)
	}
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

// gets a struct from the specified url and deserialises it into the supplied interface
// will return error if there is a problem getting the struct or if it cannot deserialise into the supplied interface
func (c *client) getStruct(u *url.URL, s interface{}) error {
	resp, err := c.get(u)
	if err != nil {
		return fmt.Errorf("error getting url %q: %s", u.String(), err)
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(s)
	if err != nil {
		return fmt.Errorf("could not deserialise response from %q: %s", u.String(), err)
	}
	return nil
}

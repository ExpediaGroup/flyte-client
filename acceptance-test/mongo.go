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

package tests

import (
	"github.com/HotelsDotCom/go-docker-client"
)

type Mongo struct {
	mongoContainer docker.Container
}

func StartMongo() (*Mongo, error) {
	d, err := docker.NewDocker()
	if err != nil {
		return nil, err
	}

	mongoContainer, err := d.Run("mongo", "mongo", nil, []string{"27017"})
	if err != nil {
		return nil, err
	}

	return &Mongo{mongoContainer}, nil
}

func (m *Mongo) GetIP() (string, error) {
	return m.mongoContainer.GetIP()
}

func (m *Mongo) Stop() error {
	return m.mongoContainer.StopAndRemove()
}

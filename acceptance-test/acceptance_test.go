// +build acceptance

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
	"github.com/HotelsDotCom/go-logger"
	"testing"
)

type FeatureFile struct {
	name  string
	tests []Test
}

type Test struct {
	name     string
	testFunc func(t *testing.T)
}

var suite = []FeatureFile{
	{"client", ClientFeatures},
	{"pack", PackFeatures},
}

func TestFeatures(t *testing.T) {
	mng, err := StartMongo()
	if err != nil {
		logger.Fatalf("Unable to start mongo: %s", err.Error())
	}
	defer tearDownMongo(mng)

	flyte, err := StartFlyte(*mng)
	if err != nil {
		logger.Fatalf("Unable to start flyte: %s", err.Error())
	}
	defer tearDownFlyte(flyte)

	for _, feature := range suite {
		t.Run(feature.name, func(t *testing.T) {
			for _, test := range feature.tests {
				t.Run(test.name, test.testFunc)
			}
		})
	}
}

func tearDownFlyte(flyte *Flyte) {
	if err := flyte.Stop(); err != nil {
		logger.Errorf("unable to stop flyte api: %v", err)
	}
}

func tearDownMongo(mongo *Mongo) {
	if err := mongo.Stop(); err != nil {
		logger.Errorf("unable to stop mongo: %v", err)
	}
}

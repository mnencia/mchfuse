/*
Copyright 2020 Marco Nenciarini <mnencia@gmail.com>

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

package mch_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mnencia/mchfuse/mch"
)

func TestISOTimeJSON(t *testing.T) {
	testString := `"2020-04-10T18:56:55.003+02:00"`

	testDate, err := time.ParseInLocation(`"`+time.RFC3339Nano+`"`, testString, time.UTC)
	if err != nil {
		t.Fatal(err)
	}

	var parsedDate mch.ISOTime

	err = json.Unmarshal([]byte(testString), &parsedDate)
	if err != nil {
		t.Fatal(err)
	}

	if !testDate.Equal(time.Time(parsedDate)) {
		t.Fatalf("Time %s does not equal expected %s", time.Time(parsedDate), testDate)
	}

	jsonDate, err := json.Marshal(parsedDate)
	if err != nil {
		t.Fatal(err)
	}

	if testString != string(jsonDate) {
		t.Fatalf("String format for %s does not equal expected %s", string(jsonDate), testString)
	}
}

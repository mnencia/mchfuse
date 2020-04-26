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

package mch

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

const (
	configURL = "https://config.mycloud.com/config/v1/config"
)

type configurationResponse struct {
	Configuration Configuration `json:"data"`
}

type Configuration struct {
	ConfigurationID string                            `json:"configurationId"`
	ComponentMap    map[string]map[string]interface{} `json:"componentMap"`
}

func GetConfiguration() (*Configuration, error) {
	resp, err := http.Get(configURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var configResponse configurationResponse

	err = json.Unmarshal(body, &configResponse)
	if err != nil {
		return nil, err
	}

	return &configResponse.Configuration, nil
}

func (c Configuration) GetString(section, config string) string {
	return c.ComponentMap[section][config].(string)
}

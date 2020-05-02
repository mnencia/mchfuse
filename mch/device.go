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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type DeviceInfo struct {
	Data   []Device `json:"data"`
	client *Client
}

type Device struct {
	DeviceID             string        `json:"deviceId"`
	Name                 string        `json:"name"`
	Mac                  string        `json:"mac"`
	DeviceType           string        `json:"type"`
	CreatedOn            ISOTime       `json:"createdOn"`
	AttachedStatus       string        `json:"attachedStatus"`
	Lang                 string        `json:"lang"`
	Network              DeviceNetwork `json:"network"`
	LastHDStoragePercent int           `json:"lastHDStoragePercent"`
	CloudConnected       bool          `json:"cloudConnected"`
	OwnerAccess          bool          `json:"ownerAccess"`
	SerialNumber         string        `json:"serialNumber"`
	APIVersion           string        `json:"apiVersion"`
	client               *Client
}

type DeviceNetwork struct {
	LocalIPAddress              string `json:"localIpAddress"`
	ExternalIPAddress           string `json:"externalIpAddress"`
	LocalHTTPPort               int    `json:"localHttpPort"`
	LocalHTTPSPort              int    `json:"localHttpsPort"`
	PortForwardPort             int    `json:"portForwardPort"`
	TunnelID                    string `json:"tunnelId"`
	InternalDNSName             string `json:"internalDNSName"`
	InternalURL                 string `json:"internalURL"`
	PortForwardURL              string `json:"portForwardURL"`
	PortForwardDomain           string `json:"portForwardDomain"`
	ProxyURL                    string `json:"proxyURL"`
	ExternalURI                 string `json:"externalURI"`
	PortForwardInfoUpdateStatus string `json:"portForwardInfoUpdateStatus"`
}

func (di DeviceInfo) Find(name string) *Device {
	for _, device := range di.Data {
		if device.Name == name || device.DeviceID == name {
			return &device
		}
	}

	return nil
}

func (d *Device) NewAuthorizedRequest(method, path string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s/sdk/%s", d.Network.InternalURL, strings.TrimLeft(path, "/"))

	req, err := d.client.NewAuthorizedRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (d *Device) fileSearchParents(ids string, pageToken string) (*FileList, error) {
	req, err := d.NewAuthorizedRequest("GET", "/v2/filesSearch/parents", nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("ids", ids)
	q.Add("fields", "pageToken,"+FileFields)
	q.Add("hidden", d.client.OSType)

	if pageToken != "" {
		q.Add("pageToken", pageToken)
	}

	req.URL.RawQuery = q.Encode()

	resp, err := d.client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"unexpected status code %v searching files by parents %v at %v",
			resp.StatusCode,
			ids,
			resp.Request.URL,
		)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var fileList FileList

	err = json.Unmarshal(body, &fileList)
	if err != nil {
		return nil, err
	}

	fileList.ETag = resp.Header.Get("Etag")
	fileList.client = d.client
	fileList.device = d

	for i := range fileList.Files {
		fileList.Files[i].client = d.client
		fileList.Files[i].device = d
	}

	return &fileList, nil
}

func (d *Device) fileSearchParentAndName(parentID string, name string) (*File, error) {
	req, err := d.NewAuthorizedRequest(
		"GET",
		"/v2/filesSearch/parentAndName",
		nil,
	)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("name", name)
	q.Add("parentID", parentID)
	q.Add("fields", FileFields)
	req.URL.RawQuery = q.Encode()

	resp, err := d.client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, nil
	default:
		return nil, fmt.Errorf(
			"unexpected status code %v searching files by parent %v and name %v at %v",
			resp.StatusCode,
			parentID,
			name,
			resp.Request.URL,
		)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var child File

	err = json.Unmarshal(body, &child)
	if err != nil {
		return nil, err
	}

	child.client = d.client
	child.device = d

	return &child, nil
}

func (d *Device) fileByID(id string, file *File) (*File, error) {
	req, err := d.NewAuthorizedRequest(
		"GET",
		fmt.Sprintf("/v2/files/%s", id),
		nil,
	)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("fields", FileFields)
	req.URL.RawQuery = q.Encode()

	resp, err := d.client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"unexpected status code %v retrieveing file metadata %v at %v",
			resp.StatusCode,
			id,
			resp.Request.URL,
		)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, file)
	if err != nil {
		return nil, err
	}

	file.client = d.client
	file.device = d

	return file, nil
}

func (d *Device) Root() (*File, error) {
	return d.fileByID("root", &File{})
}

func (d *Device) GetFileByPath(path string) (*File, error) {
	path = strings.Trim(path, "/")

	current, err := d.Root()
	if err != nil {
		return nil, err
	}

	// If path is empty or is '/' return the root
	if path == "" {
		return current, nil
	}

	for _, dir := range strings.Split(path, "/") {
		if !current.IsDirectory() {
			return nil, fmt.Errorf("path component %s is not a directory", current.Name)
		}

		files, err := current.ListDirectory()
		if err != nil {
			return nil, err
		}

		entry, found := files[dir]
		if !found {
			return nil, fmt.Errorf("path component %s not found", dir)
		}

		current = &entry
	}

	return current, nil
}

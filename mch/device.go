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
	"net"
	"net/http"
	"strings"
	"time"
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
	connectionMode       DeviceConnectionMode
	connectionCheckedAt  time.Time
}

type DeviceNetwork struct {
	LocalIPAddress              net.IP `json:"localIpAddress"`
	ExternalIPAddress           net.IP `json:"externalIpAddress"`
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

// DeviceConnectionMode represents the current connection status for a device.
type DeviceConnectionMode int

const (
	UnknownConnection DeviceConnectionMode = iota
	InternalConnection
	ExternalConnection
)

const ConnectionRecheckTime = 30 * time.Second

func (dm DeviceConnectionMode) String() string {
	switch dm {
	case UnknownConnection:
		return "UnknownConnectionMode"
	case InternalConnection:
		return "InternalConnectionMode"
	case ExternalConnection:
		return "ExternalConnectionMode"
	default:
		return fmt.Sprintf("DeviceConnectionMode(%v)", int(dm))
	}
}

func (di DeviceInfo) Find(name string) *Device {
	for _, device := range di.Data {
		if device.Name == name || device.DeviceID == name {
			return &device
		}
	}

	return nil
}

func (d *Device) checkConnectionMode() bool {
	oldMode := d.connectionMode
	address := d.Network.InternalDNSName
	timeout := 1 * time.Second

	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		d.connectionMode = ExternalConnection
	} else {
		_ = conn.Close()
		d.connectionMode = InternalConnection
	}

	d.connectionCheckedAt = time.Now()

	return oldMode != d.connectionMode
}

func (d *Device) DeviceURI() string {
	if d.connectionMode == UnknownConnection || time.Since(d.connectionCheckedAt) > ConnectionRecheckTime {
		d.checkConnectionMode()
	}

	if d.connectionMode == ExternalConnection {
		return d.Network.ExternalURI
	}

	return fmt.Sprintf("https://%s", d.Network.InternalDNSName)
}

func (d *Device) getURI(path string) string {
	url := fmt.Sprintf("%s/sdk/%s", d.DeviceURI(), strings.TrimLeft(path, "/"))
	return url
}

func (d *Device) api(
	method, path string,
	body io.Reader,
	requestMutator func(*http.Request),
) (
	*http.Response,
	error,
) {
	uri := d.getURI(path)

	req, err := d.client.NewAuthorizedRequest(method, uri, body)
	if err != nil {
		return nil, err
	}

	if requestMutator != nil {
		requestMutator(req)
	}

	resp, err := d.client.HTTPClient.Do(req)
	if err != nil {
		if d.checkConnectionMode() {
			return nil, fmt.Errorf("connection mode changed after an error: %w", err)
		}

		return nil, err
	}

	return resp, nil
}

func (d *Device) fileSearchParents(ids string, pageToken string) (*FileList, error) {
	resp, err := d.api("GET", "/v2/filesSearch/parents", nil, func(req *http.Request) {
		q := req.URL.Query()
		q.Add("ids", ids)
		q.Add("fields", "pageToken,"+FileFields)
		q.Add("hidden", d.client.OSType)

		if pageToken != "" {
			q.Add("pageToken", pageToken)
		}

		req.URL.RawQuery = q.Encode()
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"status code %v searching files by parents %v at %v: %w",
			resp.StatusCode,
			ids,
			resp.Request.URL,
			ErrorUnexpectedStatusCode,
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
	resp, err := d.api(
		"GET",
		"/v2/filesSearch/parentAndName",
		nil,
		func(req *http.Request) {
			q := req.URL.Query()
			q.Add("name", name)
			q.Add("parentID", parentID)
			q.Add("fields", FileFields)
			req.URL.RawQuery = q.Encode()
		},
	)
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
			"status code %v searching files by parent %v and name %v at %v: %w",
			resp.StatusCode,
			parentID,
			name,
			resp.Request.URL,
			ErrorUnexpectedStatusCode,
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
	resp, err := d.api(
		"GET",
		fmt.Sprintf("/v2/files/%s", id),
		nil,
		func(req *http.Request) {
			q := req.URL.Query()
			q.Add("fields", FileFields)
			req.URL.RawQuery = q.Encode()
		},
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"status code %v retrieveing file metadata %v at %v: %w",
			resp.StatusCode,
			id,
			resp.Request.URL,
			ErrorUnexpectedStatusCode,
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
			return nil, fmt.Errorf("path component %s is not a directory: %w", current.Name,
				ErrorInvalidOperation)
		}

		files, err := current.ListDirectory()
		if err != nil {
			return nil, err
		}

		entry, found := files[dir]
		if !found {
			return nil, fmt.Errorf("path component %s not found: %w", dir, ErrorInvalidOperation)
		}

		current = &entry
	}

	return current, nil
}

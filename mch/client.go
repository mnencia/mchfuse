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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type Client struct {
	AccessToken   string         `json:"access_token"`
	RefreshToken  string         `json:"refresh_token"`
	IDToken       string         `json:"id_token"`
	Scope         string         `json:"scope"`
	TokenType     string         `json:"token_type"`
	ExpiresIn     int            `json:"expires_in"`
	UserID        string         `json:"user_id,omitempty"`
	Configuration *Configuration `json:"configuration,omitempty"`
	OSType        string         `json:"os_type,omitempty"`
	HTTPClient    http.Client    `json:"-"`
}

var ErrorUnexpectedStatusCode = errors.New("unexpected status code")

func Login(username string, password string) (*Client, error) {
	config, err := GetConfiguration()
	if err != nil {
		return nil, err
	}

	client := Client{Configuration: config, OSType: osType()}

	req := map[string]string{
		"grant_type":    "http://auth0.com/oauth/grant-type/password-realm",
		"realm":         "Username-Password-Authentication",
		"audience":      "mycloud.com",
		"username":      username,
		"password":      password,
		"scope":         "openid offline_access nas_read_write nas_read_only user_read device_read",
		"client_id":     "9B0Gi617tROKHc2rS95sT1yJzR6MkQDm",
		"client_secret": "oSJOB1KOWnLVZm11DVknu2wZkTj5AGKxcINEDtEUPE30jHKvEqorM8ocWbyo17Hd",
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := client.HTTPClient.Post(
		fmt.Sprintf("%s/oauth/token", config.GetString("cloud.service.urls", "service.auth0.url")),
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code %v logging in on %v: %w",
			resp.StatusCode, resp.Request.URL, ErrorUnexpectedStatusCode)
	}

	if err := client.unmarshalAuthResponse(resp); err != nil {
		return nil, err
	}

	return &client, nil
}

func osType() string {
	switch runtime.GOOS {
	case "linux", "windows":
		return runtime.GOOS
	case "darwin":
		return "mac"
	default:
		return "none"
	}
}

func (c *Client) refreshAccessToken() error {
	req := map[string]string{
		"audience":      "mycloud.com",
		"client_id":     c.Configuration.GetString("com.wd.portal", "portal.auth0.client"),
		"grant_type":    "refresh_token",
		"refresh_token": c.RefreshToken,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Post(
		fmt.Sprintf("%s/oauth/token", c.Configuration.GetString("cloud.service.urls", "service.auth0.url")),
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code %v refreshing token at %v: %w", resp.StatusCode, resp.Request.URL,
			ErrorUnexpectedStatusCode)
	}

	if err := c.unmarshalAuthResponse(resp); err != nil {
		return err
	}

	return nil
}

func (c *Client) unmarshalAuthResponse(response *http.Response) error {
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, c)
	if err != nil {
		return err
	}

	claims := jwt.MapClaims{}
	if _, _, err := new(jwt.Parser).ParseUnverified(c.IDToken, claims); err != nil {
		return err
	}

	c.UserID = claims["sub"].(string)

	return nil
}

func (c *Client) isAccessTokenExpired() bool {
	claims := jwt.MapClaims{}
	if _, _, err := new(jwt.Parser).ParseUnverified(c.AccessToken, claims); err != nil {
		return true
	}

	exp := int64(claims["exp"].(float64))

	return time.Now().Unix() > exp
}

func (c *Client) NewAuthorizedRequest(method, url string, body io.Reader) (*http.Request, error) {
	if c.isAccessTokenExpired() {
		if err := c.refreshAccessToken(); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+c.AccessToken)

	return req, nil
}

func (c *Client) DeviceInfo() (*DeviceInfo, error) {
	req, err := c.NewAuthorizedRequest(
		"GET",
		fmt.Sprintf(
			"%s/device/v1/user/%s",
			c.Configuration.GetString("cloud.service.urls", "service.device.url"),
			c.UserID,
		),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code %v getting deviceinfo on %v: %w", resp.StatusCode, resp.Request.URL,
			ErrorUnexpectedStatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var deviceList DeviceInfo

	err = json.Unmarshal(body, &deviceList)
	if err != nil {
		return nil, err
	}

	deviceList.client = c
	for i := range deviceList.Data {
		deviceList.Data[i].client = c
	}

	return &deviceList, nil
}

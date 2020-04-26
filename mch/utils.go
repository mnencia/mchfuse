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
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/relvacode/iso8601"
)

type ISOTime time.Time

func (t *ISOTime) UnmarshalJSON(input []byte) error {
	strInput := string(input)
	strInput = strings.Trim(strInput, `"`)

	newTime, err := iso8601.ParseString(strInput)
	if err != nil {
		return err
	}

	*t = ISOTime(newTime)

	return nil
}

func (t ISOTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

func (t ISOTime) String() string {
	return time.Time(t).Format(time.RFC3339Nano)
}

type MultipartBody struct {
	buffer *bytes.Buffer
	writer *multipart.Writer
}

func NewMultipartBody(reqJSON interface{}) (*MultipartBody, error) {
	buffer := new(bytes.Buffer)
	mb := &MultipartBody{
		buffer: buffer,
		writer: multipart.NewWriter(buffer),
	}

	dataJSON, err := json.Marshal(reqJSON)
	if err != nil {
		return nil, err
	}

	metadataHeader := textproto.MIMEHeader{}
	metadataHeader.Set("Content-Type", "application/json")

	part, err := mb.writer.CreatePart(metadataHeader)
	if err != nil {
		return nil, err
	}

	if _, err := part.Write(dataJSON); err != nil {
		return nil, err
	}

	err = mb.writer.Close()
	if err != nil {
		return nil, err
	}

	return mb, nil
}

func (mb MultipartBody) Buffer() *bytes.Buffer {
	return mb.buffer
}

func (mb MultipartBody) AddContentType(req *http.Request) {
	req.Header.Add(
		"Content-Type",
		fmt.Sprintf("multipart/related; boundary=%s", mb.writer.Boundary()),
	)
}

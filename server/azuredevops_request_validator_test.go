// Copyright 2017 HootSuite Media Inc.
//
// Licensed under the Apache License, Version 2.0 (the License);
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an AS IS BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Modified hereafter by contributors to runatlantis/atlantis.

package server_test

import (
	"bytes"
	"net/http"
	"testing"

	. "github.com/petergtz/pegomock"
	"github.com/runatlantis/atlantis/server"
	. "github.com/runatlantis/atlantis/testing"
)

func TestAzureDevopsValidate_WithBasicAuthErr(t *testing.T) {
	t.Log("if the request does not have a valid basic auth user and password there is an error")
	RegisterMockTestingT(t)
	g := server.DefaultAzureDevopsRequestValidator{}
	buf := bytes.NewBufferString("")
	req, err := http.NewRequest("POST", "http://localhost/event", buf)
	Ok(t, err)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz") // user:pass
	req.Header.Set("Content-Type", "application/json")

	_, err = g.Validate(req, []byte("user"), []byte("wrongpass"))
	Assert(t, err != nil, "error should not be nil")
	Equals(t, "ValidatePayload authentication failed", err.Error())
}

func TestAzureDevopsValidate_WithBasicAuth(t *testing.T) {
	t.Log("if the request has a valid basic auth user and password the payload is returned")
	RegisterMockTestingT(t)
	g := server.DefaultAzureDevopsRequestValidator{}
	buf := bytes.NewBufferString(`{"yo":true}`)
	req, err := http.NewRequest("POST", "http://localhost/event", buf)
	Ok(t, err)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz") // user:pass
	req.Header.Set("Content-Type", "application/json")

	bs, err := g.Validate(req, []byte("user"), []byte("pass"))
	Ok(t, err)
	Equals(t, `{"yo":true}`, string(bs))
}

func TestAzureDevopsValidate_WithoutSecretInvalidContentType(t *testing.T) {
	t.Log("if the request has an invalid content type an error is returned")
	RegisterMockTestingT(t)
	g := server.DefaultAzureDevopsRequestValidator{}
	buf := bytes.NewBufferString("")
	req, err := http.NewRequest("POST", "http://localhost/event", buf)
	Ok(t, err)
	req.Header.Set("Content-Type", "invalid")

	_, err = g.Validate(req, nil, nil)
	Assert(t, err != nil, "error should not be nil")
	Equals(t, "webhook request has unsupported Content-Type \"invalid\"", err.Error())
}

func TestAzureDevopsValidate_WithoutSecretJSON(t *testing.T) {
	t.Log("if the request is JSON the body is returned")
	RegisterMockTestingT(t)
	g := server.DefaultAzureDevopsRequestValidator{}
	buf := bytes.NewBufferString(`{"yo":true}`)
	req, err := http.NewRequest("POST", "http://localhost/event", buf)
	Ok(t, err)
	req.Header.Set("Content-Type", "application/json")

	bs, err := g.Validate(req, nil, nil)
	Ok(t, err)
	Equals(t, `{"yo":true}`, string(bs))
}
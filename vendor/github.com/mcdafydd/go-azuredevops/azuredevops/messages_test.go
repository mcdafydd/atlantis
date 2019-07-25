// Copyright 2016 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted for Azure Devops

package azuredevops_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mcdafydd/go-azuredevops/azuredevops"
)

func TestParseWebHook(t *testing.T) {
	payload := `{
		"eventType":"git.pullrequest.created",
		"resource":{
			"isDraft":true
		}
	}`
	got, err := azuredevops.ParseWebHook([]byte(payload))
	if err != nil {
		t.Fatalf("ParseWebHook: %v", err)
	}
	got.Resource = nil
	want := &azuredevops.Event{}
	err = json.Unmarshal([]byte(payload), &want)
	if err != nil {
		t.Fatalf("ParseWebHook: %v", err)
	}
	if !cmp.Equal(got, want) {
		diff := cmp.Diff(got, want)
		t.Errorf("ParseWebHook error: %s", diff)
	}
}

func TestActivityID(t *testing.T) {
	id := "8970a780-244e-11e7-91ca-da3aabcb9793"

	req, err := http.NewRequest("POST", "http://localhost", nil)
	if err != nil {
		t.Fatalf("ActivityID: %v", err)
	}
	req.Header.Set("X-VSS-ActivityID", id)

	got := azuredevops.GetActivityID(req)
	if got != id {
		t.Errorf("ActivityID(%#v) = %q, want %q", req, got, id)
	}
}

func TestRequestID(t *testing.T) {
	id := "|2c08c6334570ae4bb625b27e533afd00.1fc0bd4d_1fc0bd50_791563."

	req, err := http.NewRequest("POST", "http://localhost", nil)
	if err != nil {
		t.Fatalf("RequestID: %v", err)
	}
	req.Header.Set("Request-Id", id)

	got := azuredevops.GetRequestID(req)
	if got != id {
		t.Errorf("RequestID(%#v) = %q, want %q", req, got, id)
	}
}

func TestSubscriptionID(t *testing.T) {
	id := "6b9490e4-940d-4d16-8dae-d36580e7e2b4"

	req, err := http.NewRequest("POST", "http://localhost", nil)
	if err != nil {
		t.Fatalf("SubscriptionID: %v", err)
	}
	req.Header.Set("X-VSS-SubscriptionId", id)

	got := azuredevops.GetSubscriptionID(req)
	if got != id {
		t.Errorf("SubscriptionID(%#v) = %q, want %q", req, got, id)
	}
}

func TestValidatePayload(t *testing.T) {
	user := "testuser"
	pass := "testpass"
	want := []byte(`{"payload":"test"}`)

	body := struct {
		Payload string `json:"payload"`
	}{
		Payload: "test",
	}
	v, _ := json.Marshal(body)
	buf := bytes.NewBuffer(v)
	req, err := http.NewRequest("POST", "http://localhost/event", buf)
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(user, pass)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	got, _ := azuredevops.ValidatePayload(req, []byte(user), []byte(pass))

	if !bytes.Equal(got, want) {
		t.Fatalf("ValidatePayload: %v", err)
	}
}

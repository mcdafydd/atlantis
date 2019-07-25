// Copyright 2016 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted for Azure Devops

package azuredevops_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mcdafydd/go-azuredevops/azuredevops"
)

func TestParsePayload(t *testing.T) {
	tests := []struct {
		payload   interface{}
		eventType string
		want      interface{}
		err       error
	}{
		{
			payload:   &azuredevops.GitPullRequest{},
			eventType: "git.pullrequest.created",
			want:      &azuredevops.GitPullRequest{},
			err:       nil,
		},
		{
			payload:   &azuredevops.GitPullRequest{},
			eventType: "git.pullrequest.merged",
			want:      &azuredevops.GitPullRequest{},
			err:       nil,
		},
		{
			payload:   &azuredevops.GitPullRequest{},
			eventType: "git.pullrequest.updated",
			want:      &azuredevops.GitPullRequest{},
			err:       nil,
		},
		{
			payload:   &azuredevops.GitPush{},
			eventType: "git.push",
			want:      &azuredevops.GitPush{},
			err:       nil,
		},
		{
			payload:   &azuredevops.WorkItem{},
			eventType: "workitem.commented",
			want:      &azuredevops.WorkItem{},
			err:       nil,
		},
		{
			payload:   &azuredevops.WorkItemUpdate{},
			eventType: "workitem.updated",
			want:      &azuredevops.WorkItemUpdate{},
			err:       nil,
		},
		{
			payload:   nil,
			eventType: "git.pullrequest.created",
			want:      nil,
			err:       nil,
		},
		{
			payload:   "{}",
			eventType: "",
			want:      nil,
			err:       errors.New("Unknown EventType in webhook payload"),
		},
	}

	for _, test := range tests {
		event := new(azuredevops.Event)
		event.EventType = test.eventType
		payload, err := json.Marshal(test.payload)
		event.RawPayload = (json.RawMessage)(payload)
		if err != nil {
			t.Fatalf("Marshal(%#v): %v", test.payload, err)
		}
		want := test.want
		got, err := event.ParsePayload()
		if err != nil && test.err.Error() != err.Error() {
			t.Fatalf("ParsePayload: %v", err)
		}
		if !cmp.Equal(got, want) {
			diff := cmp.Diff(got, want)
			t.Errorf("ParsePayload error: %s", diff)
		}

	}
}

package azuredevops_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mcdafydd/go-azuredevops/azuredevops"
)

const (
	gitRefsListURL      = "/o/p/_apis/git/repositories/r/refs/heads"
	gitRefsListResponse = `{
		"count": 6,
		"value": [
		  {
			"name": "refs/heads/develop",
			"objectId": "67cae2b029dff7eb3dc062b49403aaedca5bad8d",
			"url": "https://fabrikam.visualstudio.com/_apis/git/repositories/278d5cd2-584d-4b63-824a-2ba458937249/refs/heads/develop"
		  },
		  {
			"name": "refs/heads/master",
			"objectId": "23d0bc5b128a10056dc68afece360d8a0fabb014",
			"url": "https://fabrikam.visualstudio.com/_apis/git/repositories/278d5cd2-584d-4b63-824a-2ba458937249/refs/heads/master"
		  },
		  {
			"name": "refs/heads/npaulk/feature",
			"objectId": "23d0bc5b128a10056dc68afece360d8a0fabb014",
			"url": "https://fabrikam.visualstudio.com/_apis/git/repositories/278d5cd2-584d-4b63-824a-2ba458937249/refs/heads/npaulk/feature"
		  },
		  {
			"name": "refs/tags/v1.0",
			"objectId": "23d0bc5b128a10056dc68afece360d8a0fabb014",
			"url": "https://fabrikam.visualstudio.com/_apis/git/repositories/278d5cd2-584d-4b63-824a-2ba458937249/refs/tags/v1.0"
		  },
		  {
			"name": "refs/tags/v1.1",
			"objectId": "23d0bc5b128a10056dc68afece360d8a0fabb014",
			"url": "https://fabrikam.visualstudio.com/_apis/git/repositories/278d5cd2-584d-4b63-824a-2ba458937249/refs/tags/v1.1"
		  },
		  {
			"name": "refs/tags/v2.0",
			"objectId": "23d0bc5b128a10056dc68afece360d8a0fabb014",
			"url": "https://fabrikam.visualstudio.com/_apis/git/repositories/278d5cd2-584d-4b63-824a-2ba458937249/refs/tags/v2.0"
		  }
		]
		}`
	gitRepositoryURL         = "/o/p/_apis/git/repositories/r"
	gitGetRepositoryResponse = `{
		"serverUrl": "https://dev.azure.com/fabrikam",
		"collection": {
			"id": "e22ddea7-989e-455d-b46a-67e991b04714",
			"name": "fabrikam",
			"url": "https://dev.azure.com/fabrikam/_apis/projectCollections/e22ddea7-989e-455d-b46a-67e991b04714"
		},
		"repository": {
			"id": "2f3d611a-f012-4b39-b157-8db63f380226",
			"name": "FabrikamCloud",
			"url": "https://dev.azure.com/fabrikam/_apis/git/repositories/2f3d611a-f012-4b39-b157-8db63f380226",
			"project": {
				"id": "3b3ae425-0079-421f-9101-bcf15d6df041",
				"name": "FabrikamCloud",
				"url": "https://dev.azure.com/fabrikam/_apis/projects/3b3ae425-0079-421f-9101-bcf15d6df041",
				"state": 1,
				"revision": 411518573
			},
			"remoteUrl": "https://dev.azure.com/fabrikam/FabrikamCloud/_git/FabrikamCloud"
		}
	}`
	gitCreateStatusURL      = "/o/p/_apis/git/repositories/r/commits/67cae2b029dff7eb3dc062b49403aaedca5bad8d/statuses"
	gitCreateStatusResponse = `{
		"state": "succeeded",
		"description": "The build is successful",
		"context": {
			"name": "Build123",
			"genre": "continuous-integration"
		},
		"targetUrl": "https://ci.fabrikam.com/my-project/build/123 "
	}`
)

func TestGitService_ListRefs(t *testing.T) {
	tt := []struct {
		name     string
		URL      string
		response string
		count    int
		index    int
		refName  string
		refID    string
	}{
		{name: "return 6 refs", URL: gitRefsListURL, response: gitRefsListResponse, count: 6, index: 0, refName: "refs/heads/develop", refID: "67cae2b029dff7eb3dc062b49403aaedca5bad8d"},
		{name: "can handle no refs returned", URL: gitRefsListURL, response: "{}", count: 0, index: -1},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			c, mux, _, teardown := setup()
			defer teardown()

			mux.HandleFunc(tc.URL, func(w http.ResponseWriter, r *http.Request) {
				testMethod(t, r, "GET")
				json := tc.response
				fmt.Fprint(w, json)
			})

			opts := azuredevops.GitRefListOptions{}
			refs, _, err := c.Git.ListRefs(context.Background(), "o", "p", "r", "heads", &opts)
			if err != nil {
				t.Fatalf("returned error: %v", err)
			}

			if tc.index > -1 {
				if *refs[tc.index].Name != tc.refName {
					t.Fatalf("expected git ref name %s, got %s", tc.refName, *refs[tc.index].Name)
				}
				if *refs[tc.index].ObjectID != tc.refID {
					t.Fatalf("expected git ref object id %s, got %s", tc.refID, *refs[tc.index].ObjectID)
				}
			}

			if len(refs) != tc.count {
				t.Fatalf("expected length of git refs to be %d; got %d", tc.count, len(refs))
			}

		})
	}
}

func TestGitService_Get(t *testing.T) {
	tt := []struct {
		name     string
		URL      string
		response string
		count    int
		repoName string
		id       string
	}{
		{name: "GetRepository() success", URL: gitRepositoryURL, response: gitGetRepositoryResponse, count: 1, repoName: "r", id: "2f3d611a-f012-4b39-b157-8db63f380226"},
		{name: "GetRepository() empty response", URL: gitRepositoryURL, response: "{}", count: 0, repoName: "", id: ""},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			c, mux, _, teardown := setup()
			defer teardown()

			mux.HandleFunc(tc.URL, func(w http.ResponseWriter, r *http.Request) {
				testMethod(t, r, "GET")
				json := tc.response
				fmt.Fprint(w, json)
			})

			_, _, err := c.Git.GetRepository(context.Background(), "o", "p", "r")
			if err != nil {
				t.Fatalf("returned error: %v", err)
			}
		})
	}
}

func TestGitService_CreateStatus(t *testing.T) {
	n := "Build123"
	g := "continuous-integration"
	gitContext := azuredevops.GitStatusContext{
		Name:  &n,
		Genre: &g,
	}

	tt := []struct {
		name        string
		URL         string
		response    string
		count       int
		description string
		targetUrl   string
		state       string
		gitContext  *azuredevops.GitStatusContext
	}{
		{name: "CreateStatus() success", URL: gitCreateStatusURL, response: gitCreateStatusResponse, count: 1, description: "The build is successful", targetUrl: "", state: "succeeded", gitContext: &gitContext},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			c, mux, _, teardown := setup()
			defer teardown()

			mux.HandleFunc(tc.URL, func(w http.ResponseWriter, r *http.Request) {
				testMethod(t, r, "POST")
				json := tc.response
				fmt.Fprint(w, json)
			})

			// Build the example request payload
			// https://docs.microsoft.com/en-us/rest/api/azure/devops/git/statuses/create?view#examples
			s := "The build is successful"
			state := "succeeded"
			target := "https://ci.fabrikam.com/my-project/build/123"
			status := azuredevops.GitStatus{
				Context:     tc.gitContext,
				Description: &s,
				State:       &state,
				TargetURL:   &target,
			}
			r, _, err := c.Git.CreateStatus(context.Background(), "o", "p", "r", "67cae2b029dff7eb3dc062b49403aaedca5bad8d", status)
			if err != nil {
				t.Fatalf("returned error: %v", err)
			}

			if !cmp.Equal(r.Context, tc.gitContext) {
				diff := cmp.Diff(r.Context, tc.gitContext)
				t.Errorf("Git.GetRef error: %s", diff)
			}
		})
	}
}

// azuredevops.VersionControlChangeType
func TestGitService_GetChanges(t *testing.T) {
	changeMap := map[string]int{
		azuredevops.Add.String(): 1,
	}

	changes := &azuredevops.GitChange{
		ChangeID:   Int(1),
		ChangeType: String(azuredevops.Add.String()),
	}
	changesList := []*azuredevops.GitChange{changes}
	want := &azuredevops.GitCommitChanges{
		ChangeCounts: &changeMap,
		Changes:      changesList,
	}

	client, mux, _, teardown := setup()
	defer teardown()
	mux.HandleFunc("/o/p/_apis/git/repositories/r/commits/67cae2b029dff7eb3dc062b49403aaedca5bad8d/changes", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"changeCounts": {
				"add": 1
			},
			"changes": [{
				"changeId": 1,
				"changeType": "add"
			}]
			}`)
	})

	got, _, err := client.Git.GetChanges(context.Background(), "o", "p", "r", "67cae2b029dff7eb3dc062b49403aaedca5bad8d")
	if err != nil {
		t.Fatalf("returned error: %v", err)
	}

	if !cmp.Equal(got, want) {
		diff := cmp.Diff(got, want)
		fmt.Printf(diff)
		t.Errorf("Git.GetChanges returned %+v, want %+v", got, want)
	}
}

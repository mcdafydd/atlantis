package azuredevops_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mcdafydd/go-azuredevops/azuredevops"
)

const (
	pullrequestsListURL = "/AZURE_DEVOPS_Project/_apis/git/pullrequests"
	// https://docs.microsoft.com/en-us/rest/api/vsts/git/pull%20requests/get%20pull%20requests%20by%20project
	pullrequestsResponse = `{
		"value": [
		  {
			"repository": {
			  "id": "3411ebc1-d5aa-464f-9615-0b527bc66719",
			  "name": "2016_10_31",
			  "url": "https://fabrikam.visualstudio.com/_apis/git/repositories/3411ebc1-d5aa-464f-9615-0b527bc66719",
			  "project": {
				"id": "a7573007-bbb3-4341-b726-0c4148a07853",
				"name": "2016_10_31",
				"state": "unchanged"
			  }
			},
			"pullRequestId": 22,
			"codeReviewId": 22,
			"status": "active",
			"createdBy": {
			  "id": "d6245f20-2af8-44f4-9451-8107cb2767db",
			  "displayName": "Normal Paulk",
			  "uniqueName": "fabrikamfiber16@hotmail.com",
			  "url": "https://fabrikam.visualstudio.com/_apis/Identities/d6245f20-2af8-44f4-9451-8107cb2767db",
			  "imageUrl": "https://fabrikam.visualstudio.com/_api/_common/identityImage?id=d6245f20-2af8-44f4-9451-8107cb2767db"
			},
			"creationDate": "2016-11-01T16:30:31.6655471Z",
			"title": "A new feature",
			"description": "Adding a new feature",
			"sourceRefName": "refs/heads/npaulk/my_work",
			"targetRefName": "refs/heads/new_feature",
			"mergeStatus": "succeeded",
			"mergeId": "f5fc8381-3fb2-49fe-8a0d-27dcc2d6ef82",
			"lastMergeSourceCommit": {
			  "commitId": "b60280bc6e62e2f880f1b63c1e24987664d3bda3",
			  "url": "https://fabrikam.visualstudio.com/_apis/git/repositories/3411ebc1-d5aa-464f-9615-0b527bc66719/commits/b60280bc6e62e2f880f1b63c1e24987664d3bda3"
			},
			"lastMergeTargetCommit": {
			  "commitId": "f47bbc106853afe3c1b07a81754bce5f4b8dbf62",
			  "url": "https://fabrikam.visualstudio.com/_apis/git/repositories/3411ebc1-d5aa-464f-9615-0b527bc66719/commits/f47bbc106853afe3c1b07a81754bce5f4b8dbf62"
			},
			"lastMergeCommit": {
			  "commitId": "39f52d24533cc712fc845ed9fd1b6c06b3942588",
			  "url": "https://fabrikam.visualstudio.com/_apis/git/repositories/3411ebc1-d5aa-464f-9615-0b527bc66719/commits/39f52d24533cc712fc845ed9fd1b6c06b3942588"
			},
			"reviewers": [
			  {
				"reviewerUrl": "https://fabrikam.visualstudio.com/_apis/git/repositories/3411ebc1-d5aa-464f-9615-0b527bc66719/pullRequests/22/reviewers/d6245f20-2af8-44f4-9451-8107cb2767db",
				"vote": 0,
				"id": "d6245f20-2af8-44f4-9451-8107cb2767db",
				"displayName": "Normal Paulk",
				"uniqueName": "fabrikamfiber16@hotmail.com",
				"url": "https://fabrikam.visualstudio.com/_apis/Identities/d6245f20-2af8-44f4-9451-8107cb2767db",
				"imageUrl": "https://fabrikam.visualstudio.com/_api/_common/identityImage?id=d6245f20-2af8-44f4-9451-8107cb2767db"
			  }
			],
			"url": "https://fabrikam.visualstudio.com/_apis/git/repositories/3411ebc1-d5aa-464f-9615-0b527bc66719/pullRequests/22",
			"supportsIterations": true
		  }
		],
		"count": 1
	}`
)

func TestPullRequestsService_List(t *testing.T) {
	c, mux, _, teardown := setup()
	defer teardown()
	mux.HandleFunc("/o/p/_apis/git/pullrequests", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testFormValues(t, r, values{
			"searchCriteria.status":        "active",
			"searchCriteria.sourceRefName": "h",
			"searchCriteria.targetRefName": "b",
		})
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"count": 1,
			"value": [{
				"pullRequestId": 22
			}]
			}`)
	})

	opt := &azuredevops.PullRequestListOptions{
		Status:        "active",
		SourceRefName: "h",
		TargetRefName: "b",
	}

	got, _, err := c.PullRequests.List(context.Background(), "o", "p", opt)
	if err != nil {
		t.Errorf("PullRequests.List returned error: %v", err)
	}

	want := []*azuredevops.GitPullRequest{{PullRequestID: Int(22)}}
	if !cmp.Equal(got, want) {
		diff := cmp.Diff(got, want)
		fmt.Printf(diff)
		t.Errorf("PullRequests.List returned %+v, want %+v", got, want)
	}
}

func TestPullRequestsService_ListCommits(t *testing.T) {
	c, mux, _, teardown := setup()
	defer teardown()
	mux.HandleFunc("/o/p/_apis/git/repositories/r/pullrequests/22/commits", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"count": 1,
			"value": [{
				"commitId": "6ebe65c46761913eaf81476dc10aa6a743fb99a0",
				"comment": "COMMENT"
			}]
			}`)
	})

	got, _, err := c.PullRequests.ListCommits(context.Background(), "o", "p", "r", 22)
	if err != nil {
		t.Errorf("PullRequests.ListCommits returned error: %v", err)
	}

	want := []*azuredevops.GitCommitRef{{
		CommitID: String("6ebe65c46761913eaf81476dc10aa6a743fb99a0"),
		Comment:  String("COMMENT"),
	}}
	if !cmp.Equal(got, want) {
		diff := cmp.Diff(got, want)
		fmt.Printf(diff)
		t.Errorf("PullRequests.ListCommits returned %+v, want %+v", got, want)
	}
}

func TestPullRequestsService_Get(t *testing.T) {
	c, mux, _, teardown := setup()
	defer teardown()
	mux.HandleFunc("/o/p/_apis/git/pullrequests/22", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"pullRequestId": 22
		}`)
	})

	opts := &azuredevops.PullRequestListOptions{}
	got, _, err := c.PullRequests.Get(context.Background(), "o", "p", 22, opts)
	if err != nil {
		t.Errorf("PullRequests.Get returned error: %v", err)
	}

	want := &azuredevops.GitPullRequest{
		PullRequestID: Int(22),
	}
	if !cmp.Equal(got, want) {
		diff := cmp.Diff(got, want)
		fmt.Printf(diff)
		t.Errorf("PullRequests.Get returned %+v, want %+v", got, want)
	}
}

func TestPullRequestsService_Merge(t *testing.T) {
	c, mux, _, teardown := setup()
	defer teardown()
	mux.HandleFunc("/o/p/_apis/git/repositories/r/pullrequests/22", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PATCH")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"status": "completed",
			"autoCompleteSetBy": {
				"id": "54d125f7-69f7-4191-904f-c5b96b6261c8",
				"displayName": "Jamal Hartnett",
				"uniqueName": "fabrikamfiber4@hotmail.com",
				"url": "https://vssps.dev.azure.com/fabrikam/_apis/Identities/54d125f7-69f7-4191-904f-c5b96b6261c8",
				"imageUrl": "https://dev.azure.com/fabrikam/DefaultCollection/_api/_common/identityImage?id=54d125f7-69f7-4191-904f-c5b96b6261c8"
			},
			"pullRequestId": 22,
			"completionOptions": {
				"bypassPolicy":false,
				"bypassReason":"",
				"deleteSourceBranch":false,
				"mergeCommitMessage":"TEST MERGE COMMIT MESSAGE",
				"mergeStrategy":"noFastForward",
				"squashMerge":false,
				"transitionWorkItems":true,
				"triggeredByAutoComplete":false
			}
		}`)
	})

	// Set default pull request completion options
	empty := ""
	mcm := azuredevops.NoFastForward.String()
	twi := new(bool)
	*twi = true
	completionOpts := azuredevops.GitPullRequestCompletionOptions{
		BypassPolicy:            new(bool),
		BypassReason:            &empty,
		DeleteSourceBranch:      new(bool),
		MergeCommitMessage:      String("TEST MERGE COMMIT MESSAGE"),
		MergeStrategy:           &mcm,
		SquashMerge:             new(bool),
		TransitionWorkItems:     twi,
		TriggeredByAutoComplete: new(bool),
	}

	id := azuredevops.IdentityRef{
		ID:          String("54d125f7-69f7-4191-904f-c5b96b6261c8"),
		DisplayName: String("Jamal Hartnett"),
		UniqueName:  String("fabrikamfiber4@hotmail.com"),
		URL:         String("https://vssps.dev.azure.com/fabrikam/_apis/Identities/54d125f7-69f7-4191-904f-c5b96b6261c8"),
		ImageURL:    String("https://dev.azure.com/fabrikam/DefaultCollection/_api/_common/identityImage?id=54d125f7-69f7-4191-904f-c5b96b6261c8"),
	}

	got, _, err := c.PullRequests.Merge(context.Background(), "o", "p", "r", 22, nil, completionOpts, id)
	if err != nil {
		t.Errorf("PullRequests.Merge returned error: %v", err)
	}

	want := &azuredevops.GitPullRequest{
		Status:            String("completed"),
		PullRequestID:     Int(22),
		CompletionOptions: &completionOpts,
		AutoCompleteSetBy: &id,
	}
	if !cmp.Equal(got, want) {
		diff := cmp.Diff(got, want)
		t.Errorf("PullRequests.Merge error: %s", diff)
	}
}

func TestPullRequestsService_Create(t *testing.T) {
	c, mux, _, teardown := setup()
	defer teardown()
	mux.HandleFunc("/o/p/_apis/git/repositories/r/pullrequests", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		w.Header().Set("Content-Type", "application/json")
		b, _ := ioutil.ReadAll(r.Body)
		parsed := &azuredevops.GitPullRequest{}
		json.Unmarshal(b, parsed)
		if parsed.GetSourceRefName() != "refs/heads/mytopic" {
			t.Errorf("GetSourceRefName error: %v", parsed.GetSourceRefName())
		}
		if parsed.GetTargetRefName() != "refs/heads/master" {
			t.Errorf("GetTargetRefName returned error: %v", parsed.GetTargetRefName())
		}
		fmt.Fprint(w, `{
			"pullRequestId": 10,
			"title": "TEST PULL REQUEST TITLE",
			"description": "TEST PULL REQUEST DESCRIPTION",
			"sourceRefName": "refs/heads/mytopic",
			"targetRefName": "refs/heads/master",
			"mergeStatus": "succeeded"
		}`)
	})

	pull := &azuredevops.GitPullRequest{
		Title:         String("TEST PULL REQUEST TITLE"),
		Description:   String("TEST PULL REQUEST DESCRIPTION"),
		SourceRefName: String("mytopic"),
		TargetRefName: String("refs/heads/master"),
	}

	got, _, err := c.PullRequests.Create(context.Background(), "o", "p", "r", pull)
	if err != nil {
		t.Errorf("PullRequests.Create returned error: %v", err)
	}

	want := &azuredevops.GitPullRequest{
		PullRequestID: Int(10),
		Title:         String("TEST PULL REQUEST TITLE"),
		Description:   String("TEST PULL REQUEST DESCRIPTION"),
		SourceRefName: String("refs/heads/mytopic"),
		TargetRefName: String("refs/heads/master"),
		MergeStatus:   String("succeeded"),
	}
	if !cmp.Equal(got, want) {
		diff := cmp.Diff(got, want)
		fmt.Printf(diff)
		t.Errorf("PullRequests.Create returned %+v, want %+v", got, want)
	}
}

func TestPullRequestsService_GetWithRepo(t *testing.T) {
	c, mux, _, teardown := setup()
	defer teardown()
	mux.HandleFunc("/o/p/_apis/git/repositories/r/pullrequests/22", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"pullRequestId": 22
		}`)
	})

	opts := &azuredevops.PullRequestGetOptions{
		RepositoryID:        "",
		IncludeWorkItemRefs: true,
	}
	got, _, err := c.PullRequests.GetWithRepo(context.Background(), "o", "p", "r", 22, opts)
	if err != nil {
		t.Errorf("PullRequests.Get returned error: %v", err)
	}

	want := &azuredevops.GitPullRequest{
		PullRequestID: Int(22),
	}
	if !cmp.Equal(got, want) {
		diff := cmp.Diff(got, want)
		fmt.Printf(diff)
		t.Errorf("PullRequests.Get returned %+v, want %+v", got, want)
	}
}

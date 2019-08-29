package vcs_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mcdafydd/go-azuredevops/azuredevops"
	"github.com/runatlantis/atlantis/server/events/models"
	"github.com/runatlantis/atlantis/server/events/vcs"
	"github.com/runatlantis/atlantis/server/events/vcs/fixtures"
	. "github.com/runatlantis/atlantis/testing"
)

func TestAzureDevopsClient_MergePull(t *testing.T) {
	cases := []struct {
		description string
		response    string
		code        int
		expErr      string
	}{
		{
			"success",
			adMergeSuccess,
			200,
			"",
		},
		{
			"405",
			`{"message":"405 Method Not Allowed"}`,
			405,
			"405 {message: 405 Method Not Allowed}",
		},
		{
			"406",
			`{"message":"406 Branch cannot be merged"}`,
			406,
			"406 {message: 406 Branch cannot be merged}",
		},
	}

	// Set default pull request completion options
	mcm := azuredevops.NoFastForward.String()
	twi := new(bool)
	*twi = true
	completionOptions := azuredevops.GitPullRequestCompletionOptions{
		BypassPolicy:            new(bool),
		BypassReason:            azuredevops.String(""),
		DeleteSourceBranch:      new(bool),
		MergeCommitMessage:      azuredevops.String("commit message"),
		MergeStrategy:           &mcm,
		SquashMerge:             new(bool),
		TransitionWorkItems:     twi,
		TriggeredByAutoComplete: new(bool),
	}

	id := azuredevops.IdentityRef{}
	pull := azuredevops.GitPullRequest{
		PullRequestID: azuredevops.Int(22),
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			testServer := httptest.NewTLSServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.RequestURI {
					// The first request should hit this URL.
					case "/owner/project/_apis/git/repositories/repo/pullrequests/22?api-version=5.1-preview.1":
						w.WriteHeader(c.code)
						w.Write([]byte(c.response)) // nolint: errcheck
					default:
						t.Errorf("got unexpected request at %q", r.RequestURI)
						http.Error(w, "not found", http.StatusNotFound)
					}
				}))

			testServerURL, err := url.Parse(testServer.URL)
			Ok(t, err)
			client, err := vcs.NewAzureDevopsClient(testServerURL.Host, "owner", "user", "project", "token")
			Ok(t, err)
			defer disableSSLVerification()()

			merge, _, err := client.Client.PullRequests.Merge(context.Background(),
				"owner",
				"project",
				"repo",
				pull.GetPullRequestID(),
				&pull,
				completionOptions,
				id,
			)

			if err != nil {
				fmt.Printf("Merge failed: %+v\n", err)
				return
			}
			fmt.Printf("Successfully merged pull request: %+v\n", merge)

			err = client.MergePull(models.PullRequest{
				Num: 22,
				BaseRepo: models.Repo{
					FullName: "owner/project/repo",
					Owner:    "owner",
					Name:     "repo",
				},
			})
			if c.expErr == "" {
				Ok(t, err)
			} else {
				ErrContains(t, c.expErr, err)
				ErrContains(t, "unable to merge merge request, it may not be in a mergeable state", err)
			}
		})
	}
}

func TestAzureDevopsClient_UpdateStatus(t *testing.T) {
	cases := []struct {
		status   models.CommitStatus
		expState string
	}{
		{
			models.PendingCommitStatus,
			"pending",
		},
		{
			models.SuccessCommitStatus,
			"succeeded",
		},
		{
			models.FailedCommitStatus,
			"failed",
		},
	}
	response := `{"context":{"genre":"Atlantis Bot","name":"src"},"description":"description","state":"%s","targetUrl":"https://google.com"}
`
	for _, c := range cases {
		t.Run(c.expState, func(t *testing.T) {
			gotRequest := false
			testServer := httptest.NewTLSServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.RequestURI {
					case "/owner/project/_apis/git/repositories/repo/commits/sha/statuses?api-version=5.1-preview.1":
						gotRequest = true
						body, err := ioutil.ReadAll(r.Body)
						Ok(t, err)
						exp := fmt.Sprintf(response, c.expState)
						Equals(t, exp, string(body))
						defer r.Body.Close()      // nolint: errcheck
						w.Write([]byte(response)) // nolint: errcheck
					default:
						t.Errorf("got unexpected request at %q", r.RequestURI)
						http.Error(w, "not found", http.StatusNotFound)
					}
				}))

			testServerURL, err := url.Parse(testServer.URL)
			Ok(t, err)
			client, err := vcs.NewAzureDevopsClient(testServerURL.Host, "owner", "user", "project", "token")
			Ok(t, err)
			defer disableSSLVerification()()

			repo := models.Repo{
				FullName: "owner/project/repo",
				Owner:    "owner",
				Name:     "repo",
			}
			err = client.UpdateStatus(repo, models.PullRequest{
				Num:        22,
				BaseRepo:   repo,
				HeadCommit: "sha",
			}, c.status, "src", "description", "https://google.com")
			Ok(t, err)
			Assert(t, gotRequest, "expected to get the request")
		})
	}
}

// GetModifiedFiles should make multiple requests if more than one page
// and concat results.
func TestAzureDevopsClient_GetModifiedFiles(t *testing.T) {
	itemRespTemplate := `{
		"changes": [
	{
		"item": {
			"gitObjectType": "blob",
			"path": "%s",
			"url": "https://dev.azure.com/fabrikam/_apis/git/repositories/278d5cd2-584d-4b63-824a-2ba458937249/items/MyWebSite/MyWebSite/%s?versionType=Commit"
		},
		"changeType": "add"
	},
	{
		"item": {
			"gitObjectType": "blob",
			"path": "%s",
			"url": "https://dev.azure.com/fabrikam/_apis/git/repositories/278d5cd2-584d-4b63-824a-2ba458937249/items/MyWebSite/MyWebSite/%s?versionType=Commit"
		},
		"changeType": "add"
	}
]}`
	resp := fmt.Sprintf(itemRespTemplate, "file1.txt", "file1.txt", "file2.txt", "file2.txt")
	testServer := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.RequestURI {
			// The first request should hit this URL.
			case "/owner/project/_apis/git/repositories/repo/pullrequests/1?api-version=5.1-preview.1&includeWorkItemRefs=true":
				w.Write([]byte(fixtures.ADPullJSON)) // nolint: errcheck
			// The second should hit this URL.
			case "/owner/project/_apis/git/repositories/repo/commits/b60280bc6e62e2f880f1b63c1e24987664d3bda3/changes?api-version=5.1-preview.1":
				// We write a header that means there's an additional page.
				w.Write([]byte(resp)) // nolint: errcheck
				return
			default:
				t.Errorf("got unexpected request at %q", r.RequestURI)
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
		}))

	testServerURL, err := url.Parse(testServer.URL)
	Ok(t, err)
	client, err := vcs.NewAzureDevopsClient(testServerURL.Host, "owner", "user", "project", "token")
	Ok(t, err)
	defer disableSSLVerification()()

	files, err := client.GetModifiedFiles(models.Repo{
		FullName:          "owner/project/repo",
		Owner:             "owner",
		Name:              "repo",
		CloneURL:          "",
		SanitizedCloneURL: "",
		VCSHost: models.VCSHost{
			Type:     models.AzureDevops,
			Hostname: "dev.azure.com",
		},
	}, models.PullRequest{
		Num: 1,
	})
	Ok(t, err)
	Equals(t, []string{"file1.txt", "file2.txt"}, files)
}

func TestAzureDevopsClient_PullIsMergeable(t *testing.T) {
	cases := []struct {
		state        string
		expMergeable bool
	}{
		{
			azuredevops.MergeConflicts.String(),
			false,
		},
		{
			azuredevops.MergeRejectedByPolicy.String(),
			false,
		},
		{
			azuredevops.MergeFailure.String(),
			true,
		},
		{
			azuredevops.MergeNotSet.String(),
			true,
		},
		{
			azuredevops.MergeQueued.String(),
			true,
		},
		{
			azuredevops.MergeSucceeded.String(),
			true,
		},
	}

	// Use a real Azure Devops json response and edit the mergeable_state field.
	jsBytes, err := ioutil.ReadFile("fixtures/azuredevops-pr.json")
	Ok(t, err)
	json := string(jsBytes)

	for _, c := range cases {
		t.Run(c.state, func(t *testing.T) {
			response := strings.Replace(json,
				`"mergeStatus": "NotSet"`,
				fmt.Sprintf(`"mergeStatus": "%s"`, c.state),
				1,
			)

			testServer := httptest.NewTLSServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.RequestURI {
					case "/owner/project/_apis/git/repositories/repo/pullrequests/1?api-version=5.1-preview.1&includeWorkItemRefs=true":
						w.Write([]byte(response)) // nolint: errcheck
						return
					default:
						t.Errorf("got unexpected request at %q", r.RequestURI)
						http.Error(w, "not found", http.StatusNotFound)
						return
					}
				}))
			testServerURL, err := url.Parse(testServer.URL)
			Ok(t, err)
			client, err := vcs.NewAzureDevopsClient(testServerURL.Host, "owner", "user", "project", "token")
			Ok(t, err)
			defer disableSSLVerification()()

			actMergeable, err := client.PullIsMergeable(models.Repo{
				FullName:          "owner/project/repo",
				Owner:             "owner",
				Name:              "repo",
				CloneURL:          "",
				SanitizedCloneURL: "",
				VCSHost: models.VCSHost{
					Type:     models.AzureDevops,
					Hostname: "dev.azure.com",
				},
			}, models.PullRequest{
				Num: 1,
			})
			Ok(t, err)
			Equals(t, c.expMergeable, actMergeable)
		})
	}
}

func TestAzureDevopsClient_PullIsApproved(t *testing.T) {
	cases := []struct {
		testName    string
		vote        int
		expApproved bool
	}{
		{
			"approved",
			azuredevops.VoteApproved,
			true,
		},
		{
			"approved with suggestions",
			azuredevops.VoteApprovedWithSuggestions,
			false,
		},
		{
			"no vote",
			azuredevops.VoteNone,
			false,
		},
		{
			"vote waiting for author",
			azuredevops.VoteWaitingForAuthor,
			false,
		},
		{
			"vote rejected",
			azuredevops.VoteRejected,
			false,
		},
	}

	// Use a real Azure Devops json response and edit the mergeable_state field.
	jsBytes, err := ioutil.ReadFile("fixtures/azuredevops-pr.json")
	Ok(t, err)
	json := string(jsBytes)

	for _, c := range cases {
		t.Run(c.testName, func(t *testing.T) {
			response := strings.Replace(json,
				`"vote": 0,`,
				fmt.Sprintf(`"vote": %d,`, c.vote),
				1,
			)

			testServer := httptest.NewTLSServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.RequestURI {
					case "/owner/project/_apis/git/repositories/repo/pullrequests/1?api-version=5.1-preview.1&includeWorkItemRefs=true":
						w.Write([]byte(response)) // nolint: errcheck
						return
					default:
						t.Errorf("got unexpected request at %q", r.RequestURI)
						http.Error(w, "not found", http.StatusNotFound)
						return
					}
				}))
			testServerURL, err := url.Parse(testServer.URL)
			Ok(t, err)
			client, err := vcs.NewAzureDevopsClient(testServerURL.Host, "owner", "user", "project", "token")
			Ok(t, err)
			defer disableSSLVerification()()

			actApproved, err := client.PullIsApproved(models.Repo{
				FullName:          "owner/project/repo",
				Owner:             "owner",
				Name:              "repo",
				CloneURL:          "",
				SanitizedCloneURL: "",
				VCSHost: models.VCSHost{
					Type:     models.AzureDevops,
					Hostname: "dev.azure.com",
				},
			}, models.PullRequest{
				Num: 1,
			})
			Ok(t, err)
			Equals(t, c.expApproved, actApproved)
		})
	}
}

func TestAzureDevopsClient_GetPullRequest(t *testing.T) {
	// Use a real Azure Devops json response and edit the mergeable_state field.
	jsBytes, err := ioutil.ReadFile("fixtures/azuredevops-pr.json")
	Ok(t, err)
	response := string(jsBytes)

	t.Run("get pull request", func(t *testing.T) {
		testServer := httptest.NewTLSServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.RequestURI {
				case "/owner/project/_apis/git/repositories/repo/pullrequests/1?api-version=5.1-preview.1&includeWorkItemRefs=true":
					w.Write([]byte(response)) // nolint: errcheck
					return
				default:
					t.Errorf("got unexpected request at %q", r.RequestURI)
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
			}))
		testServerURL, err := url.Parse(testServer.URL)
		Ok(t, err)
		client, err := vcs.NewAzureDevopsClient(testServerURL.Host, "owner", "user", "project", "token")
		Ok(t, err)
		defer disableSSLVerification()()

		_, err = client.GetPullRequest(models.Repo{
			FullName:          "owner/project/repo",
			Owner:             "owner",
			Name:              "repo",
			CloneURL:          "",
			SanitizedCloneURL: "",
			VCSHost: models.VCSHost{
				Type:     models.AzureDevops,
				Hostname: "dev.azure.com",
			},
		}, 1)
		Ok(t, err)
	})
}

var adMergeSuccess = `{
	"status": "completed",
	"mergeStatus": "succeeded",
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
}`
package azuredevops_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mcdafydd/go-azuredevops/azuredevops"
)

const (
	// Pulled from https://docs.microsoft.com/en-gb/rest/api/vsts/wit/work%20items/list
	getResponse = `{
		"count": 3,
		"value": [
		  {
			"id": 297,
			"rev": 1,
			"fields": {
			  "System.AreaPath": "Fabrikam-Fiber-Git",
			  "System.TeamProject": "Fabrikam-Fiber-Git",
			  "System.IterationPath": "Fabrikam-Fiber-Git",
			  "System.WorkItemType": "Product Backlog Item",
			  "System.State": "New",
			  "System.Reason": "New backlog item",
			  "System.CreatedDate": "2014-12-29T20:49:20.77Z",
			  "System.CreatedBy": "Jamal Hartnett ",
			  "System.ChangedDate": "2014-12-29T20:49:20.77Z",
			  "System.ChangedBy": "Jamal Hartnett ",
			  "System.Title": "Customer can sign in using their Microsoft Account",
			  "Microsoft.VSTS.Scheduling.Effort": 8,
			  "WEF_6CB513B6E70E43499D9FC94E5BBFB784_Kanban.Column": "New",
			  "System.Description": "Our authorization logic needs to allow for users with Microsoft accounts (formerly Live Ids) - http://msdn.microsoft.com/en-us/library/live/hh826547.aspx"
			},
			"url": "https://fabrikam.visualstudio.com/_apis/wit/workItems/297"
		  },
		  {
			"id": 299,
			"rev": 7,
			"fields": {
			  "System.AreaPath": "Fabrikam-Fiber-Git\\Website",
			  "System.TeamProject": "Fabrikam-Fiber-Git",
			  "System.IterationPath": "Fabrikam-Fiber-Git",
			  "System.WorkItemType": "Task",
			  "System.State": "To Do",
			  "System.Reason": "New task",
			  "System.AssignedTo": "Johnnie McLeod ",
			  "System.CreatedDate": "2014-12-29T20:49:21.617Z",
			  "System.CreatedBy": "Jamal Hartnett ",
			  "System.ChangedDate": "2014-12-29T20:49:28.74Z",
			  "System.ChangedBy": "Jamal Hartnett ",
			  "System.Title": "JavaScript implementation for Microsoft Account",
			  "Microsoft.VSTS.Scheduling.RemainingWork": 4,
			  "System.Description": "Follow the code samples from MSDN",
			  "System.Tags": "Tag1; Tag2"
			},
			"url": "https://fabrikam.visualstudio.com/_apis/wit/workItems/299"
		  },
		  {
			"id": 300,
			"rev": 1,
			"fields": {
			  "System.AreaPath": "Fabrikam-Fiber-Git",
			  "System.TeamProject": "Fabrikam-Fiber-Git",
			  "System.IterationPath": "Fabrikam-Fiber-Git",
			  "System.WorkItemType": "Task",
			  "System.State": "To Do",
			  "System.Reason": "New task",
			  "System.CreatedDate": "2014-12-29T20:49:22.103Z",
			  "System.CreatedBy": "Jamal Hartnett ",
			  "System.ChangedDate": "2014-12-29T20:49:22.103Z",
			  "System.ChangedBy": "Jamal Hartnett ",
			  "System.Title": "Unit Testing for MSA login",
			  "Microsoft.VSTS.Scheduling.RemainingWork": 3,
			  "System.Description": "We need to ensure we have coverage to prevent regressions"
			},
			"url": "https://fabrikam.visualstudio.com/_apis/wit/workItems/300"
		  }
		]
	}
	`
	// Pulled from https://docs.microsoft.com/en-gb/rest/api/vsts/work/iterations/get%20iteration%20work%20items
	getIdsURL      = "/o/p/t/_apis/work/teamsettings/iterations/a589a806-bf11-4d4f-a031-c19813331553/workitems"
	getIdsResponse = `{
		"workItemRelations": [
		  {
			"rel": null,
			"source": null,
			"target": {
			  "id": 1,
			  "url": "https://fabrikam.visualstudio.com/_apis/wit/workItems/1"
			}
		  },
		  {
			"rel": "System.LinkTypes.Hierarchy-Forward",
			"source": {
			  "id": 1,
			  "url": "https://fabrikam.visualstudio.com/_apis/wit/workItems/1"
			},
			"target": {
			  "id": 3,
			  "url": "https://fabrikam.visualstudio.com/_apis/wit/workItems/3"
			}
		  }
		],
		"url": "https://fabrikam.visualstudio.com/Fabrikam-Fiber/_apis/work/teamsettings/iterations/a589a806-bf11-4d4f-a031-c19813331553/workitems",
		"_links": {
		  "self": {
			"href": "https://fabrikam.visualstudio.com/Fabrikam-Fiber/_apis/work/teamsettings/iterations/a589a806-bf11-4d4f-a031-c19813331553/workitems"
		  },
		  "iteration": {
			"href": "https://fabrikam.visualstudio.com/Fabrikam-Fiber/_apis/work/teamsettings/iterations/a589a806-bf11-4d4f-a031-c19813331553"
		  }
		}
	}`

	commentResponse = `{
		"workItemId" : 1,
		"text" : "TEST COMMENT",
		"version" : 1,
		"id" : 4222704,
		"createdDate" : "0001-01-01 00:00:00 +0000 UTC",
		"modifiedDate" : "0001-01-01 00:00:00 +0000 UTC"
 }`
)

func TestWorkItems_GetForIteration(t *testing.T) {
	actualIdsURL := fmt.Sprintf("/o/p/t/_apis/work/teamsettings/iterations/a589a806-bf11-4d4f-a031-c19813331553/workitems?api-version=5.1-preview.1")
	actualGetURL := fmt.Sprintf("/o/p/_apis/wit/workitems?ids=1,3&fields=System.Id,System.Title,System.State,System.WorkItemType,Microsoft.VSTS.Scheduling.StoryPoints,System.BoardColumn,System.CreatedBy,System.AssignedTo,System.Tags&api-version=5.1-preview.1")

	tt := []struct {
		name              string
		idsBaseURL        string
		getBaseURL        string
		actualIdsURL      string
		actualGetURL      string
		idsResponse       string
		getResponse       string
		expectedWorkItems int
		tagString         string
	}{
		{
			name:              "we get ids and we get iterations",
			idsBaseURL:        getIdsURL,
			getBaseURL:        "/o/p/_apis/wit/workitems",
			actualIdsURL:      actualIdsURL,
			actualGetURL:      actualGetURL,
			idsResponse:       getIdsResponse,
			getResponse:       getResponse,
			expectedWorkItems: 3,
			tagString:         "Tag1; Tag2",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			c, mux, _, teardown := setup()
			defer teardown()

			mux.HandleFunc(tc.idsBaseURL, func(w http.ResponseWriter, r *http.Request) {
				testMethod(t, r, "GET")
				testURL(t, r, tc.actualIdsURL)
				json := tc.idsResponse
				fmt.Fprint(w, json)
			})
			mux.HandleFunc(tc.getBaseURL, func(w http.ResponseWriter, r *http.Request) {
				testMethod(t, r, "GET")
				testURL(t, r, tc.actualGetURL)
				json := tc.getResponse
				fmt.Fprint(w, json)
			})

			iteration := azuredevops.Iteration{ID: String("a589a806-bf11-4d4f-a031-c19813331553")}
			got, _, err := c.WorkItems.GetForIteration(context.Background(), "o", "p", "t", iteration)
			if err != nil {
				t.Fatalf("returned error: %v", err)
			}

			if len(got) != tc.expectedWorkItems {
				t.Fatalf("expected %d work items; got %d", tc.expectedWorkItems, len(got))
			}
		})
	}
}

func TestWorkItems_GetComment(t *testing.T) {
	c, mux, _, teardown := setup()
	defer teardown()

	comment := "TEST COMMENT"
	want := &azuredevops.WorkItemComment{ID: Int(1), Text: String(comment)}

	mux.HandleFunc("/o/p/_apis/wit/workItems/1/comments/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")

		fmt.Fprint(w, `{"id":1, "text": "TEST COMMENT"}`)
	})

	opts := azuredevops.WorkItemCommentListOptions{}
	got, _, err := c.WorkItems.GetComment(context.Background(), "o", "p", 1, 1, &opts)
	if err != nil {
		t.Errorf("WorkItems.GetComment returned error: %v", err)
	}

	if !cmp.Equal(got, want) {
		diff := cmp.Diff(got, want)
		t.Errorf("WorkItems.GetComment error: %s", diff)
	}
}

func TestWorkItems_ListComments(t *testing.T) {
	c, mux, _, teardown := setup()
	defer teardown()

	commentText := "TEST COMMENT"
	comment := &azuredevops.WorkItemComment{ID: Int(1), Text: &commentText}
	comments := []*azuredevops.WorkItemComment{comment}
	want := &azuredevops.WorkItemCommentList{
		TotalCount: Int(1),
		Count:      Int(1),
		Comments:   comments,
	}

	mux.HandleFunc("/o/p/_apis/wit/workItems/1/comments", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")

		fmt.Fprint(w, `{
			"totalCount": 1,
			"count": 1,
			"comments": [{
				"id": 1,
				"text": "TEST COMMENT"
			}]
		}`)
	})

	opts := azuredevops.WorkItemCommentListOptions{
		IDs: []int{1, 2, 3},
	}
	got, _, err := c.WorkItems.ListComments(context.Background(), "o", "p", 1, &opts)
	if err != nil {
		t.Errorf("WorkItems.ListComments returned error: %v", err)
	}

	if !cmp.Equal(got, want) {
		diff := cmp.Diff(got, want)
		t.Errorf("WorkItems.ListComments error: %s", diff)
	}
}

func TestWorkItems_CreateComment(t *testing.T) {
	c, mux, _, teardown := setup()
	defer teardown()

	comment := "TEST COMMENT"
	id := 1
	want := &azuredevops.WorkItemComment{ID: &id, Text: &comment}

	mux.HandleFunc("/o/p/_apis/wit/workItems/1/comments", func(w http.ResponseWriter, r *http.Request) {
		v := new(azuredevops.WorkItemComment)
		json.NewDecoder(r.Body).Decode(v)

		testMethod(t, r, "POST")

		fmt.Fprint(w, `{"id":1, "text": "TEST COMMENT"}`)
	})

	got, _, err := c.WorkItems.CreateComment(context.Background(), "o", "p", 1, want)
	if err != nil {
		t.Errorf("WorkItems.CreateComment returned error: %v", err)
	}

	if !cmp.Equal(got, want) {
		diff := cmp.Diff(got, want)
		t.Errorf("WorkItems.CreateComment error: %s", diff)
	}
}

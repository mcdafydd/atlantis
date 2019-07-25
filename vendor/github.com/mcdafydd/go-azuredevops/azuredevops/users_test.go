package azuredevops_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mcdafydd/go-azuredevops/azuredevops"
)

func Test_UsersGet(t *testing.T) {
	c, mux, _, teardown := setup()
	defer teardown()
	u, _ := url.Parse("")
	c.VsspsBaseURL = *u
	mux.HandleFunc("/o/_apis/graph/users/descriptor", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
      "subjectKind": "user",
      "metaType": "member",
      "domain": "45aa3d2d-7442-473d-b4d3-3c670da9dd96",
      "principalName": "jmarks@vscsi.us",
      "mailAddress": "jmarks@vscsi.us",
      "origin": "aad"
    }`)
	})

	got, _, err := c.Users.Get(context.Background(), "o", "descriptor")
	if err != nil {
		t.Fatalf("returned error: %v", err)
	}

	graphSubject := azuredevops.GraphSubject{
		SubjectKind: String("user"),
		Origin:      String("aad"),
	}
	graphMember := azuredevops.GraphMember{
		Domain:        String("45aa3d2d-7442-473d-b4d3-3c670da9dd96"),
		PrincipalName: String("jmarks@vscsi.us"),
		MailAddress:   String("jmarks@vscsi.us"),
	}
	graphUser := azuredevops.GraphUser{
		MetaType: String("member"),
	}
	want := &azuredevops.GraphUser{}
	want.MetaType = graphUser.MetaType
	want.Domain = graphMember.Domain
	want.PrincipalName = graphMember.PrincipalName
	want.MailAddress = graphMember.MailAddress
	want.SubjectKind = graphSubject.SubjectKind
	want.Origin = graphSubject.Origin

	if !cmp.Equal(got, want) {
		diff := cmp.Diff(got, want)
		fmt.Printf(diff)
		t.Errorf("Users.Get returned %+v, want %+v", got, want)
	}
}

func Test_UsersList(t *testing.T) {
	c, mux, _, teardown := setup()
	defer teardown()
	u, _ := url.Parse("")
	c.VsspsBaseURL = *u
	mux.HandleFunc("/o/_apis/graph/users", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"count": 2,
		  "value": [{
				"subjectKind": "user",
				"metaType": "member",
				"domain": "45aa3d2d-7442-473d-b4d3-3c670da9dd96",
				"principalName": "jmarks@vscsi.us",
				"mailAddress": "jmarks@vscsi.us",
				"origin": "aad"
			},
			{
				"subjectKind": "user",
				"domain": "Build",
				"principalName": "10feb381-82c3-4902-8e1f-840299a48ae4",
				"mailAddress": "",
				"origin": "vsts"
			}]
		}`)
	})

	got, _, err := c.Users.List(context.Background(), "o")
	if err != nil {
		t.Fatalf("returned error: %v", err)
	}

	graphSubject0 := azuredevops.GraphSubject{
		SubjectKind: String("user"),
		Origin:      String("aad"),
	}
	graphMember0 := azuredevops.GraphMember{
		Domain:        String("45aa3d2d-7442-473d-b4d3-3c670da9dd96"),
		PrincipalName: String("jmarks@vscsi.us"),
		MailAddress:   String("jmarks@vscsi.us"),
	}
	graphUser0 := azuredevops.GraphUser{
		MetaType: String("member"),
	}
	graphSubject1 := azuredevops.GraphSubject{
		SubjectKind: String("user"),
		Origin:      String("vsts"),
	}
	graphMember1 := azuredevops.GraphMember{
		Domain:        String("Build"),
		PrincipalName: String("10feb381-82c3-4902-8e1f-840299a48ae4"),
		MailAddress:   String(""),
	}
	want := []*azuredevops.GraphUser{}
	el := &azuredevops.GraphUser{}
	el.MetaType = graphUser0.MetaType
	el.Domain = graphMember0.Domain
	el.PrincipalName = graphMember0.PrincipalName
	el.MailAddress = graphMember0.MailAddress
	el.SubjectKind = graphSubject0.SubjectKind
	el.Origin = graphSubject0.Origin
	want = append(want, el)
	el = &azuredevops.GraphUser{}
	el.Domain = graphMember1.Domain
	el.PrincipalName = graphMember1.PrincipalName
	el.MailAddress = graphMember1.MailAddress
	el.SubjectKind = graphSubject1.SubjectKind
	el.Origin = graphSubject1.Origin
	want = append(want, el)

	if !cmp.Equal(got, want) {
		diff := cmp.Diff(got, want)
		fmt.Printf(diff)
		t.Errorf("Users.List returned %+v, want %+v", got, want)
	}
}

func Test_UsersGetDescriptors(t *testing.T) {
	c, mux, _, teardown := setup()
	defer teardown()
	u, _ := url.Parse("")
	c.VsspsBaseURL = *u
	mux.HandleFunc("/o/_apis/graph/descriptors/storageKey", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"value": "aad.OWI3MWYyMTYtNGM0Zi03Yjc0LWE5MTEtZWZiMGZhOWM3Nzdm",
		"_links": {
			"self": {
				"href": "https://vssps.dev.azure.com/Fabrikam/_apis/Graph/Descriptors/9b71f216-4c4f-6b74-a911-efb0fa9c777f"
			},
			"storageKey": {
				"href": "https://vssps.dev.azure.com/Fabrikam/_apis/Graph/StorageKeys/aad.OWI3MWYyMTYtNGM0Zi03Yjc0LWE5MTEtZWZiMGZhOWM3Nzdm"
			},
			"subject": {
				"href": "https://vssps.dev.azure.com/Fabrikam/_apis/Graph/Users/aad.OWI3MWYyMTYtNGM0Zi03Yjc0LWE5MTEtZWZiMGZhOWM3Nzdm"
			}
		}}`)
	})

	got, _, err := c.Users.GetDescriptors(context.Background(), "o", "storageKey")
	if err != nil {
		t.Fatalf("returned error: %v", err)
	}

	want := &azuredevops.GraphDescriptorResult{
		Links: map[string]azuredevops.Link{
			"self":       {Href: String("https://vssps.dev.azure.com/Fabrikam/_apis/Graph/Descriptors/9b71f216-4c4f-6b74-a911-efb0fa9c777f")},
			"storageKey": {Href: String("https://vssps.dev.azure.com/Fabrikam/_apis/Graph/StorageKeys/aad.OWI3MWYyMTYtNGM0Zi03Yjc0LWE5MTEtZWZiMGZhOWM3Nzdm")},
			"subject":    {Href: String("https://vssps.dev.azure.com/Fabrikam/_apis/Graph/Users/aad.OWI3MWYyMTYtNGM0Zi03Yjc0LWE5MTEtZWZiMGZhOWM3Nzdm")},
		},
		Value: String("aad.OWI3MWYyMTYtNGM0Zi03Yjc0LWE5MTEtZWZiMGZhOWM3Nzdm"),
	}

	if !cmp.Equal(got, want) {
		diff := cmp.Diff(got, want)
		fmt.Printf(diff)
		t.Errorf("Users.GetDescriptors returned %+v, want %+v", got, want)
	}
}

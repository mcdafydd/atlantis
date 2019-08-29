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

package vcs

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	version "github.com/hashicorp/go-version"
	"github.com/mcdafydd/go-azuredevops/azuredevops"
	"github.com/pkg/errors"
	"github.com/runatlantis/atlantis/server/events/vcs/common"

	"github.com/runatlantis/atlantis/server/events/models"

	"gopkg.in/russross/blackfriday.v2"
)

// AzureDevopsClient represents an Azure Devops VCS client
type AzureDevopsClient struct {
	Client *azuredevops.Client
	// Version is set to the server version.
	Version *version.Version
	ctx     context.Context
}

// NewAzureDevopsClient returns a valid Azure Devops client.
func NewAzureDevopsClient(hostname string, org string, username string, project string, token string) (*AzureDevopsClient, error) {
	tp := azuredevops.BasicAuthTransport{
		Username: "",
		Password: strings.TrimSpace(token),
	}
	httpClient := tp.Client()
	httpClient.Timeout = time.Second * 10
	var adClient, err = azuredevops.NewClient(httpClient)
	if err != nil {
		return nil, errors.Wrapf(err, "azuredevops.NewClient() %p", adClient)
	}

	if hostname != "dev.azure.com" {
		baseURL := fmt.Sprintf("https://%s/", hostname)
		base, err := url.Parse(baseURL)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid azure devops hostname trying to parse %s", baseURL)
		}
		adClient.BaseURL = *base
	}

	client := &AzureDevopsClient{
		Client: adClient,
		ctx:    context.Background(),
	}

	return client, nil
}

// GetModifiedFiles returns the names of files that were modified in the merge request.
// The names include the path to the file from the repo root, ex. parent/child/file.txt.
func (g *AzureDevopsClient) GetModifiedFiles(repo models.Repo, pull models.PullRequest) ([]string, error) {
	var files []string

	opts := azuredevops.PullRequestGetOptions{
		IncludeWorkItemRefs: true,
	}
	owner, project, repoName := SplitAzureDevopsRepoFullName(repo.FullName)
	commitIDResponse, _, _ := g.Client.PullRequests.GetWithRepo(g.ctx, owner, project, repoName, pull.Num, &opts)

	commitID := commitIDResponse.GetLastMergeSourceCommit().GetCommitID()

	r, _, _ := g.Client.Git.GetChanges(g.ctx, owner, project, repoName, commitID)

	for _, change := range r.Changes {
		item := change.GetItem()
		files = append(files, item.GetPath())

		// If the file was renamed, we'll want to run plan in the directory
		// it was moved from as well.
		changeType := azuredevops.Rename.String()
		if change.ChangeType == &changeType {
			files = append(files, change.GetSourceServerItem())
		}
	}

	return files, nil
}

// CreateComment creates a comment on a work item linked to pullNum.
// Comments made on pull requests do not have an associated webhook event
// trigger, only comments made on work items (user stories, tasks, etc.).
// If pull request is linked to multiple work items, log an error and ignore.
//
// If comment length is greater than the max comment length we split into
// multiple comments.
// Azure Devops doesn't support markdown in Work Item comments, but it will
// convert text to HTML.  We use the blackfriday library to convert Atlantis
// comment markdown before submission.
func (g *AzureDevopsClient) CreateComment(repo models.Repo, pullNum int, comment string) (err error) {
	sepEnd := "\n```\n</details>" +
		"\n<br>\n\n**Warning**: Output length greater than max comment size. Continued in next comment."
	sepStart := "Continued from previous comment.\n<details><summary>Show Output</summary>\n\n" +
		"```diff\n"

	// maxCommentLength is the maximum number of chars allowed in a single comment
	// This length was copied from the Github client - haven't found documentation
	// or tested limit in Azure Devops.
	const maxCommentLength = 65536

	comments := common.SplitComment(comment, maxCommentLength, sepEnd, sepStart)
	opts := azuredevops.PullRequestGetOptions{
		IncludeWorkItemRefs: true,
	}
	owner, project, repoName := SplitAzureDevopsRepoFullName(repo.FullName)
	pull, _, err := g.Client.PullRequests.GetWithRepo(g.ctx, owner, project, repoName, pullNum, &opts)
	if err != nil {
		return err
	}
	if len(pull.WorkItemRefs) == 1 {
		workItemID, err := strconv.Atoi(*pull.WorkItemRefs[0].ID)
		if err != nil {
			return err
		}
		for _, c := range comments {
			input := blackfriday.Run([]byte(c))
			s := string(input)
			workItemComment := azuredevops.WorkItemComment{
				Text: &s,
			}
			owner, project, _ := SplitAzureDevopsRepoFullName(repo.FullName)
			_, _, err := g.Client.WorkItems.CreateComment(g.ctx, owner, project, workItemID, &workItemComment)
			if err != nil {
				return err
			}
		}
	} else if len(pull.WorkItemRefs) > 1 {
		return errors.New("pull request linked to more than one work item - ignoring")
	}
	return
}

// PullIsApproved returns true if the merge request was approved.
// https://docs.microsoft.com/en-us/azure/devops/repos/git/branch-policies?view=azure-devops#require-a-minimum-number-of-reviewers
func (g *AzureDevopsClient) PullIsApproved(repo models.Repo, pull models.PullRequest) (bool, error) {
	opts := azuredevops.PullRequestGetOptions{
		IncludeWorkItemRefs: true,
	}
	owner, project, repoName := SplitAzureDevopsRepoFullName(repo.FullName)
	adPull, _, err := g.Client.PullRequests.GetWithRepo(g.ctx, owner, project, repoName, pull.Num, &opts)
	if err != nil {
		return false, errors.Wrap(err, "getting pull request")
	}
	for _, review := range adPull.Reviewers {
		if review == nil {
			continue
		}
		if *review.Vote != azuredevops.VoteApproved {
			return false, err
		}
	}

	return true, err
}

// PullIsMergeable returns true if the merge request can be merged.
func (g *AzureDevopsClient) PullIsMergeable(repo models.Repo, pull models.PullRequest) (bool, error) {

	opts := azuredevops.PullRequestGetOptions{
		IncludeWorkItemRefs: true,
	}
	owner, project, repoName := SplitAzureDevopsRepoFullName(repo.FullName)
	adPull, _, err := g.Client.PullRequests.GetWithRepo(g.ctx, owner, project, repoName, pull.Num, &opts)
	if err != nil {
		return false, errors.Wrap(err, "getting pull request")
	}
	if *adPull.MergeStatus != azuredevops.MergeConflicts.String() &&
		*adPull.MergeStatus != azuredevops.MergeRejectedByPolicy.String() {
		return true, nil
	}
	return false, nil
}

// GetPullRequest returns the pull request.
func (g *AzureDevopsClient) GetPullRequest(repo models.Repo, num int) (*azuredevops.GitPullRequest, error) {
	opts := azuredevops.PullRequestGetOptions{
		IncludeWorkItemRefs: true,
	}

	owner, project, repoName := SplitAzureDevopsRepoFullName(repo.FullName)
	pull, _, err := g.Client.PullRequests.GetWithRepo(g.ctx, owner, project, repoName, num, &opts)
	return pull, err
}

// UpdateStatus updates the build status of a commit.
func (g *AzureDevopsClient) UpdateStatus(repo models.Repo, pull models.PullRequest, state models.CommitStatus, src string, description string, url string) error {
	adState := azuredevops.GitError.String()
	switch state {
	case models.PendingCommitStatus:
		adState = azuredevops.GitPending.String()
	case models.SuccessCommitStatus:
		adState = azuredevops.GitSucceeded.String()
	case models.FailedCommitStatus:
		adState = azuredevops.GitFailed.String()
	}

	genreStr := "Atlantis Bot"
	status := azuredevops.GitStatus{
		State:       &adState,
		Description: &description,
		Context: &azuredevops.GitStatusContext{
			Name:  &src,
			Genre: &genreStr,
		},
	}
	if url != "" {
		status.TargetURL = &url
	}
	owner, project, repoName := SplitAzureDevopsRepoFullName(repo.FullName)
	_, _, err := g.Client.Git.CreateStatus(g.ctx, owner, project, repoName, pull.HeadCommit, status)
	return err
}

// MergePull merges the merge request using the default no fast-forward strategy
// If the user has set a branch policy that disallows no fast-forward, the merge will fail
// until we handle branch policies
// https://docs.microsoft.com/en-us/azure/devops/repos/git/branch-policies?view=azure-devops
func (g *AzureDevopsClient) MergePull(pull models.PullRequest) error {

	descriptor := "Atlantis Terraform Pull Request Automation"
	i := "atlantis"
	imageURL := "https://github.com/runatlantis/atlantis/raw/master/runatlantis.io/.vuepress/public/hero.png"
	id := azuredevops.IdentityRef{
		Descriptor: &descriptor,
		ID:         &i,
		ImageURL:   &imageURL,
	}
	// Set default pull request completion options
	mcm := azuredevops.NoFastForward.String()
	twi := new(bool)
	*twi = true
	completionOpts := azuredevops.GitPullRequestCompletionOptions{
		BypassPolicy:            new(bool),
		BypassReason:            azuredevops.String(""),
		DeleteSourceBranch:      new(bool),
		MergeCommitMessage:      azuredevops.String(common.AutomergeCommitMsg),
		MergeStrategy:           &mcm,
		SquashMerge:             new(bool),
		TransitionWorkItems:     twi,
		TriggeredByAutoComplete: new(bool),
	}

	// Construct request body from supplied parameters
	mergePull := new(azuredevops.GitPullRequest)
	mergePull.AutoCompleteSetBy = &id
	mergePull.CompletionOptions = &completionOpts

	owner, project, repoName := SplitAzureDevopsRepoFullName(pull.BaseRepo.FullName)
	mergeResult, _, err := g.Client.PullRequests.Merge(
		g.ctx,
		owner,
		project,
		repoName,
		pull.Num,
		mergePull,
		completionOpts,
		id,
	)
	if err != nil {
		return errors.Wrap(err, "merging pull request")
	}
	if *mergeResult.MergeStatus != azuredevops.MergeSucceeded.String() {
		return fmt.Errorf("could not merge pull request: %s", mergeResult.GetMergeFailureMessage())
	}
	return nil
}

// SplitAzureDevopsRepoFullName splits a repo full name up into its owner,
// repo and project name segments. If the repoFullName is malformed, may
// return empty strings for owner, repo, or project.  Azure Devops uses
// repoFullName format owner/project/repo.
//
// Ex. runatlantis/atlantis => (runatlantis, atlantis)
//     gitlab/subgroup/runatlantis/atlantis => (gitlab/subgroup/runatlantis, atlantis)
//     azuredevops/project/atlantis => (azuredevops, project, atlantis)
func SplitAzureDevopsRepoFullName(repoFullName string) (owner string, project string, repo string) {
	firstSlashIdx := strings.Index(repoFullName, "/")
	lastSlashIdx := strings.LastIndex(repoFullName, "/")
	slashCount := strings.Count(repoFullName, "/")
	if lastSlashIdx == -1 || lastSlashIdx == len(repoFullName)-1 {
		return "", "", ""
	}
	if firstSlashIdx != lastSlashIdx && slashCount == 2 {
		return repoFullName[:firstSlashIdx],
			repoFullName[firstSlashIdx+1 : lastSlashIdx],
			repoFullName[lastSlashIdx+1:]
	}
	return repoFullName[:lastSlashIdx], "", repoFullName[lastSlashIdx+1:]
}
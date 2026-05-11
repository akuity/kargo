package azure

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	adocore "github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	adogit "github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"k8s.io/utils/ptr"

	"github.com/akuity/kargo/pkg/gitprovider"
	"github.com/akuity/kargo/pkg/urls"
)

const ProviderName = "azure"

// validMergeMethods is the set of merge strategies supported by Azure DevOps.
// Azure does not validate this server-side, so we validate client-side.
var validMergeMethods = map[string]struct{}{
	"noFastForward": {},
	"rebase":        {},
	"rebaseMerge":   {},
	"squash":        {},
}

// Azure DevOps URLs can be of two different forms:
//
//   - https://dev.azure.com/org/<project>/_git/<repo>
//   - https://<org>.visualstudio.com/<project>/_git/<repo>
//
// We support both forms.
const (
	legacyHostSuffix = "visualstudio.com"
	modernHostSuffix = "dev.azure.com"
)

var registration = gitprovider.Registration{
	Predicate: func(repoURL string) bool {
		u, err := url.Parse(repoURL)
		if err != nil {
			return false
		}
		return u.Host == modernHostSuffix || strings.HasSuffix(u.Host, legacyHostSuffix)
	},
	NewProvider: func(
		repoURL string,
		opts *gitprovider.Options,
	) (gitprovider.Interface, error) {
		return NewProvider(repoURL, opts)
	},
}

func init() {
	gitprovider.Register(ProviderName, registration)
}

// azureGitClient is the subset of adogit.Client methods used by the provider.
type azureGitClient interface {
	GetRepository(
		context.Context,
		adogit.GetRepositoryArgs,
	) (*adogit.GitRepository, error)
	CreatePullRequest(
		context.Context,
		adogit.CreatePullRequestArgs,
	) (*adogit.GitPullRequest, error)
	GetPullRequest(
		context.Context,
		adogit.GetPullRequestArgs,
	) (*adogit.GitPullRequest, error)
	GetPullRequests(
		context.Context,
		adogit.GetPullRequestsArgs,
	) (*[]adogit.GitPullRequest, error)
	UpdatePullRequest(
		context.Context,
		adogit.UpdatePullRequestArgs,
	) (*adogit.GitPullRequest, error)
}

type provider struct {
	org     string
	project string
	repo    string
	client  azureGitClient
}

// NewProvider returns an Azure DevOps-based implementation of gitprovider.Interface.
func NewProvider(
	repoURL string,
	opts *gitprovider.Options,
) (gitprovider.Interface, error) {
	if opts == nil || opts.Token == "" {
		return nil, fmt.Errorf("token is required for Azure DevOps provider")
	}
	org, project, repo, err := parseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}
	organizationUrl := fmt.Sprintf("https://%s/%s", modernHostSuffix, org)
	client, err := adogit.NewClient(
		// The Azure SDK's NewClient performs a one-time service discovery HTTP call
		// to resolve the git resource area endpoint. Given it's limited use, using
		// background context is preferable here to refactoring all provider
		// registrations to be context-aware.
		context.Background(),
		azuredevops.NewPatConnection(organizationUrl, opts.Token),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating Azure DevOps client: %w", err)
	}

	return &provider{
		org:     org,
		project: project,
		repo:    repo,
		client:  client,
	}, nil
}

// CreatePullRequest implements gitprovider.Interface.
func (p *provider) CreatePullRequest(
	ctx context.Context,
	opts *gitprovider.CreatePullRequestOpts,
) (*gitprovider.PullRequest, error) {
	repository, err := p.client.GetRepository(
		ctx,
		adogit.GetRepositoryArgs{
			Project:      &p.project,
			RepositoryId: &p.repo,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error getting repository %q: %w", p.repo, err)
	}
	repoID := ptr.To(repository.Id.String())
	labels := make([]adocore.WebApiTagDefinition, 0, len(opts.Labels))
	for _, label := range opts.Labels {
		labels = append(labels, adocore.WebApiTagDefinition{Name: &label})
	}
	sourceRefName := ptr.To(fmt.Sprintf("refs/heads/%s", opts.Head))
	targetRefName := ptr.To(fmt.Sprintf("refs/heads/%s", opts.Base))
	adoPR, err := p.client.CreatePullRequest(
		ctx,
		adogit.CreatePullRequestArgs{
			Project:      &p.project,
			RepositoryId: repoID,
			GitPullRequestToCreate: &adogit.GitPullRequest{
				Title:         &opts.Title,
				Description:   &opts.Description,
				Labels:        &labels,
				SourceRefName: sourceRefName,
				TargetRefName: targetRefName,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating pull request from %q to %q: %w", opts.Head, opts.Base, err)
	}
	pr, err := convertADOPullRequest(adoPR)
	if err != nil {
		return nil, fmt.Errorf("error converting pull request %d: %w", adoPR.PullRequestId, err)
	}
	return pr, nil
}

// GetPullRequest implements gitprovider.Interface.
func (p *provider) GetPullRequest(
	ctx context.Context,
	id int64,
) (*gitprovider.PullRequest, error) {
	adoPR, err := p.client.GetPullRequest(
		ctx,
		adogit.GetPullRequestArgs{
			Project:       &p.project,
			RepositoryId:  &p.repo,
			PullRequestId: ptr.To(int(id)),
		},
	)
	if err != nil {
		return nil, err
	}
	pr, err := convertADOPullRequest(adoPR)
	if err != nil {
		return nil, fmt.Errorf("error converting pull request %d: %w", id, err)
	}
	return pr, nil
}

// ListPullRequests implements gitprovider.Interface.
func (p *provider) ListPullRequests(
	ctx context.Context,
	opts *gitprovider.ListPullRequestOptions,
) ([]gitprovider.PullRequest, error) {
	adoPRs, err := p.client.GetPullRequests(
		ctx,
		adogit.GetPullRequestsArgs{
			Project:      &p.project,
			RepositoryId: &p.repo,
			SearchCriteria: &adogit.GitPullRequestSearchCriteria{
				Status:        ptr.To(mapADOPrState(opts.State)),
				SourceRefName: ptr.To(opts.HeadBranch),
				TargetRefName: ptr.To(opts.BaseBranch),
			},
		},
	)
	if err != nil {
		return nil, err
	}

	pts := []gitprovider.PullRequest{}
	for _, adoPR := range *adoPRs {
		pr, err := convertADOPullRequest(&adoPR)
		if err != nil {
			return nil, fmt.Errorf("error converting pull request %d: %w", adoPR.PullRequestId, err)
		}
		pts = append(pts, *pr)
	}
	return pts, nil
}

// MergePullRequest implements gitprovider.Interface.
func (p *provider) MergePullRequest(
	ctx context.Context,
	id int64,
	opts *gitprovider.MergePullRequestOpts,
) (*gitprovider.PullRequest, bool, error) {
	if opts == nil {
		opts = &gitprovider.MergePullRequestOpts{}
	}

	// Get the current PR to check its status and get the last merge source commit
	adoPR, err := p.client.GetPullRequest(
		ctx,
		adogit.GetPullRequestArgs{
			Project:       &p.project,
			RepositoryId:  &p.repo,
			PullRequestId: ptr.To(int(id)),
		},
	)
	if err != nil {
		return nil, false, fmt.Errorf("error getting pull request %d: %w", id, err)
	}
	if adoPR == nil {
		return nil, false, fmt.Errorf("pull request %d not found", id)
	}

	status := ptr.Deref(adoPR.Status, adogit.PullRequestStatusValues.NotSet)
	mergeStatus := ptr.Deref(adoPR.MergeStatus, adogit.PullRequestAsyncStatusValues.NotSet)

	switch status {
	case adogit.PullRequestStatusValues.Completed:
		var pr *gitprovider.PullRequest
		if pr, err = convertADOPullRequest(adoPR); err != nil {
			return nil, false, fmt.Errorf("error converting pull request %d: %w", id, err)
		}
		return pr, true, nil
	case adogit.PullRequestStatusValues.Abandoned:
		return nil, false, fmt.Errorf("pull request %d is abandoned", id)
	case adogit.PullRequestStatusValues.Active:
		// Draft PRs can have a merge status of `succeeded`, but aren't actually
		// mergable, so we explicitly check for draft status.
		if ptr.Deref(adoPR.IsDraft, false) {
			return nil, false, nil
		}
		if mergeStatus != adogit.PullRequestAsyncStatusValues.Succeeded {
			// Not ready to merge yet
			return nil, false, nil
		}
	default:
		return nil, false, nil
	}

	var completionOptions *adogit.GitPullRequestCompletionOptions
	if opts.MergeMethod != "" {
		if _, ok := validMergeMethods[opts.MergeMethod]; !ok {
			return nil, false,
				fmt.Errorf("unsupported merge method %q", opts.MergeMethod)
		}
		completionOptions = &adogit.GitPullRequestCompletionOptions{
			MergeStrategy: ptr.To(adogit.GitPullRequestMergeStrategy(opts.MergeMethod)),
		}
	}
	updatedPR, err := p.client.UpdatePullRequest(
		ctx,
		adogit.UpdatePullRequestArgs{
			Project:       &p.project,
			RepositoryId:  &p.repo,
			PullRequestId: ptr.To(int(id)),
			GitPullRequestToUpdate: &adogit.GitPullRequest{
				Status: ptr.To(adogit.PullRequestStatusValues.Completed),
				// LastMergeSourceCommit ensures merge is based on the exact commit we validated.
				// If the PR was amended between our validation and merge attempt, Azure DevOps
				// will reject the merge operation, preventing race conditions.
				LastMergeSourceCommit: adoPR.LastMergeSourceCommit,
				CompletionOptions:     completionOptions,
			},
		},
	)
	if err != nil {
		return nil, false, fmt.Errorf("error merging pull request %d: %w", id, err)
	}
	if updatedPR == nil {
		return nil, false, fmt.Errorf("unexpected nil response after merging pull request %d", id)
	}

	// Azure DevOps processes merges asynchronously. Poll until the PR reaches
	// Completed status so we can return the merge commit information. This is
	// deliberately a simple polling loop with a fixed number of attempts and
	// short delay between attempts instead of a progressive backoff strategy
	// because, knowing how this code is used, contextually (by the git-merge-pr
	// promotion step), we really don't want this call to block for too long.
	var completedPR *adogit.GitPullRequest
	for range 10 {
		completedPR, err = p.client.GetPullRequest(
			ctx,
			adogit.GetPullRequestArgs{
				Project:       &p.project,
				RepositoryId:  &p.repo,
				PullRequestId: ptr.To(int(id)),
			},
		)
		if err != nil {
			return nil, false,
				fmt.Errorf("error getting pull request %d after merge: %w", id, err)
		}
		if completedPR == nil {
			return nil, false,
				fmt.Errorf("unexpected nil pull request after merge of %d", id)
		}
		if ptr.Deref(completedPR.Status, "") ==
			adogit.PullRequestStatusValues.Completed {
			break
		}
		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	// If the PR hasn't reached Completed after polling, return false without an
	// error so the caller can retry on a future reconciliation — at which point
	// the PR will either be Completed (caught by the early return at the top of
	// this function) or still Active with a fresh state.
	if ptr.Deref(completedPR.Status, "") !=
		adogit.PullRequestStatusValues.Completed {
		return nil, false, nil
	}

	pr, err := convertADOPullRequest(completedPR)
	if err != nil {
		return nil, false, fmt.Errorf("error converting merged pull request %d: %w", id, err)
	}
	return pr, true, nil
}

// GetCommitURL implements gitprovider.Interface.
func (p *provider) GetCommitURL(repoURL string, sha string) (string, error) {
	normalizedURL := urls.NormalizeGit(repoURL)

	parsedURL, err := url.Parse(normalizedURL)
	if err != nil {
		return "", fmt.Errorf("error processing repository URL: %s: %s", repoURL, err)
	}

	formattedPath := strings.TrimPrefix(parsedURL.Path, "/v3")

	commitURL := fmt.Sprintf("https://dev.azure.com%s/commit/%s", formattedPath, sha)

	return commitURL, nil
}

// mapADOPrState maps a gitprovider.PullRequestState to an adogit.PullRequestStatus.
func mapADOPrState(state gitprovider.PullRequestState) adogit.PullRequestStatus {
	switch state {
	case gitprovider.PullRequestStateOpen:
		return adogit.PullRequestStatusValues.Active
	case gitprovider.PullRequestStateClosed:
		return adogit.PullRequestStatusValues.Completed
	}
	return adogit.PullRequestStatusValues.All
}

// convertADOPullRequest converts an adogit.GitPullRequest to a gitprovider.PullRequest.
func convertADOPullRequest(pr *adogit.GitPullRequest) (*gitprovider.PullRequest, error) {
	if pr.LastMergeSourceCommit == nil {
		return nil, fmt.Errorf("no last merge source commit found for pull request %d", ptr.Deref(pr.PullRequestId, 0))
	}
	mergeCommit := ptr.Deref(pr.LastMergeCommit, adogit.GitCommitRef{})
	var webURL string
	if pr.Repository != nil {
		webURL = ptr.Deref(pr.Repository.WebUrl, "")
	}
	return &gitprovider.PullRequest{
		Number: int64(ptr.Deref(pr.PullRequestId, 0)),
		URL: fmt.Sprintf(
			"%s/pullrequest/%d",
			webURL,
			ptr.Deref(pr.PullRequestId, 0),
		),
		Open:           ptr.Deref(pr.Status, "notSet") == "active",
		Merged:         ptr.Deref(pr.Status, "notSet") == "completed",
		MergeCommitSHA: ptr.Deref(mergeCommit.CommitId, ""),
		Object:         pr,
		HeadSHA:        ptr.Deref(pr.LastMergeSourceCommit.CommitId, ""),
	}, nil
}

func parseRepoURL(repoURL string) (string, string, string, error) {
	u, err := url.Parse(urls.NormalizeGit(repoURL))
	if err != nil {
		return "", "", "", fmt.Errorf("error parsing Azure DevOps repository URL %q: %w", repoURL, err)
	}
	if u.Host == modernHostSuffix {
		return parseModernRepoURL(u)
	} else if strings.HasSuffix(u.Host, legacyHostSuffix) {
		return parseLegacyRepoURL(u)
	}
	return "", "", "", fmt.Errorf("unsupported host %q", u.Host)
}

// parseModernRepoURL parses a modern Azure DevOps repository URL.
func parseModernRepoURL(u *url.URL) (string, string, string, error) {
	parts := strings.Split(u.Path, "/")
	if len(parts) != 5 {
		return "", "", "", fmt.Errorf("could not extract repository organization, project, and name from URL %q", u)
	}
	return parts[1], parts[2], parts[4], nil
}

// parseLegacyRepoURL parses a legacy Azure DevOps repository URL.
func parseLegacyRepoURL(u *url.URL) (string, string, string, error) {
	organization := strings.TrimSuffix(u.Host, ".visualstudio.com")
	parts := strings.Split(u.Path, "/")
	if len(parts) != 4 {
		return "", "", "", fmt.Errorf("could not extract repository organization, project, and name from URL %q", u)
	}
	return organization, parts[1], parts[3], nil
}

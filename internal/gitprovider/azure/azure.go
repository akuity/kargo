package azure

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/gitprovider"
	"k8s.io/utils/ptr"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	adogit "github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

const ProviderName = "azure"

// Azure DevOps URLs can be of two different forms:
//
//   - https://dev.azure.com/org/project/_git/repo
//   - https://org.visualstudio.com/project/_git/repo
//
// We support both forms.
var providerSuffixes = []string{"dev.azure.com", "visualstudio.com"}

var registration = gitprovider.Registration{
	Predicate: func(repoURL string) bool {
		u, err := url.Parse(repoURL)
		if err != nil {
			return false
		}
		return slices.ContainsFunc(providerSuffixes, func(suffix string) bool {
			return strings.HasSuffix(u.Host, suffix)
		})
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

type provider struct {
	org        string
	project    string
	repo       string
	connection *azuredevops.Connection
}

// NewProvider returns an Azure DevOps-based implementation of gitprovider.Interface.
func NewProvider(
	repoURL string,
	opts *gitprovider.Options,
) (gitprovider.Interface, error) {
	if opts == nil || opts.Token == "" {
		return nil, fmt.Errorf("options are required for Azure DevOps provider")
	}
	org, project, repo, err := parseRepoURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("error creating Azure DevOps provider: %w", err)
	}
	organizationUrl := fmt.Sprintf("https://dev.azure.com/%s", org)
	connection := azuredevops.NewPatConnection(organizationUrl, opts.Token)

	return &provider{
		org:        org,
		project:    project,
		repo:       repo,
		connection: connection,
	}, nil
}

// CreatePullRequest implements gitprovider.Interface.
func (p *provider) CreatePullRequest(
	ctx context.Context,
	opts *gitprovider.CreatePullRequestOpts,
) (*gitprovider.PullRequest, error) {
	gitClient, err := adogit.NewClient(ctx, p.connection)
	if err != nil {
		return nil, err
	}
	repository, err := gitClient.GetRepository(ctx, adogit.GetRepositoryArgs{
		Project:      &p.project,
		RepositoryId: &p.repo,
	})
	if err != nil {
		return nil, err
	}
	repoID := ptr.To(repository.Id.String())
	sourceRefName := ptr.To(fmt.Sprintf("refs/heads/%s", opts.Head))
	targetRefName := ptr.To(fmt.Sprintf("refs/heads/%s", opts.Base))
	adoPR, err := gitClient.CreatePullRequest(ctx, adogit.CreatePullRequestArgs{
		Project:      &p.project,
		RepositoryId: repoID,
		GitPullRequestToCreate: &adogit.GitPullRequest{
			Title:         &opts.Title,
			Description:   &opts.Description,
			SourceRefName: sourceRefName,
			TargetRefName: targetRefName,
		},
	})
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
	gitClient, err := adogit.NewClient(ctx, p.connection)
	if err != nil {
		return nil, err
	}
	adoPR, err := gitClient.GetPullRequest(ctx, adogit.GetPullRequestArgs{
		Project:       &p.project,
		RepositoryId:  &p.repo,
		PullRequestId: ptr.To(int(id)),
	})
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
	gitClient, err := adogit.NewClient(ctx, p.connection)
	if err != nil {
		return nil, err
	}
	adoPRs, err := gitClient.GetPullRequests(ctx, adogit.GetPullRequestsArgs{
		Project:      &p.project,
		RepositoryId: &p.repo,
		SearchCriteria: &adogit.GitPullRequestSearchCriteria{
			Status:        ptr.To(mapADOPrState(opts.State)),
			SourceRefName: ptr.To(opts.HeadBranch),
			TargetRefName: ptr.To(opts.BaseBranch),
		},
	})
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
	return &gitprovider.PullRequest{
		Number:         int64(ptr.Deref(pr.PullRequestId, 0)),
		URL:            ptr.Deref(pr.Url, ""),
		Open:           ptr.Deref(pr.Status, "notSet") == "active",
		Merged:         ptr.Deref(pr.Status, "notSet") == "completed",
		MergeCommitSHA: ptr.Deref(mergeCommit.CommitId, ""),
		Object:         pr,
		HeadSHA:        ptr.Deref(pr.LastMergeSourceCommit.CommitId, ""),
	}, nil
}

func parseRepoURL(repoURL string) (string, string, string, error) {
	u, err := url.Parse(git.NormalizeURL(repoURL))
	if err != nil {
		return "", "", "", fmt.Errorf("error parsing Azure DevOps repository URL %q: %w", repoURL, err)
	}
	if u.Host == "dev.azure.com" {
		return parseModernRepoURL(u)
	} else if strings.HasSuffix(u.Host, ".visualstudio.com") {
		return parseLegacyRepoURL(u)
	}
	return "", "", "", fmt.Errorf("unsupported host %q", u.Host)
}

// parseModernRepoURL parses a modern Azure DevOps repository URL. example: https://dev.azure.com/org/project/_git/repo
func parseModernRepoURL(u *url.URL) (string, string, string, error) {
	parts := strings.Split(u.Path, "/")
	if len(parts) != 5 {
		return "", "", "", fmt.Errorf("could not extract repository organization, project, and name from URL %q", u)
	}
	return parts[1], parts[2], parts[4], nil
}

// parseLegacyRepoURL parses a legacy Azure DevOps repository URL. example: https://org.visualstudio.com/project/_git/repo
func parseLegacyRepoURL(u *url.URL) (string, string, string, error) {
	organization := strings.TrimSuffix(u.Host, ".visualstudio.com")
	parts := strings.Split(u.Path, "/")
	if len(parts) != 4 {
		return "", "", "", fmt.Errorf("could not extract repository organization, project, and name from URL %q", u)
	}
	return organization, parts[1], parts[3], nil
}

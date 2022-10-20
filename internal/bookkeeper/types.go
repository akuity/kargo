package bookkeeper

import (
	"github.com/akuityio/k8sta/internal/git"
)

// ActionTaken indicates what action, if any was taken in response to a
// RenderRequest.
type ActionTaken string

const (
	// ActionTakenPushedDirectly represents the case where Bookkeeper responded
	// to a RenderRequest by pushing a new commit directly to the target branch.
	ActionTakenPushedDirectly ActionTaken = "PUSHED_DIRECTLY"
	// ActionTakenOpenedPR represents the case where Bookkeeper responded to a
	// RenderRequest by opening a new pull request against the target branch.
	ActionTakenOpenedPR ActionTaken = "OPENED_PR"
	// ActionTakenNone represents the case where Bookkeeper responded
	// to a RenderRequest by, effectively, doing nothing. This occurs in cases
	// where the fully rendered configuration that would have been written to the
	// target branch does not differ from what is already present at the head of
	// that branch.
	ActionTakenNone ActionTaken = "NONE"
)

// RenderRequest is a request for Bookkeeper to render some environment-specific
// configuration from the default branch of the repository specified by RepoURL
// into plain YAML in an environment-specific branch.
type RenderRequest struct {
	// RepoURL is the URL of a remote GitOps repository.
	RepoURL string `json:"repoURL,omitempty"`
	// RepoCreds encapsulates read/write credentials for the remote GitOps
	// repository referenced by the RepoURL field.
	RepoCreds git.RepoCredentials `json:"repoCreds,omitempty"`
	// Commit specifies a precise commit to render configuration from. When this
	// is omitted, the request is assumed to be one to render from the head of the
	// default branch.
	Commit string `json:"commit,omitempty"`
	// TargetBranch is the name of an environment-specific branch in the GitOps
	// repository referenced by the RepoURL field into which plain YAML should be
	// rendered. The path to environment-specific configuration in the
	// repository's default branch is inferred to be equal to this value.
	TargetBranch string `json:"targetBranch,omitempty"`
	// Images specifies images to incorporate into environment-specific
	// configuration.
	Images []string `json:"images,omitempty"`
	// OpenPR specifies whether to open a PR against TargetBranch (true) instead
	// of directly committing directly to it (false).
	OpenPR bool `json:"openPR,omitempty"`
}

// Response encapsulates details of a successful rendering of some some
// environment-specific configuration into plain YAML in an environment-specific
// branch.
type Response struct {
	ActionTaken ActionTaken `json:"actionTaken,omitempty"`
	// CommitID is the ID (sha) of the commit to the environment-specific branch
	// containing the rendered configuration. This is only set when the OpenPR
	// field of the corresponding RenderRequest was false.
	CommitID string `json:"commitID,omitempty"`
	// PullRequestURL is a URL for a pull request containing the rendered
	// configuration. This is only set when the OpenPR field of the corresponding
	// RenderRequest was true.
	PullRequestURL string `json:"pullRequestURL,omitempty"`
}

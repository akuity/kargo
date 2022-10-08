package bookkeeper

import (
	api "github.com/akuityio/k8sta/api/v1alpha1"
	"github.com/akuityio/k8sta/internal/git"
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
	// TargetBranch is the name of an environment-specific branch in the GitOps
	// repository referenced by the RepoURL field into which plain YAML should be
	// rendered. The path to environment-specific configuration in the
	// repository's default branch is inferred to be equal to this value.
	TargetBranch string `json:"targetBranch,omitempty"`
	// ConfigManagement encapsulates details of which configuration management
	// tool is to be used and, if applicable, configuration options for the
	// selected tool.
	ConfigManagement api.ConfigManagementConfig `json:"configManagement,omitempty"` // nolint: lll
	// Images specifies images to incorporate into environment-specific
	// configuration.
	Images []string `json:"images,omitempty"`
}

// Response encapsulates details of a successful rendering of some some
// environment-specific configuration into plain YAML in an environment-specific
// branch.
type Response struct {
	// CommitID is the ID (sha) of the commit to the environment-specific branch
	// containing the rendered configuration.
	CommitID string `json:"commitID,omitempty"`
}

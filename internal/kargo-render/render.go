package render

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/akuity/kargo/internal/controller/git"
	libExec "github.com/akuity/kargo/internal/exec"
)

// ActionTaken indicates what action, if any was taken in response to a
// RenderRequest.
type ActionTaken string

const (
	// ActionTakenNone represents the case where Kargo Render responded
	// to a RenderRequest by, effectively, doing nothing. This occurs in cases
	// where the fully rendered manifests that would have been written to the
	// target branch do not differ from what is already present at the head of
	// that branch.
	ActionTakenNone ActionTaken = "NONE"
	// ActionTakenOpenedPR represents the case where Kargo Render responded to a
	// RenderRequest by opening a new pull request against the target branch.
	ActionTakenOpenedPR ActionTaken = "OPENED_PR"
	// ActionTakenPushedDirectly represents the case where Kargo Render responded
	// to a RenderRequest by pushing a new commit directly to the target branch.
	ActionTakenPushedDirectly ActionTaken = "PUSHED_DIRECTLY"
	// ActionTakenUpdatedPR represents the case where Kargo Render responded to a
	// RenderRequest by updating an existing PR.
	ActionTakenUpdatedPR ActionTaken = "UPDATED_PR"
)

// Request is a request for Kargo Render to render environment-specific
// manifests from input in the  default branch of the repository specified by
// RepoURL.
type Request struct {
	// RepoURL is the URL of a remote GitOps repository.
	RepoURL string `json:"repoURL,omitempty"`
	// RepoCreds encapsulates read/write credentials for the remote GitOps
	// repository referenced by the RepoURL field.
	RepoCreds git.RepoCredentials `json:"repoCreds,omitempty"`
	// Ref specifies either a branch or a precise commit to render manifests from.
	// When this is omitted, the request is assumed to be one to render from the
	// head of the default branch.
	Ref string `json:"ref,omitempty"`
	// TargetBranch is the name of an environment-specific branch in the GitOps
	// repository referenced by the RepoURL field into which plain YAML should be
	// rendered.
	TargetBranch string `json:"targetBranch,omitempty"`
	// Images specifies images to incorporate into environment-specific
	// manifests.
	Images []string `json:"images,omitempty"`
}

// Response encapsulates details of a successful rendering of some
// environment-specific manifests into an environment-specific branch.
type Response struct {
	ActionTaken ActionTaken `json:"actionTaken,omitempty"`
	// CommitID is the ID (sha) of the commit to the environment-specific branch
	// containing the rendered manifests. This is only set when the OpenPR field
	// of the corresponding RenderRequest was false.
	CommitID string `json:"commitID,omitempty"`
	// PullRequestURL is a URL for a pull request containing the rendered
	// manifests. This is only set when the OpenPR field of the corresponding
	// RenderRequest was true.
	PullRequestURL string `json:"pullRequestURL,omitempty"`
}

// Execute a `kargo-render render` command and return the response.
func RenderManifests(req Request) (Response, error) { // nolint: revive
	res := Response{}
	resBytes, err := libExec.Exec(buildRenderCmd(req))
	if err != nil {
		return res, errors.Wrap(err, "error rendering manifests")
	}
	err = json.Unmarshal(resBytes, &res)
	return res, errors.Wrap(err, "error unmarshalling response")
}

func buildRenderCmd(req Request) *exec.Cmd {
	cmdTokens := []string{
		"kargo-render",
		"render",
		"--repo",
		req.RepoURL,
		"--ref",
		req.Ref,
		"--target-branch",
		req.TargetBranch,
		"--repo-username",
		req.RepoCreds.Username,
		"--output",
		"json",
	}
	for _, image := range req.Images {
		cmdTokens = append(cmdTokens, "--image", image)
	}
	cmd := exec.Command(cmdTokens[0], cmdTokens[1:]...) // nolint: gosec
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("KARGO_RENDER_REPO_PASSWORD=%s", req.RepoCreds.Password),
	)
	return cmd
}

package render

import (
	"encoding/json"
	"fmt"
	"os/exec"

	libExec "github.com/akuity/kargo/internal/exec"
)

// ActionTaken indicates what action, if any was taken in response to a
// RenderRequest.
type ActionTaken string

const (
	// ActionTakenWroteToLocalPath represents the case where Kargo Render
	// responded to a RenderRequest by writing the rendered manifests to a local
	// path.
	ActionTakenWroteToLocalPath ActionTaken = "WROTE_TO_LOCAL_PATH"
)

// Request is a request for Kargo Render to render environment-specific
// manifests from input in the  default branch of the repository specified by
// RepoURL.
type Request struct {
	// TargetBranch is the name of an environment-specific branch in the GitOps
	// repository referenced by the RepoURL field into which plain YAML should be
	// rendered.
	TargetBranch string `json:"targetBranch,omitempty"`
	// LocalInPath specifies a path to the repository's working tree with the
	// desired source commit already checked out.
	LocalInPath string `json:"localInPath,omitempty"`
	// LocalOutPath specifies a path where the rendered manifests should be
	// written. The specified path must NOT exist already.
	LocalOutPath string `json:"localOutPath,omitempty"`
	// Images specifies images to incorporate into environment-specific
	// manifests.
	Images []string `json:"images,omitempty"`
}

// Response encapsulates details of a successful rendering of some
// environment-specific manifests into an environment-specific branch.
type Response struct {
	ActionTaken ActionTaken `json:"actionTaken,omitempty"`
	// LocalPath is the path to the directory where the rendered manifests
	// were written.
	LocalPath string `json:"localPath,omitempty"`
}

// Execute a `kargo-render` command and return the response.
func RenderManifests(req Request) error { // nolint: revive
	cmdTokens := []string{
		"kargo-render",
		"--target-branch",
		req.TargetBranch,
		"--local-in-path",
		req.LocalInPath,
		"--local-out-path",
		req.LocalOutPath,
		"--output",
		"json",
	}
	for _, image := range req.Images {
		cmdTokens = append(cmdTokens, "--image", image)
	}
	cmd := exec.Command(cmdTokens[0], cmdTokens[1:]...) // nolint: gosec

	res := Response{}
	resBytes, err := libExec.Exec(cmd)
	if err != nil {
		return fmt.Errorf("error rendering manifests: %w", err)
	}
	if err = json.Unmarshal(resBytes, &res); err != nil {
		return fmt.Errorf("error unmarshaling response: %w", err)
	}

	// TODO: Make some assertions about the response. It should have written the
	// rendered manifests to a directory. If anything other than that happened,
	// something went very wrong.

	return nil
}
